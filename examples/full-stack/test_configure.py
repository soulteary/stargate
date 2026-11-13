"""Offline safety and repeatability checks for the full-stack example setup."""
import argparse
import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import stat
import shutil
import subprocess
import tempfile
import time

import certificates
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('example_configure', Path(__file__).with_name('configure.py'))
config = importlib.util.module_from_spec(spec)
spec.loader.exec_module(config)


class ConfigurationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.output = Path(self.tmp.name) / 'deployment'

    def args(self, **values):
        result = dict(output=str(self.output), environment='local', mode='mail', auth_host=None,
                      app_host=None, port=None, mailbox_port=8025, project='stargate-example',
                      subnet='172.31.86.0/24', email=None, user_id=None)
        result.update(values)
        return argparse.Namespace(**result)

    def test_local_delivery_and_nondefault_authorities(self):
        self.assertTrue(config.configure(self.args(), {}))
        text = (self.output / '.env').read_text()
        self.assertIn('AUTH_AUTHORITY="auth.test.localhost:8088"', text)
        self.assertIn('APP_AUTHORITY="hello.test.localhost:8088"', text)
        self.assertIn('SMTP_HOST="mailpit"', text)
        self.assertIn('COOKIE_SECURE="false"', text)
        self.assertIn('compose.local.yml', text)
        self.assertEqual(stat.S_IMODE((self.output / '.env').stat().st_mode), 0o600)
        self.assertEqual(stat.S_IMODE((self.output / 'state').stat().st_mode), 0o700)
        users = json.loads((self.output / 'state/users.json').read_text())
        self.assertEqual(users[0]['user_id'], 'demo-user-001')
        self.assertEqual(users[0]['status'], 'active')

    def test_repeat_retains_all_keys_and_identity(self):
        args = self.args(mode='totp')
        config.configure(args, {})
        before = {p: p.read_bytes() for p in self.output.rglob('*') if p.is_file()}
        with patch.object(config.secrets, 'token_hex', side_effect=AssertionError('must not rotate')):
            self.assertFalse(config.configure(args, {}))
        self.assertEqual(before, {p: p.read_bytes() for p in before})

    def test_existing_missing_tls_is_partial_and_never_recreated(self):
        args = self.args()
        config.configure(args, {})
        state = self.output / 'state/local-smtp'
        (state / 'ca.pem').unlink()
        before = (self.output / '.env').read_bytes()
        with self.assertRaisesRegex(ValueError, 'partial SMTP TLS'):
            config.configure(args, {})
        self.assertFalse((state / 'ca.pem').exists())
        self.assertEqual((self.output / '.env').read_bytes(), before)

    def test_mismatched_smtp_private_key_is_rejected_without_overwrite(self):
        args = self.args()
        config.configure(args, {})
        directory = self.output / 'state/local-smtp'
        (directory / 'smtp-key.pem').write_bytes((directory / 'ca-key.pem').read_bytes())
        before = (directory / 'smtp-key.pem').read_bytes()
        with self.assertRaisesRegex(ValueError, 'do not match'):
            config.configure(args, {})
        self.assertEqual((directory / 'smtp-key.pem').read_bytes(), before)

    def test_openssl_required_without_partial_output(self):
        with patch.object(config.shutil, 'which', return_value=None):
            with self.assertRaisesRegex(ValueError, 'OpenSSL'):
                config.configure(self.args(), {})
        self.assertFalse(self.output.exists())

    def test_actual_smtp_certificate_verification_and_permissions(self):
        config.configure(self.args(), {})
        directory = self.output / 'state/local-smtp'
        for name in ['ca-key.pem', 'smtp-key.pem']:
            self.assertEqual(stat.S_IMODE((directory / name).stat().st_mode), 0o600)
        certificates.openssl('verify', '-CAfile', directory / 'ca.pem', '-verify_hostname', 'mailpit', directory / 'smtp-cert.pem')
        for extra, reason in [(['-verify_hostname', 'wrong.example'], b'hostname mismatch'),
                              (['-attime', str(int(time.time()) + 400 * 86400)], b'expired')]:
            process = subprocess.run(['openssl', 'verify', '-CAfile', str(directory / 'ca.pem'), *extra,
                                      str(directory / 'smtp-cert.pem')], capture_output=True, timeout=10)
            self.assertNotEqual(process.returncode, 0)
            self.assertIn(reason, (process.stdout + process.stderr).lower())

    def test_dollar_and_backslash_paths_repeat_without_mutation(self):
        self.output = Path(self.tmp.name) / 'private$path\\with-slash'
        args = self.args()
        config.configure(args, {})
        before = {p: p.read_bytes() for p in self.output.rglob('*') if p.is_file()}
        self.assertFalse(config.configure(args, {}))
        self.assertEqual(before, {p: p.read_bytes() for p in before})

    def test_missing_provider_bundle_is_partial_for_production_too(self):
        self.output.mkdir()
        state = self.output / 'state'
        state.mkdir()
        (state / 'users.json').write_text('[]')
        (state / 'routes.yml').write_text('{}')
        private = self.output / '.env'
        private.write_text('COMPOSE_FILE="compose.yml:compose.production.yml"\n')
        private.chmod(0o600)
        before = private.read_bytes()
        with self.assertRaisesRegex(ValueError, 'provider CA bundle is missing'):
            config.configure(self.args(environment='production'), {})
        self.assertFalse((state / 'provider-ca.pem').exists())
        self.assertEqual(private.read_bytes(), before)

    def test_distinct_keys_and_exact_encryption_key_bytes(self):
        config.configure(self.args(), {})
        text = (self.output / '.env').read_text()
        values = dict(line.split('=', 1) for line in text.splitlines())
        self.assertEqual(len(set(values[key] for key in config.SECRET_KEYS)), len(config.SECRET_KEYS))
        self.assertEqual(len(values['TOTP_ENCRYPTION_KEY'][1:-1].encode()), 32)

    def test_default_ports_have_no_authority_suffix(self):
        self.assertEqual(config.authority('auth.example.test', 80, 80), 'auth.example.test')
        self.assertEqual(config.authority('auth.example.test', 443, 443), 'auth.example.test')
        self.assertEqual(config.authority('auth.example.test', 9443, 443), 'auth.example.test:9443')

    def test_rejects_unsafe_or_ambiguous_hosts_before_writing(self):
        for host in ['https://auth.example.test', 'auth.example.test:8080', 'a.test`) || Path(`/x',
                     '127.0.0.1', 'a..test', '-a.test', 'a.test/path', 'a.test\nOTHER=value']:
            with self.subTest(host=host), self.assertRaises(ValueError):
                config.configure(self.args(auth_host=host), {})
            self.assertFalse(self.output.exists())

    def test_rejects_conflicting_ports_or_public_subnet(self):
        for args in [self.args(port=8025), self.args(port=65536), self.args(subnet='8.8.8.0/24'),
                     self.args(subnet='127.0.0.0/24'), self.args(auth_host='same.test', app_host='same.test')]:
            with self.assertRaises(ValueError):
                config.configure(args, {})
            self.assertFalse(self.output.exists())

    def test_partial_state_fails_without_overwrite(self):
        self.output.mkdir()
        (self.output / 'state').mkdir()
        with self.assertRaisesRegex(ValueError, 'partial'):
            config.configure(self.args(), {})
        self.assertFalse((self.output / '.env').exists())

    def test_production_requires_explicit_identity_and_provider(self):
        with self.assertRaises(ValueError):
            config.configure(self.args(environment='production'), {})
        args = self.args(environment='production', auth_host='auth.example.com', app_host='app.example.com',
                         email='real@example.com', user_id='stable-123')
        with self.assertRaisesRegex(ValueError, 'SMTP_HOST'):
            config.configure(args, {})
        self.assertFalse(self.output.exists())

    def test_production_configuration_is_tls_and_has_no_capture_service(self):
        tls = Path(self.tmp.name) / 'tls'
        tls.mkdir()
        (tls / 'tls.yml').write_text('tls: {}')
        (tls / 'totp-ca.pem').write_text('synthetic-test-public-ca')
        env = dict(SMTP_HOST='smtp.example.com', SMTP_PORT='587', SMTP_FROM='sender@example.com',
                   SMTP_USER='sender', SMTP_PASSWORD="synthetic$quote'password", TLS_DIR=str(tls),
                   STARGATE_IMAGE='registry.example.com/stargate@sha256:' + 'a' * 64)
        args = self.args(environment='production', mode='totp', auth_host='auth.example.com',
                         app_host='app.example.com', email='user@example.com', user_id='stable-001')
        with patch.object(config.ssl, 'SSLContext') as context, patch.object(config.ssl, 'create_default_context') as trust:
            config.configure(args, env)
            self.assertEqual(context.return_value.load_cert_chain.call_count, 2)
            trust.assert_called_once_with(cafile=str(tls.resolve() / 'totp-ca.pem'))
        text = (self.output / '.env').read_text()
        self.assertIn('COOKIE_SECURE="true"', text)
        self.assertIn('AUTH_AUTHORITY="auth.example.com"', text)
        self.assertIn('compose.production.yml', text)
        self.assertNotIn('compose.local.yml', text)
        routes = json.loads((self.output / 'state/routes.yml').read_text())
        self.assertTrue(all('tls' in router for router in routes['http']['routers'].values()))
        self.assertEqual((self.output / 'state/totp-tls.yml').read_bytes(),
                         (config.HERE / 'totp-tls.yml.example').read_bytes())
        if shutil.which('docker'):
            child_env = {key: os.environ[key] for key in ['PATH', 'HOME', 'DOCKER_CONFIG'] if key in os.environ}
            rendered = subprocess.run(['docker', 'compose', 'config', '--format', 'json'],
                                      cwd=self.output, env=child_env, capture_output=True, timeout=30)
            self.assertEqual(rendered.returncode, 0, 'Production Compose rendering failed')
            services = json.loads(rendered.stdout)['services']
            self.assertNotIn('mailpit', services)
            self.assertNotIn('build', services['stargate'])
            self.assertEqual(services['stargate']['environment']['COOKIE_SECURE'], 'true')
            self.assertEqual(services['traefik']['ports'][0]['target'], 443)
            self.assertEqual(services['herald']['environment']['ENVIRONMENT'], 'production')
            self.assertEqual(services['herald']['environment']['HERALD_TOTP_BASE_URL'], 'https://totp-tls:8443')
            self.assertEqual(services['herald']['environment']['SSL_CERT_FILE'], '/certs/provider-ca.pem')
            self.assertFalse(services['totp-tls'].get('ports'))
            self.assertEqual(services['herald']['environment']['SMTP_HOST'], 'smtp.example.com')
            # Compose config serializes literal dollars as $$ on some CLI versions.
            self.assertIn(services['herald']['environment']['SMTP_PASSWORD'],
                          [env['SMTP_PASSWORD'], env['SMTP_PASSWORD'].replace('$', '$$')])

    def test_private_existing_file_is_required(self):
        args = self.args()
        config.configure(args, {})
        (self.output / '.env').chmod(0o644)
        with self.assertRaisesRegex(ValueError, '0600'):
            config.configure(args, {})

    def test_dotenv_preserves_special_password_without_interpolation(self):
        values = ["a$bc 'quoted'\\tail", 'ending\\', '${UNSET_VARIABLE}', 'two$$literal']
        for value in values:
            encoded = config.dotenv({'SMTP_PASSWORD': value})
            self.assertEqual(encoded, 'SMTP_PASSWORD=' + json.dumps(value).replace('$', '$$') + '\n')
        with self.assertRaises(ValueError):
            config.dotenv({'SMTP_PASSWORD': 'secret\nINJECTED=true'})
        if shutil.which('docker'):
            for index, value in enumerate(values):
                output = Path(self.tmp.name) / ('credential-' + str(index))
                output.mkdir()
                (output / '.env').write_text(config.dotenv({'SMTP_PASSWORD': value}))
                (output / 'compose.json').write_text(json.dumps({'services': {'probe': {
                    'image': 'alpine', 'environment': {'SMTP_PASSWORD': '${SMTP_PASSWORD:?}'}}}}))
                env = {key: os.environ[key] for key in ['PATH', 'HOME', 'DOCKER_CONFIG'] if key in os.environ}
                result = subprocess.run(['docker', 'compose', '-f', 'compose.json', 'config', '--format', 'json'],
                                        cwd=output, env=env, capture_output=True, timeout=30)
                self.assertEqual(result.returncode, 0, 'Credential Compose parsing failed')
                actual = json.loads(result.stdout)['services']['probe']['environment']['SMTP_PASSWORD']
                self.assertIn(actual, [value, value.replace('$', '$$')])

    def test_actual_compose_rendering_local_modes(self):
        if not shutil.which('docker'):
            self.skipTest('Docker Compose CLI is unavailable; CI supplies it')
        for mode in ['mail', 'totp']:
            self.output = Path(self.tmp.name) / mode
            config.configure(self.args(mode=mode), {})
            env = {key: os.environ[key] for key in ['PATH', 'HOME', 'DOCKER_CONFIG'] if key in os.environ}
            command = subprocess.run(['docker', 'compose', 'config', '--format', 'json'],
                                     cwd=self.output, env=env, capture_output=True, timeout=30)
            # Never include rendered configuration (which contains secrets) in test diagnostics.
            self.assertEqual(command.returncode, 0, 'Compose configuration rendering failed')
            rendered = json.loads(command.stdout)
            services = rendered['services']
            self.assertTrue(all(service.get('labels', {}).get('traefik.enable') == 'false' for service in services.values()))
            self.assertFalse(any('docker.sock' in str(volume) for service in services.values() for volume in service.get('volumes', [])))
            self.assertEqual(services['herald']['image'], 'ghcr.io/soulteary/herald:1.6.0@sha256:f2ca281f15d43be1bd5a1304fb7db535c4f53b00fc343162476343dbd3f04dee')
            self.assertEqual(services['stargate']['image'], 'stargate-example-stargate:local')
            self.assertEqual(Path(services['stargate']['build']['context']), config.HERE.parents[1])
            gateway = services['stargate']['environment']
            self.assertEqual(gateway['AUTH_HOST'], 'auth.test.localhost:8088')
            self.assertEqual(gateway['CALLBACK_ALLOWED_HOSTS'], 'hello.test.localhost:8088')
            self.assertEqual(gateway['HERALD_TOTP_ENABLED'], str(mode == 'totp').lower())
            self.assertEqual('totp' in services, mode == 'totp')
            self.assertEqual(services['herald']['environment']['PROVIDER_FAILURE_POLICY'], 'strict')
            self.assertEqual(services['herald']['environment']['ENVIRONMENT'], 'development')
            self.assertEqual(services['herald']['environment']['HERALD_TEST_MODE'], 'false')
            self.assertEqual(services['herald']['environment']['SSL_CERT_FILE'], '/certs/provider-ca.pem')
            self.assertEqual(services['mailpit']['environment']['MP_SMTP_REQUIRE_STARTTLS'], 'true')
            self.assertFalse(any('ca-key.pem' in str(v) for service in services.values() for v in service.get('volumes', [])))
            self.assertIn('--appendonly yes', ' '.join(services['redis']['command']))
            self.assertEqual(services['mailpit']['ports'][0]['host_ip'], '127.0.0.1')
            for service in ['warden', 'herald', 'redis']:
                self.assertFalse(services[service].get('ports'))
            self.assertEqual(json.loads(services['warden']['environment']['WARDEN_HMAC_KEYS'])['stargate-directory'], gateway['WARDEN_HMAC_SECRET'])
            self.assertEqual(json.loads(services['herald']['environment']['HERALD_HMAC_KEYS'])['stargate-otp'], gateway['HERALD_HMAC_SECRET'])
            self.assertEqual(services['herald']['environment']['HERALD_TOTP_HMAC_KEY_ID'], 'herald-totp')

    def test_cli_does_not_print_generated_credentials(self):
        output = io.StringIO()
        with patch('sys.argv', ['configure.py', '--output', str(self.output)]), contextlib.redirect_stdout(output):
            self.assertEqual(config.main(), 0)
        env = (self.output / '.env').read_text()
        for line in env.splitlines():
            if line.startswith(tuple(config.SECRET_KEYS)):
                self.assertNotIn(line.split('=', 1)[1][1:-1], output.getvalue())


if __name__ == '__main__':
    unittest.main()
