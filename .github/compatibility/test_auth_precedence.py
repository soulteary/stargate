"""Pure preflight and assertion counterexamples, without service execution."""
import copy
from contextlib import contextmanager
import unittest
import json
import tempfile
from pathlib import Path
from types import SimpleNamespace
from unittest import mock
import auth_precedence as profile
import test_release_compatibility as original


class AuthPrecedenceTests(unittest.TestCase):
    def manifest(self):
        value = json.loads((original.ROOT / 'manifests/baseline.json').read_text())
        value['old']['gateway']['version'] = '1.3.4'
        value.update(mode='upgrade', new=copy.deepcopy(value['old']), auth_precedence=True)
        value['new']['gateway']['version'] = '1.3.5'
        value['new']['gateway']['source'] = 'f' * 40
        return value

    def test_only_changed_service_is_a_version_transition(self):
        value = self.manifest()
        self.assertEqual(original.module.changed_services(value), ['gateway'])
        del value['auth_precedence']
        value['new']['gateway'] = copy.deepcopy(value['old']['gateway'])
        value['new']['herald']['version'] = '1.4.4'
        self.assertEqual(original.module.changed_services(value), ['herald'])
        self.assertTrue(original.module.rotate_after_stage('upgrade', 'herald', ['herald']))
        self.assertFalse(original.module.rotate_after_stage('upgrade', 'gateway', ['herald']))
        self.assertFalse(original.module.rotate_after_stage('rollback', 'herald', ['herald']))
        self.assertFalse(original.module.rotate_after_stage('baseline', None, []))

    def test_same_version_conflicting_identity_rejected(self):
        value = self.manifest()
        del value['auth_precedence']
        value['new']['gateway']['version'] = '1.3.4'
        with self.assertRaisesRegex(ValueError, 'conflicting identities'):
            original.module.validate_manifest(value)

    def test_optional_profile_requires_exact_bool_and_known_versions(self):
        value = self.manifest()
        original.module.validate_manifest(value)
        for flag in (False, 1, 'true', None):
            invalid = copy.deepcopy(value)
            invalid['auth_precedence'] = flag
            with self.assertRaisesRegex(ValueError, 'boolean true'):
                original.module.validate_manifest(invalid)
        for phase, version in [('old', '1.3.3'), ('new', '1.3.6')]:
            invalid = copy.deepcopy(value)
            invalid[phase]['gateway']['version'] = version
            with self.assertRaisesRegex(ValueError, 'known'):
                original.module.validate_manifest(invalid)
        value['new']['totp']['version'] = '2.0.4'
        with self.assertRaisesRegex(ValueError, 'unchanged upstream'):
            original.module.validate_manifest(value)

    def test_fixed_expectations_are_version_specific_not_phase_specific(self):
        self.assertEqual(profile.expected_status('1.3.4', 'password', 'outage'), 503)
        self.assertEqual(profile.expected_status('1.3.5', 'password', 'outage'), 200)
        for version in ('1.3.4', '1.3.5'):
            self.assertEqual(profile.expected_status(version, 'trusted', 'outage'), 503)
            self.assertEqual(profile.expected_status(version, 'session', 'deleted'), 401)
            self.assertEqual(profile.expected_status(version, 'password', 'protected'), 403)
            self.assertEqual(profile.expected_status(version, 'invalid-password', 'healthy'), 200)
        with self.assertRaises(ValueError):
            profile.expected_status('new', 'password', 'outage')

    def test_success_cannot_publish_wrong_identity_or_stale_password_claims(self):
        a, b = profile.users()[:2]
        for method, headers in [('trusted', {'x-auth-user': a['user_id']}),
                                ('password', {'x-auth-user': a['user_id']}),
                                ('password', {'x-auth-scopes': 'profile-a'}),
                                ('session', {'x-auth-user': b['user_id']})]:
            with self.subTest(method=method, headers=headers), self.assertRaises(AssertionError):
                profile.assert_identity((200, {}, headers), 200, method, a, b)
        profile.assert_identity((200, {}, {'x-forwarded-user': 'authenticated'}), 200, 'password', a, b)
        profile.assert_identity((200, {}, {'x-auth-user': b['user_id'], 'x-forwarded-user': b['user_id'], 'x-auth-scopes': 'profile-b', 'x-auth-role': 'profile-b'}), 200, 'trusted', a, b)
        with self.assertRaises(AssertionError):
            profile.assert_identity((401, {}, {'x-auth-user': a['user_id']}), 401, 'session', a, b)
        with self.assertRaisesRegex(AssertionError, 'wrong role'):
            profile.assert_identity((200, {}, {'x-auth-user': b['user_id'], 'x-forwarded-user': b['user_id'], 'x-auth-scopes': 'profile-b', 'x-auth-role': 'profile-a'}), 200, 'trusted', a, b)
        with self.assertRaisesRegex(AssertionError, 'published an identity'):
            profile.assert_identity((503, {}, {'x-auth-scopes': 'profile-a', 'x-auth-role': 'profile-a'}), 503, 'session', a, b)
        with self.assertRaisesRegex(AssertionError, 'published an identity'):
            profile.assert_identity((403, {}, {'x-forwarded-user': a['user_id']}), 403, 'trusted', a, b)

    def test_each_case_has_independent_real_login_recipient(self):
        values = profile.users()
        self.assertEqual(len(values), len({user['mail'] for user in values}))
        self.assertEqual(len(values), 2 * 2 * len(profile.CASES))
        self.assertTrue(all(user['status'] == 'active' for user in values))

    def test_profile_login_and_unprotected_request_have_explicit_original_uri(self):
        with mock.patch.object(original.module, 'browser_cookies', {'session': 'r1.example.synthetic'}), mock.patch.object(original.module, 'request', side_effect=[(200, {'success': True}, {}), (200, {}, {'x-auth-user': 'chain-user-c'})]) as request, mock.patch.object(original.module, 'save'):
            original.module.login('chain-c@example.test', 'challenge', 'code', auth_headers={'X-Forwarded-Uri': '/profile'})
            self.assertEqual(request.call_args_list[-1], mock.call(original.module.gateway, '/_auth', opener=True, extra_headers={'X-Forwarded-Uri': '/profile'}))

    def test_failed_actual_response_restores_browser_source_and_process_configuration(self):
        manifest = self.manifest()
        with tempfile.TemporaryDirectory() as directory:
            runtime = Path(directory)
            initial_source = json.dumps(profile.users()).encode()
            (runtime / 'users.json').write_bytes(initial_source)
            previous_browser = {'pending': 'preserved'}
            starts = []
            env = {name: {'original': name} for name in ('warden', 'gateway', 'herald', 'totp')}
            paths = {phase: {name: name + '-' + phase for name in env} for phase in ('old', 'new')}
            harness = SimpleNamespace(RUNTIME=runtime, browser_cookies=previous_browser,
                processes={name: object() for name in env}, gateway='http://127.0.0.1:38080',
                warden='http://127.0.0.1:38081', key_w='synthetic', events=[],
                send=lambda subject: ('challenge', 'synthetic-code'),
                login=lambda *args, **kwargs: None, budget=lambda: 10, save=lambda: None)
            def stop(name):
                harness.processes.pop(name)
            def start(name, path, config, templates=None):
                self.assertNotIn(name, harness.processes)
                harness.processes[name] = object()
                starts.append((name, path, dict(config)))
            harness.stop, harness.start = stop, start
            # Known legacy password/outage requires503, so200mustabort immediately.
            @contextmanager
            def isolated_browser():
                previous = harness.browser_cookies
                harness.browser_cookies = {}
                try:
                    yield
                finally:
                    harness.browser_cookies = previous
            harness.isolated_browser = isolated_browser
            harness.request = lambda *args, **kwargs: (200, {}, {'x-forwarded-user': 'authenticated'})
            with mock.patch.object(profile.time, 'monotonic', side_effect=[0, 3]), self.assertRaisesRegex(AssertionError, 'expected 503'):
                profile.run(harness, manifest, paths, {'old': 'old-templates', 'new': 'new-templates'}, env)
            self.assertIs(harness.browser_cookies, previous_browser)
            self.assertEqual((runtime / 'users.json').read_bytes(), initial_source)
            self.assertEqual(starts[-2:], [('warden', 'warden-old', env['warden']), ('gateway', 'gateway-old', env['gateway'])])
            self.assertEqual(set(harness.processes), set(env))


if __name__ == '__main__':
    unittest.main()
