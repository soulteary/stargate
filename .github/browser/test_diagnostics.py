import json
import unittest
from diagnostics import smtp_signatures


class Diagnostics(unittest.TestCase):
    def test_only_boolean_signatures_survive(self):
        secret = "private-password@example.test"
        raw = '\n'.join(json.dumps({"error": error, "mail": secret}) for error in (
            "STARTTLS failed: x509: certificate signed by unknown authority " + secret,
            "SMTP dial failed: " + secret, "authentication failed: " + secret)).encode()
        result = smtp_signatures(raw)
        self.assertTrue(all(result.values()))
        self.assertTrue(all(type(value) is bool for value in result.values()))
        self.assertNotIn(secret, json.dumps(result))
        self.assertNotIn("example.test", json.dumps(result))

    def test_unstructured_and_non_error_fields_do_not_classify(self):
        for raw in (b'STARTTLS failed: private', b'{"message":"STARTTLS failed: private"}',
                    b'{"error":null}', b'[]', b'{"error":{"raw":"STARTTLS failed:"}}'):
            self.assertFalse(any(smtp_signatures(raw).values()))

    def test_oversized_lines_ignored_and_x509_requires_starttls(self):
        raw = json.dumps({"error": "STARTTLS failed:" + "x"*9000}).encode()
        self.assertFalse(any(smtp_signatures(raw).values()))
        self.assertFalse(smtp_signatures(b'{"error":"x509: unrelated"}')["starttls_x509_failure_seen"])


class PostHistory(unittest.TestCase):
    def test_only_specific_playwright_error_allows_optional_gap(self):
        from diagnostics import post_history_cache_miss
        class PlaywrightError(Exception):
            pass
        self.assertTrue(post_history_cache_miss(PlaywrightError("Page.go_back: net::ERR_CACHE_MISS"), PlaywrightError))
        for error in (Exception("net::ERR_CACHE_MISS"), PlaywrightError("timeout"),
                      PlaywrightError("net::ERR_CONNECTION_REFUSED"), PlaywrightError("ERR_CACHE_MISS")):
            self.assertFalse(post_history_cache_miss(error, PlaywrightError))


class RecoveryControlFlow(unittest.TestCase):
    def test_non_cache_errors_are_rethrown_without_new_tab(self):
        import importlib.util
        import os
        from pathlib import Path
        import tempfile
        import types
        from unittest.mock import patch
        class BrowserError(Exception):
            pass
        api = types.ModuleType("playwright.sync_api")
        api.Error = BrowserError
        api.sync_playwright = lambda: None
        with tempfile.TemporaryDirectory() as temporary, patch.dict(os.environ, {"RUNNER_TEMP": temporary}), patch.dict("sys.modules", {"playwright": types.ModuleType("playwright"), "playwright.sync_api": api}):
            spec = importlib.util.spec_from_file_location("browser_control_test", Path(__file__).parent / "run.py")
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)
            module.save = lambda: None
            module.stage = lambda name: None
            class Context:
                def new_page(self):
                    raise AssertionError("unrelated error opened a new tab")
            for error in (BrowserError("timeout"), BrowserError("net::ERR_ABORTED"), Exception("net::ERR_CACHE_MISS")):
                class Page:
                    def goto(self, url):
                        pass
                    def go_back(self):
                        raise error
                with self.assertRaises(type(error)) as caught:
                    module.probe_bfcache(Context(), Page(), [], lambda request: None)
                self.assertIs(caught.exception, error)

    def test_failure_category_never_echoes_input(self):
        from diagnostics import playwright_failure_category
        class BrowserError(Exception):
            pass
        secret = "private@example.test password cookie"
        self.assertEqual(playwright_failure_category(BrowserError("interrupted by another navigation " + secret), BrowserError), "navigation_interrupted")
        self.assertEqual(playwright_failure_category(BrowserError(secret), BrowserError), "other_playwright_error")
        self.assertEqual(playwright_failure_category(Exception("net::ERR_CACHE_MISS " + secret), BrowserError), "not_playwright")
