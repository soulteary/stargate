import json
import unittest
from bfcache_probe import genuine_restore, safe_reasons


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
