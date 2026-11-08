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
