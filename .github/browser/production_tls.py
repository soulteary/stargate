"""Ephemeral hosted-runner HTTPS acceptance; no production endpoints or certificate bypass."""
from contextlib import contextmanager
import hashlib
import importlib.util
import hmac
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import smtplib
import ssl
import subprocess
import sys
import time
import urllib.error
import urllib.request
from urllib.parse import urlparse

ROOT = Path.cwd()
AUTH_HOST = 'auth.browser.example.com'
APP_HOST = 'app.browser.example.com'
AUTH = 'https://' + AUTH_HOST + ':19443'
APP = 'https://' + APP_HOST + ':19443'
MAILPIT = 'axllent/mailpit:v1.33.1@sha256:aef2aacc11f075ec0fb6874275ee4a421e688e11a96ebb296134d1735c7761ce'
RESULT = {'status': 'running', 'checks': [], 'scope': 'Real production Compose TLS/HMAC and HTTPS Chromium with synthetic authenticated SMTP capture; no external delivery or device coverage'}
STATE = None
PROJECT = None
DEADLINE = None
CA_SYSTEM_PATH = None
CA_SYSTEM_INSTALLED = False


def hosted_runner(values, platform):
    return values.get('GITHUB_ACTIONS') == 'true' and values.get('RUNNER_OS') == 'Linux' and values.get('RUNNER_ENVIRONMENT') == 'github-hosted' and platform.startswith('linux')


def url_matches(value, expected=APP):
    try:
        actual, target = urlparse(str(value)), urlparse(expected)
        return actual.scheme == target.scheme == 'https' and actual.username is None and actual.password is None and actual.hostname == target.hostname and (actual.port or 443) == (target.port or 443) and (actual.path or '/') == (target.path or '/')
    except ValueError:
        return False


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, response, code, message, headers, url):
        return None


def stage(name):
    RESULT['stage'] = name
    save()


def safe_failure(error):
    if isinstance(error, urllib.error.HTTPError):
        return {'reason': 'http_error', 'status': error.code if 100 <= error.code <= 599 else None}
    if isinstance(error, subprocess.TimeoutExpired):
        return {'reason': 'command_timeout'}
    if isinstance(error, ssl.SSLError):
        return {'reason': 'tls_error'}
    if isinstance(error, smtplib.SMTPAuthenticationError):
        return {'reason': 'smtp_authentication_failed'}
    if isinstance(error, AssertionError):
        return {'reason': 'assertion_failed'}
    text = str(error)
    for marker, reason in [('ERR_CERT_AUTHORITY_INVALID', 'certificate_untrusted'), ('ERR_NAME_NOT_RESOLVED', 'dns_failed'),
                           ('ERR_CONNECTION_REFUSED', 'connection_refused'), ('Timeout', 'browser_timeout')]:
        if marker in text:
            return {'reason': reason}
    return {'reason': 'unexpected_failure'}


def safe_http_reason(value):
    allowed = {'invalid', 'unauthorized', 'rate_limited', 'resend_cooldown', 'proxy_failed', 'provider_failed',
               'service_unavailable', 'user_not_in_list', 'context_mismatch', 'channel_disabled'}
    return value if isinstance(value, str) and value in allowed else 'unknown'


def save():
    Path('production-tls-results.json').write_text(json.dumps(RESULT, indent=2) + '\n')


def check(name, condition):
    if not condition:
        RESULT['failed_check'] = name
        save()
        raise AssertionError(name)
    RESULT['checks'].append({'name': name, 'status': 'passed'})
    save()


def command(args, cwd=None, env=None, timeout=90, cleanup=False):
    if DEADLINE and not cleanup:
        timeout = min(timeout, max(1, int(DEADLINE - time.monotonic())))
    RESULT['last_command'] = Path(str(args[0])).name if Path(str(args[0])).name in {'docker', 'openssl', 'sudo', 'certutil', 'python3', 'git'} else 'other'
    process = subprocess.run([str(x) for x in args], cwd=cwd, env=env, capture_output=True, timeout=timeout)
    if process.returncode:
        raise RuntimeError('Command failed; raw output withheld because it may contain credentials')
    return process.stdout


def child_env(extra=None):
    allowed = {'PATH', 'HOME', 'LANG', 'LC_ALL', 'TMPDIR', 'TMP', 'TEMP', 'RUNNER_TEMP', 'LD_LIBRARY_PATH',
               'DOCKER_HOST', 'DOCKER_CONFIG', 'DOCKER_TLS_VERIFY', 'DOCKER_CERT_PATH', 'XDG_RUNTIME_DIR'}
    env = {k: v for k, v in os.environ.items() if k in allowed}
    env.update(extra or {})
    return env


@contextmanager
def private_browser_environment():
    original = dict(os.environ)
    os.environ.clear()
    os.environ.update({k: v for k, v in original.items() if k in {'PATH', 'HOME', 'LANG', 'LC_ALL', 'TMPDIR', 'TMP', 'TEMP', 'RUNNER_TEMP', 'LD_LIBRARY_PATH', 'XDG_RUNTIME_DIR'}})
    try:
        yield
    finally:
        os.environ.clear()
        os.environ.update(original)


def compose(*args, cleanup=False):
    return command(['docker', 'compose', '--project-name', PROJECT, *args], cwd=STATE,
                   env=child_env(), timeout=360, cleanup=cleanup)


def private_write(path, content):
    path.write_text(content)
    path.chmod(0o600)


def certificates(directory):
    helper = ROOT/'fullstack/examples/full-stack/certificates.py'
    spec = importlib.util.spec_from_file_location('smtp_certificates', helper)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    directory.mkdir(mode=0o700)
    module.generate(directory/'smtp')
    for name in ['smtp-cert.pem', 'smtp-key.pem']:
        shutil.copyfile(directory/'smtp'/name, directory/name)
        (directory/name).chmod(0o600 if name.endswith('key.pem') else 0o644)
    # A distinct web/TOTP CA makes the merged provider trust-bundle path meaningful.
    ca_key, ca = directory/'ca-key.pem', directory/'totp-ca.pem'
    command(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-sha256', '-days', '1',
             '-subj', '/CN=Ephemeral browser and TOTP acceptance CA', '-addext', 'basicConstraints=critical,CA:TRUE',
             '-addext', 'keyUsage=critical,keyCertSign,cRLSign', '-keyout', ca_key, '-out', ca])
    ca_key.chmod(0o600)
    ca.chmod(0o644)
    for stem, hosts in [('cert', [AUTH_HOST, APP_HOST]), ('totp-cert', ['totp-tls'])]:
        key_name = {'cert': 'key.pem', 'totp-cert': 'totp-key.pem', 'smtp-cert': 'smtp-key.pem'}[stem]
        key, csr, cert = directory/key_name, directory/(stem+'.csr'), directory/(stem+'.pem')
        extension = directory/(stem+'.ext')
        private_write(extension, 'basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature,keyEncipherment\nextendedKeyUsage=serverAuth\nsubjectAltName=' + ','.join('DNS:'+h for h in hosts) + '\n')
        command(['openssl', 'req', '-new', '-newkey', 'rsa:2048', '-nodes', '-subj', '/CN='+hosts[0], '-keyout', key, '-out', csr])
        key.chmod(0o600)
        command(['openssl', 'x509', '-req', '-in', csr, '-CA', ca, '-CAkey', ca_key, '-CAcreateserial',
                 '-days', '1', '-sha256', '-extfile', extension, '-out', cert])
        command(['openssl', 'verify', '-CAfile', ca, cert])
        cert.chmod(0o644)
    return ca


def read_generated_env(path):
    # The reviewed setup tool emits JSON-quoted dotenv values; never evaluate a shell file.
    return {line.split('=', 1)[0]: json.loads(line.split('=', 1)[1]).replace('$$', '$')
            for line in path.read_text().splitlines() if line and not line.startswith('#')}


def service_ip(name, network_suffix):
    container = compose('ps', '-q', name).decode().strip()
    networks = json.loads(command(['docker', 'inspect', '--format', '{{json .NetworkSettings.Networks}}', container]))
    matches = [value['IPAddress'] for network, value in networks.items() if network.endswith(network_suffix)]
    if len(matches) != 1:
        raise AssertionError('Ambiguous isolated service network')
    return matches[0]


def smtp_negative_control(ca, username):
    client = smtplib.SMTP(service_ip('mailpit', '_delivery'), 1025, timeout=10)
    try:
        client.ehlo()
        client._host = 'mailpit'  # Verify the certificate's DNS SAN while dialing its private Docker IP.
        client.starttls(context=ssl.create_default_context(cafile=str(ca)))
        client.ehlo()
        check('SMTP fixture STARTTLS certificate verified', client.sock.cipher() is not None)
        try:
            client.login(username, 'deliberately-wrong-synthetic-password')
        except smtplib.SMTPAuthenticationError:
            check('SMTP fixture rejects incorrect credentials', True)
        else:
            raise AssertionError('SMTP fixture accepted incorrect credentials')
    finally:
        client.close()


def signed_totp_status(secret):
    target = '/v1/totp/status?subject=production-user-001'
    timestamp, nonce = str(int(time.time())), secrets.token_hex(16)
    canonical = '\n'.join(['HERALD-HMAC-V3', 'GET', '/v1/totp/status', 'subject=production-user-001',
                           timestamp, nonce, 'stargate', 'stargate-otp',
                           hashlib.sha256(b'').hexdigest(), hashlib.sha256(b'').hexdigest()])
    signature = hmac.new(secret.encode(), canonical.encode(), hashlib.sha256).hexdigest()
    headers = {'X-Signature-Version': 'v3', 'X-Signature': signature, 'X-Timestamp': timestamp,
               'X-Nonce': nonce, 'X-Service': 'stargate', 'X-Key-Id': 'stargate-otp'}
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    url = 'http://' + service_ip('herald', '_backend') + ':8082' + target
    with opener.open(urllib.request.Request(url, headers=headers), timeout=15) as response:
        body = json.load(response)
        check('Production Herald proxies authenticated TOTP status over verified HTTPS', response.status == 200 and body.get('subject') == 'production-user-001' and body.get('totp_enabled') is False)
    headers['X-Nonce'] = secrets.token_hex(16)
    headers['X-Signature'] = '0' * 64
    try:
        opener.open(urllib.request.Request(url, headers=headers), timeout=15)
    except urllib.error.HTTPError as error:
        check('Invalid HMAC does not fall back', error.code == 401)
    else:
        raise AssertionError('Invalid HMAC accepted')


def mailbox_code():
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    limit = time.monotonic() + 30
    while time.monotonic() < limit:
        with opener.open('http://127.0.0.1:18025/api/v1/messages', timeout=3) as response:
            messages = json.load(response).get('messages', [])
        if messages:
            with opener.open('http://127.0.0.1:18025/api/v1/message/'+messages[0]['ID'], timeout=3) as response:
                message = json.load(response)
            code = re.search(r'(?<!\d)\d{6}(?!\d)', message.get('Text', ''))
            if code:
                return code.group()
        time.sleep(.5)
    raise RuntimeError('Authenticated SMTP delivery did not arrive within its deadline')


def trust_and_browser(ca, home):
    global CA_SYSTEM_PATH, CA_SYSTEM_INSTALLED
    # Only the ephemeral hosted runner is modified, never a user's host or trust store.
    CA_SYSTEM_PATH = Path('/usr/local/share/ca-certificates') / (PROJECT+'.crt')
    check('Runner-owned CA destination is new', not CA_SYSTEM_PATH.exists())
    command(['sudo', 'install', '-m', '0644', ca, CA_SYSTEM_PATH])
    CA_SYSTEM_INSTALLED = True
    command(['sudo', 'update-ca-certificates'])
    nss = home/'.pki/nssdb'
    nss.mkdir(parents=True, mode=0o700)
    command(['certutil', '-N', '--empty-password', '-d', 'sql:'+str(nss)])
    command(['certutil', '-A', '-d', 'sql:'+str(nss), '-n', PROJECT, '-t', 'C,,', '-i', ca])
    from playwright.sync_api import sync_playwright
    with private_browser_environment(), sync_playwright() as pw:
        browser = pw.chromium.launch(env=child_env({'HOME': str(home)}), args=[
            '--host-resolver-rules=MAP '+AUTH_HOST+' 127.0.0.1, MAP '+APP_HOST+' 127.0.0.1'])
        context = browser.new_context(ignore_https_errors=False)
        page = context.new_page()
        page.set_default_timeout(20000)
        stage('https-browser-login')
        response = page.goto(APP)
        page.locator('#mail').wait_for()
        check('Protected HTTPS service redirects to authentication', urlparse(page.url).scheme == 'https' and urlparse(page.url).hostname == AUTH_HOST)
        page.locator('#mail').fill('production@example.test')
        page.locator('#deliver_via').select_option('email')
        with page.expect_response(lambda r: urlparse(r.url).path == '/_send_verify_code') as sent:
            page.locator('#sendCodeBtn').click()
        RESULT['last_http'] = {'operation': 'send_email', 'status': sent.value.status, 'reason': 'unknown'}
        try:
            sent_body = sent.value.json()
        except ValueError:
            sent_body = {}
        if isinstance(sent_body, dict):
            RESULT['last_http']['reason'] = safe_http_reason(sent_body.get('reason'))
        check('Strict production SMTP creation accepted', sent.value.status == 200 and isinstance(sent_body, dict) and sent_body.get('success') is True and not sent_body.get('debug_code'))
        page.locator('#verify_code').fill(mailbox_code())
        page.locator('#loginForm [type=submit]').click()
        page.wait_for_url(lambda u: url_matches(u, APP))
        check('Real HTTPS browser email login returns to protected app', url_matches(page.url, APP))
        protected = page.evaluate("""async () => { const r = await fetch('/?format=text', {headers:{Accept:'text/plain'},redirect:'manual'}); const text = await r.text(); return {status:r.status,project_matches:r.headers.get('Project')==='https://github.com/soulteary/hello',summary:text.split(String.fromCharCode(10)).map(line=>line.trim()).includes('Hello from soulteary/hello!')}; }""")
        RESULT['protected_app'] = protected
        check('Protected HTTPS app returns its actual200content', protected == {'status':200,'project_matches':True,'summary':True})
        stage('secure-cookie-check')
        cookies = [c for c in context.cookies() if c['name'] == 'stargate_session_id']
        check('Session cookies are Secure HttpOnly and host-only', {c['domain'] for c in cookies} == {AUTH_HOST, APP_HOST} and all(c['secure'] and c['httpOnly'] and c['domain'] in (AUTH_HOST, APP_HOST) and not c['domain'].startswith('.') for c in cookies))
        stage('https-browser-logout')
        page.goto(AUTH+'/_account?lang=en')
        auth = page.evaluate("""async () => { const r = await fetch('/_auth', {headers:{'X-Forwarded-Uri':'/profile'}}); return {status:r.status,user:r.headers.get('X-Auth-User')}; }""")
        check('HTTPS browser session has expected stable identity', auth == {'status': 200, 'user': 'production-user-001'})
        with page.expect_response(lambda r: r.request.method == 'POST' and urlparse(r.url).path == '/_logout') as logged_out:
            page.locator('[data-logout] button').click()
        check('HTTPS account uses real POST logout', logged_out.value.status in (200, 302, 303))
        page.goto(APP)
        page.locator('#loginForm').wait_for()
        check('Protected app requires authentication after logout', urlparse(page.url).hostname == AUTH_HOST and '/_login' in urlparse(page.url).path)
        context.close()
        browser.close()


def main():
    global STATE, PROJECT, DEADLINE
    if not hosted_runner(os.environ, sys.platform):
        raise RuntimeError('This acceptance script runs only in an ephemeral hosted Linux runner')
    for key in ['FULLSTACK_REF', 'GATEWAY_REF']:
        if not re.fullmatch(r'[0-9a-f]{40}', os.environ.get(key, '')):
            raise ValueError('Exact source commit required')
    image = os.environ.get('HERALD_IMAGE', '')
    if image != 'ghcr.io/soulteary/herald@sha256:f2ca281f15d43be1bd5a1304fb7db535c4f53b00fc343162476343dbd3f04dee':
        raise ValueError('The verified Herald v1.6.0 image is required')
    DEADLINE = time.monotonic() + 900
    PROJECT = 'tls-browser-' + secrets.token_hex(6)
    STATE = Path(os.environ['RUNNER_TEMP']) / PROJECT
    STATE.mkdir(mode=0o700)
    tls = STATE/'tls'
    RESULT.update(fullstack_ref=os.environ['FULLSTACK_REF'], gateway_ref=os.environ['GATEWAY_REF'], herald_image=image,
                  harness_source=command(['git', 'rev-parse', 'HEAD']).decode().strip(), browser_certificate_bypass=False)
    save()
    try:
        for directory, key in [('fullstack', 'FULLSTACK_REF'), ('gateway', 'GATEWAY_REF')]:
            check(directory+' exact source', command(['git', 'rev-parse', 'HEAD'], ROOT/directory).decode().strip() == os.environ[key])
        stage('generate-private-certificates')
        ca = certificates(tls)
        private_write(tls/'tls.yml', (ROOT/'fullstack/examples/full-stack/tls.yml.example').read_text())
        stage('build-exact-gateway')
        candidate = PROJECT+'-gateway:source'
        command(['docker', 'build', '--file', ROOT/'gateway/docker/Dockerfile', '--tag', candidate,
                 '--build-arg', 'VERSION=usability-candidate', '--build-arg', 'COMMIT='+os.environ['GATEWAY_REF'], ROOT/'gateway'], env=child_env(), timeout=360)
        image_id = command(['docker', 'image', 'inspect', '--format', '{{.Id}}', candidate]).decode().strip()
        check('Actual source-built gateway has immutable local image ID', re.fullmatch(r'sha256:[0-9a-f]{64}', image_id) is not None)
        RESULT.update(gateway_image_id=image_id, gateway_image_published=False)
        username, password = 'synthetic-smtp-user', secrets.token_hex(24)
        env = child_env({'SMTP_HOST': 'mailpit', 'SMTP_PORT': '1025', 'SMTP_FROM': 'login@example.test',
                         'SMTP_USER': username, 'SMTP_PASSWORD': password, 'SMTP_CA_FILE': str(tls/'smtp/ca.pem'), 'TLS_DIR': str(tls), 'STARGATE_IMAGE': image_id})
        config_dir = STATE/'config'
        stage('configure-production')
        command(['python3', ROOT/'fullstack/examples/full-stack/configure.py', '--environment', 'production', '--mode', 'totp',
                 '--auth-host', AUTH_HOST, '--app-host', APP_HOST, '--port', '19443', '--email', 'production@example.test',
                 '--user-id', 'production-user-001', '--project', PROJECT, '--subnet', '172.31.87.0/24', '--output', config_dir], env=env)
        # Mailpit is an authenticated STARTTLS test fixture, never an external-delivery claim.
        fixture = {'services': {'herald': {'image': image}, 'mailpit': {
            'image': MAILPIT, 'user': str(os.getuid())+':'+str(os.getgid()), 'labels': {'traefik.enable': 'false'},
            'environment': {'MP_SMTP_AUTH': username+':'+password, 'MP_SMTP_TLS_CERT': '/certs/smtp-cert.pem',
                            'MP_SMTP_TLS_KEY': '/certs/smtp-key.pem', 'MP_SMTP_REQUIRE_STARTTLS': 'true',
                            'MP_SMTP_AUTH_ALLOW_INSECURE': 'false', 'MP_SMTP_AUTH_ACCEPT_ANY': 'false', 'MP_SMTP_RELAY_ALL': 'false', 'MP_DATABASE': '/tmp/mailpit.db'},
            'volumes': [str(tls/'smtp-cert.pem')+':/certs/smtp-cert.pem:ro', str(tls/'smtp-key.pem')+':/certs/smtp-key.pem:ro'],
            'ports': ['127.0.0.1:18025:8025'], 'networks': ['delivery'], 'read_only': True, 'tmpfs': ['/tmp'],
            'cap_drop': ['ALL'], 'security_opt': ['no-new-privileges:true']}}}
        override = STATE/'smtp-fixture.yml'
        private_write(override, 'services:\n  traefik:\n    ports: !override ["127.0.0.1:19443:443"]\n  herald:\n    image: '+image+'\n  mailpit: '+json.dumps(fixture['services']['mailpit'])+'\n')
        envfile = config_dir/'.env'
        values = read_generated_env(envfile)
        lines = envfile.read_text().splitlines()
        lines = ['COMPOSE_FILE='+json.dumps(values['COMPOSE_FILE']+os.pathsep+str(override)) if line.startswith('COMPOSE_FILE=') else line for line in lines]
        private_write(envfile, '\n'.join(lines)+'\n')
        STATE = config_dir
        stage('production-stack-startup')
        rendered = json.loads(compose('config', '--format', 'json'))
        check('CI production fixture publishes only loopback ports', all(port.get('host_ip') in {'127.0.0.1', '::1'} for service in rendered['services'].values() for port in service.get('ports', [])))
        check('CI production fixture has no host networking or Docker socket', all(service.get('network_mode') != 'host' and all('docker.sock' not in str(volume) for volume in service.get('volumes', [])) for service in rendered['services'].values()))
        RESULT['compose_version'] = command(['docker', 'compose', 'version', '--short']).decode().strip()
        compose('up', '-d', '--wait', '--wait-timeout', '180')
        stage('authenticated-smtp-control')
        smtp_negative_control(tls/'smtp/ca.pem', username)
        stage('production-totp-https-proxy')
        signed_totp_status(values['STARGATE_HERALD_HMAC'])
        stage('ephemeral-browser-ca-trust')
        trust_and_browser(ca, tls/'browser-home')
        RESULT['status'] = 'passed'
    except Exception as error:
        RESULT['status'] = 'failed'
        RESULT['failure'] = safe_failure(error)
        RESULT['failed_stage'] = RESULT.get('stage', 'initialization')
        RESULT['failed_command'] = RESULT.get('last_command')
        RESULT['error'] = 'Production TLS acceptance failed; raw exceptions, browser state and service output withheld'
    finally:
        cleanup = {}
        candidate_name = locals().get('candidate')
        try:
            compose('down', '--volumes', '--remove-orphans', '--timeout', '15', cleanup=True)
            for key, args in [('containers', ['ps', '-aq']), ('networks', ['network', 'ls', '-q']), ('volumes', ['volume', 'ls', '-q'])]:
                cleanup[key] = command(['docker', *args, '--filter', 'label=com.docker.compose.project='+PROJECT], cleanup=True).decode().split()
            if any(cleanup.values()):
                raise RuntimeError('Owned resources remain')
            if candidate_name and locals().get('image_id'):
                command(['docker', 'image', 'rm', candidate_name], cleanup=True)
                cleanup['owned_candidate_image_tag_removed'] = True
        except Exception:
            cleanup['status'] = 'failed'
            RESULT['status'] = 'failed'
        if CA_SYSTEM_INSTALLED and CA_SYSTEM_PATH and CA_SYSTEM_PATH.exists():
            try:
                command(['sudo', 'rm', '--', CA_SYSTEM_PATH], cleanup=True)
                command(['sudo', 'update-ca-certificates'], cleanup=True)
                cleanup['ephemeral_system_ca_removed'] = not CA_SYSTEM_PATH.exists()
            except Exception:
                cleanup['ephemeral_system_ca_removed'] = False
                RESULT['status'] = 'failed'
        RESULT['cleanup'] = cleanup
        # Remove only this script's owned private temporary tree, including NSS keys/browser data.
        owned = Path(os.environ['RUNNER_TEMP'])/PROJECT
        if owned.exists():
            shutil.rmtree(owned)
        RESULT['private_test_state_removed'] = not owned.exists()
        RESULT['process_exit_code'] = 0 if RESULT['status'] == 'passed' else 1
        save()
    return RESULT['process_exit_code']


if __name__ == '__main__':
    raise SystemExit(main())
