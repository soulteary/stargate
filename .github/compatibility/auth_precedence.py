"""Version-specific, normal HTTP authentication precedence comparison.

Only published binaries are used. This profile never edits opaque session data,
reconstructs credentials, or resets provider quotas/replay state.
"""
import json
import time
import urllib.parse

PASSWORD = 'synthetic-unused-password'
TRUSTED_SECRET = 'synthetic-compat-trusted-header-secret-123456'
CASES = (
    ('password outage', 'password', 'outage'),
    ('session outage', 'session', 'outage'),
    ('trusted outage', 'trusted', 'outage'),
    ('password deleted', 'password', 'deleted'),
    ('trusted deleted', 'trusted', 'deleted'),
    ('session deleted', 'session', 'deleted'),
    ('invalid password active session', 'invalid-password', 'healthy'),
    ('invalid password outage', 'invalid-password', 'outage'),
    ('protected password', 'password', 'protected'),
    ('protected trusted', 'trusted', 'protected'),
)


class OTPPacer:
    """Respect Herald's unchanged five-per-minute fixed IP window.

    Pause beyond a complete window initially and after each four successful
    sends. The margin leaves one slot unused; no Redis counters are modified.
    """
    WINDOW_SECONDS = 61
    BATCH_SIZE = 4

    def __init__(self):
        self.ready_at = time.monotonic() + self.WINDOW_SECONDS
        self.sent_in_batch = 0

    def before_send(self, harness):
        while time.monotonic() < self.ready_at:
            harness.budget()
            time.sleep(.2)

    def sent(self):
        self.sent_in_batch += 1
        if self.sent_in_batch == self.BATCH_SIZE:
            # Start the wait after the response, so even a delayed create call's
            # actual Redis acceptance precedes the full quiet interval.
            self.ready_at = time.monotonic() + self.WINDOW_SECONDS
            self.sent_in_batch = 0


def validate_profile(manifest):
    if 'auth_precedence' not in manifest:
        return
    if manifest['auth_precedence'] is not True:
        raise ValueError('auth_precedence must be the boolean true')
    if manifest['old']['gateway']['version'] != '1.3.4':
        raise ValueError('auth_precedence requires the known legacy gateway 1.3.4')
    if manifest['mode'] == 'upgrade':
        if manifest['new']['gateway']['version'] != '1.3.5':
            raise ValueError('auth_precedence requires the known fixed gateway 1.3.5')
        if any(manifest['old'][service] != manifest['new'][service] for service in ('warden', 'herald', 'totp')):
            raise ValueError('auth_precedence requires unchanged upstream identities')


def users():
    result = []
    for version in ('1.3.4', '1.3.5'):
        for index in range(len(CASES)):
            for identity in ('a', 'b'):
                label = f'precedence-{version.replace(".", "-")}-{index}-{identity}'
                result.append({'user_id': label, 'mail': label + '@example.test',
                               'status': 'active', 'scope': ['profile-' + identity],
                               'role': 'profile-' + identity})
    return result


def expected_status(version, method, condition):
    if version not in ('1.3.4', '1.3.5'):
        raise ValueError('no authentication precedence expectation for this version')
    if condition == 'protected':
        return 403
    if condition == 'healthy':
        return 200
    if condition == 'outage':
        return 200 if version == '1.3.5' and method == 'password' else 503
    if condition == 'deleted':
        return 200 if version == '1.3.5' and method in ('password', 'trusted') else 401
    raise ValueError('unknown authentication precedence condition')


def assert_identity(response, status, method, user_a, user_b):
    """Success must identify the selected checker, never stale cookie claims."""
    if response[0] != status:
        raise AssertionError(f'authentication precedence HTTP {response[0]} expected {status}')
    headers = response[2]
    if status != 200:
        if any(headers.get(name) for name in ('x-auth-user', 'x-auth-scopes', 'x-auth-role', 'x-forwarded-user')):
            raise AssertionError('failed authentication published an identity')
        return
    if method == 'password':
        if any(headers.get(name) for name in ('x-auth-user', 'x-auth-scopes', 'x-auth-role')):
            raise AssertionError('password checker published stale directory claims')
        if headers.get('x-forwarded-user') != 'authenticated':
            raise AssertionError('password checker did not publish its independent authentication result')
    else:
        selected = user_b if method == 'trusted' else user_a
        if headers.get('x-auth-user') != selected['user_id'] or headers.get('x-forwarded-user') != selected['user_id']:
            raise AssertionError('authentication precedence selected the wrong identity')
        if headers.get('x-auth-scopes') != ','.join(selected['scope']):
            raise AssertionError('authentication precedence published the wrong scopes')
        if headers.get('x-auth-role') != selected['role']:
            raise AssertionError('authentication precedence published the wrong role')


def run(harness, manifest, paths, templates, original_env):
    validate_profile(manifest)
    if manifest.get('auth_precedence') is not True:
        return
    saved_browser = harness.browser_cookies
    source = harness.RUNTIME / 'users.json'
    original_source = source.read_bytes()
    env = {name: dict(values) for name, values in original_env.items()}
    env['gateway'].update(AUTH_REFRESH_ENABLED='true', AUTH_REFRESH_INTERVAL='1s',
                          WARDEN_SOURCE_MAX_AGE='1h', WARDEN_SOURCE_CLOCK_SKEW='0s',
                          PASSWORD_HEADER_AUTH_ENABLED='true', HEADER_AUTH_ENABLED='true',
                          HEADER_AUTH_SHARED_SECRET=TRUSTED_SECRET, TRUSTED_PROXIES='127.0.0.1',
                          STEP_UP_ENABLED='true', STEP_UP_PATHS='/admin')
    phases = ['old'] + (['new'] if manifest['mode'] == 'upgrade' else [])
    pacer = OTPPacer()
    harness.result['auth_precedence_pacing'] = {
        'unchanged_ip_limit': 5, 'fixed_window_seconds': 60, 'batch_size': 4,
        'quiet_interval_seconds': 61, 'maximum_planned_pacing_seconds':
            ((len(phases) * len(CASES) + 3) // 4) * 61,
        'counter_reset': False, 'retry_429': False}
    try:
        for phase in phases:
            version = manifest[phase]['gateway']['version']
            harness.stop('gateway')
            harness.start('gateway', paths[phase]['gateway'], env['gateway'], templates[phase])
            for index, (label, method, condition) in enumerate(CASES):
                pair = [u for u in users() if u['user_id'].startswith(
                    f'precedence-{version.replace(".", "-")}-{index}-')]
                user_a, user_b = pair
                harness.browser_cookies = {}
                pacer.before_send(harness)
                challenge, code = harness.send(user_a['mail'])
                pacer.sent()
                harness.login(user_a['mail'], challenge, code, auth_headers={'X-Forwarded-Uri': '/profile'})
                if method == 'trusted':
                    with harness.isolated_browser():
                        control = harness.request(harness.gateway, '/_auth', opener=True, extra_headers={
                            'X-Forwarded-Uri': '/profile', 'X-Stargate-Header-Auth': TRUSTED_SECRET,
                            'X-User-Mail': user_b['mail']})
                        assert_identity(control, 200, 'trusted', user_a, user_b)
                    harness.events.append({'check': 'trusted identity positive control', 'gateway_version': version, 'status': 200})
                if condition == 'protected':
                    harness.expect('legitimate session step-up', harness.request(harness.gateway, '/_step_up',
                        urllib.parse.urlencode({'password': PASSWORD}), opener=True,
                        extra_headers={'Origin': harness.gateway}), 302)
                    harness.expect('verified session accesses protected path', harness.request(harness.gateway,
                        '/_auth', opener=True, extra_headers={'X-Forwarded-Uri': '/admin/settings'}), 200)
                # Refresh timestamps use Unix seconds and an inclusive interval.
                # Wait beyond two boundaries; never rewrite session metadata.
                limit = time.monotonic() + 2.1
                while time.monotonic() < limit:
                    harness.budget()
                    time.sleep(.1)
                try:
                    if condition == 'outage':
                        harness.stop('warden')
                        with harness.isolated_browser():
                            control = harness.request(harness.gateway, '/_auth', opener=True, extra_headers={
                                'X-Forwarded-Uri': '/profile', 'Stargate-Password': PASSWORD})
                            assert_identity(control, 200, 'password', user_a, user_b)
                        harness.events.append({'check': 'password without cookie during outage positive control',
                                               'gateway_version': version, 'status': 200})
                    elif condition == 'deleted':
                        changed = [u for u in json.loads(original_source) if u['user_id'] != user_a['user_id']]
                        source.write_text(json.dumps(changed))
                        harness.stop('warden')
                        harness.start('warden', paths['old']['warden'], env['warden'])
                        harness.expect('deleted source identity absent', harness.request(harness.warden,
                            '/user?mail=' + urllib.parse.quote(user_a['mail']), key=harness.key_w), 404)
                        body = harness.expect('independent source identity remains active', harness.request(harness.warden,
                            '/user?mail=' + urllib.parse.quote(user_b['mail']), key=harness.key_w), 200)
                        if body.get('user_id') != user_b['user_id']:
                            raise AssertionError('directory control returned the wrong independent identity')
                    headers = {'X-Forwarded-Uri': '/profile'}
                    if method in ('password', 'invalid-password'):
                        headers['Stargate-Password'] = PASSWORD if method == 'password' else 'synthetic-wrong-password'
                    elif method == 'trusted':
                        headers.update({'X-Stargate-Header-Auth': TRUSTED_SECRET, 'X-User-Mail': user_b['mail']})
                    if condition == 'protected':
                        headers['X-Forwarded-Uri'] = '/admin/settings'
                    response = harness.request(harness.gateway, '/_auth', opener=True, extra_headers=headers)
                    status = expected_status(version, method, condition)
                    assert_identity(response, status, method, user_a, user_b)
                    harness.events.append({'check': 'authentication precedence comparison', 'gateway_version': version,
                        'gateway_source': manifest[phase]['gateway']['source'], 'case': label, 'status': status,
                        'expected_behavior': 'known legacy comparison' if version == '1.3.4' else 'fixed version regression',
                        'identity_checked': status == 200, 'session_metadata_mutated': False})
                    harness.save()
                finally:
                    source.write_bytes(original_source)
                    if 'warden' in harness.processes:
                        harness.stop('warden')
                    harness.start('warden', paths['old']['warden'], env['warden'])
    finally:
        source.write_bytes(original_source)
        harness.browser_cookies = saved_browser
        if 'gateway' in harness.processes:
            harness.stop('gateway')
        if 'warden' in harness.processes:
            harness.stop('warden')
        harness.start('warden', paths['old']['warden'], original_env['warden'])
        harness.start('gateway', paths['old']['gateway'], original_env['gateway'], templates['old'])
