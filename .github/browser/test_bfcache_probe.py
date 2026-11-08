import json
import unittest
from bfcache_probe import genuine_restore, safe_reasons, approved_login_entry, accepted_login
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
