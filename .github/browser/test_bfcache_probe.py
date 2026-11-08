import json
import unittest
from bfcache_probe import genuine_restore, safe_reasons, approved_login_entry, accepted_login, exact_cdp_body_limit, safe_experiment_failure, classify_back_landing
import production_tls as fixture


class BFCacheProof(unittest.TestCase):
    def test_reload_and_request_cannot_be_called_restoration(self):
        self.assertTrue(genuine_restore(True, True, 0))
        for values in ((False, True, 0), (True, False, 0), (True, True, 1), (True, True, False), (1, True, 0)):
            self.assertFalse(genuine_restore(*values))

    def test_only_reason_enum_codes_survive(self):
        secret = 'private@example.test?token=secret'
        events = [{'url': secret, 'notRestoredExplanations': [
            {'reason':'CacheControlNoStoreCookieModified', 'context':secret},
            {'reason':'HTTPMethodNotGET'}, {'reason':secret}]}]
        result = safe_reasons(events)
        self.assertEqual(result, {'codes':['CacheControlNoStoreCookieModified','HTTPMethodNotGET'], 'unknown_count':1})
        self.assertNotIn(secret, json.dumps(result))


class AppEntry(unittest.TestCase):
    def test_entry_requires_real_auth_origin_and_exact_callback(self):
        self.assertTrue(approved_login_entry(fixture.AUTH+'/_login?state=synthetic', [fixture.APP]))
        for origin, callbacks in ((fixture.AUTH+'/_login', []), (fixture.APP+'/_login', [fixture.APP]),
                                  (fixture.AUTH+'/_login', ['https://attacker.invalid']),
                                  (fixture.AUTH.replace('https:', 'http:')+'/_login', [fixture.APP])):
            self.assertFalse(approved_login_entry(origin, callbacks))

    def test_login_requires_exact_status_and_boolean_success(self):
        self.assertTrue(accepted_login(200, {'success':True}))
        for status, body in ((201, {'success':True}), (401, {'success':True}), (200, {'success':1}),
                             (200, {}), (200, ['success'])):
            self.assertFalse(accepted_login(status, body))


class BodyObservation(unittest.TestCase):
    def test_only_precise_playwright_resource_limit_has_fallback(self):
        class PlaywrightError(Exception):
            pass
        marker = 'Protocol error (Network.getResponseBody): No resource with given identifier found'
        self.assertTrue(exact_cdp_body_limit(PlaywrightError(marker), PlaywrightError))
        self.assertTrue(exact_cdp_body_limit(PlaywrightError('Network.getResponseBody: No data found for resource with given identifier'), PlaywrightError))
        for error in (Exception(marker), PlaywrightError('JSON parse failed'), PlaywrightError('Network.getResponseBody: access denied')):
            self.assertFalse(exact_cdp_body_limit(error, PlaywrightError))

    def test_failure_type_and_category_do_not_echo_values(self):
        secret = 'private@example.test-secret-code'
        result = safe_experiment_failure(RuntimeError(secret))
        self.assertEqual(result['exception_type'], 'RuntimeError')
        self.assertNotIn(secret, json.dumps(result))


class BackLanding(unittest.TestCase):
    def test_only_exact_approved_product_origins_and_paths(self):
        self.assertEqual(classify_back_landing(fixture.APP+'/'), 'approved_app')
        self.assertEqual(classify_back_landing(fixture.AUTH+'/_login?opaque=synthetic'), 'auth_login')
        for value in ('https://attacker.invalid/', fixture.APP+'/other', fixture.APP.replace(':19443',':19444'), fixture.AUTH+'/_account', fixture.APP.replace('https:','http:')):
            self.assertEqual(classify_back_landing(value), 'other')
