import importlib.util
from pathlib import Path
import subprocess
import os
from unittest.mock import patch
import unittest

module_path = Path(__file__).with_name('production_tls.py')
if not module_path.exists():
    module_path = Path(__file__).with_name('usability10-production-tls.py')
spec = importlib.util.spec_from_file_location('production_tls', module_path)
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


class Helpers(unittest.TestCase):
    def test_playwright_callback_uses_actual_string_url(self):
        self.assertTrue(helper.url_matches('https://app.browser.example.com:19443/?ok=1', helper.APP))
        self.assertFalse(helper.url_matches('http://app.browser.example.com:19443/', helper.APP))
        self.assertFalse(helper.url_matches('https://app.browser.example.com.attacker.test/', helper.APP))
        self.assertFalse(helper.url_matches('https://app.browser.example.com:19444/', helper.APP))
        self.assertFalse(helper.url_matches('https://user:password@app.browser.example.com:19443/', helper.APP))
        self.assertFalse(helper.url_matches('https://app.browser.example.com:19443/error', helper.APP))
        self.assertFalse(helper.url_matches('https://app.browser.example.com:bad/', helper.APP))

    def test_self_hosted_or_local_machine_cannot_change_trust(self):
        values = {'GITHUB_ACTIONS': 'true', 'RUNNER_OS': 'Linux', 'RUNNER_ENVIRONMENT': 'github-hosted'}
        self.assertTrue(helper.hosted_runner(values, 'linux'))
        self.assertFalse(helper.hosted_runner(dict(values, RUNNER_ENVIRONMENT='self-hosted'), 'linux'))
        self.assertFalse(helper.hosted_runner({}, 'linux'))
        self.assertFalse(helper.hosted_runner(values, 'darwin'))

    def test_signed_request_redirects_are_not_followed(self):
        self.assertIsNone(helper.NoRedirect().redirect_request(None, None, 302, 'Found', {}, 'https://untrusted.invalid'))

    def test_child_environment_excludes_runner_credentials(self):
        with patch.dict(os.environ, {'GITHUB_TOKEN': 'synthetic-private', 'ACTIONS_RUNTIME_TOKEN': 'private-runtime',
                                     'SMTP_PASSWORD': 'private-smtp', 'PATH': '/safe-bin'}, clear=True):
            self.assertEqual(helper.child_env(), {'PATH': '/safe-bin'})
            with helper.private_browser_environment():
                self.assertNotIn('GITHUB_TOKEN', os.environ)
                self.assertNotIn('ACTIONS_RUNTIME_TOKEN', os.environ)
            self.assertEqual(os.environ['GITHUB_TOKEN'], 'synthetic-private')

    def test_diagnostics_never_include_raw_error_or_server_reason(self):
        sensitive = 'synthetic-password-never-log'
        self.assertEqual(helper.safe_failure(RuntimeError(sensitive)), {'reason': 'unexpected_failure'})
        self.assertEqual(helper.safe_failure(subprocess.TimeoutExpired(['private-command', sensitive], 1)), {'reason': 'command_timeout'})
        self.assertEqual(helper.safe_http_reason(sensitive), 'unknown')
        self.assertEqual(helper.safe_http_reason('resend_cooldown'), 'resend_cooldown')


if __name__ == '__main__':
    unittest.main()
