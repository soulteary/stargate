import unittest
from validate_inputs import validate, callback_matches, hello_text_url


class Inputs(unittest.TestCase):
    def fixture(self):
        return {"FULLSTACK_REF": "a" * 40, "GATEWAY_REF": "b" * 40,
                "HERALD_IMAGE": "ghcr.io/soulteary/herald@sha256:" + "c" * 64}

    def test_immutable_inputs(self):
        validate(self.fixture())

    def test_mutable_or_shell_inputs_rejected(self):
        for field in self.fixture():
            for value in ("main", "v1.5.0", "", "$(echo unsafe)", "a"*39, "A"*40):
                with self.subTest(field=field, value=value):
                    data = self.fixture()
                    data[field] = value
                    with self.assertRaises(ValueError):
                        validate(data)

    def test_other_image_rejected(self):
        data = self.fixture()
        data["HERALD_IMAGE"] = "ghcr.io/other/herald@sha256:" + "c" * 64
        with self.assertRaises(ValueError):
            validate(data)


if __name__ == "__main__":
    unittest.main()

class CompletionRecord(unittest.TestCase):
    def test_workflow_completion_record_remains_json(self):
        import json
        from pathlib import Path
        import subprocess
        import sys
        import tempfile
        workflow = (Path(__file__).parents[1] / "workflows/browser-acceptance.yml").read_text()
        import re
        wrappers = [part.split("          PYTHON", 1)[0] for part in workflow.split("<<'PYTHON'\n")[1:] if 'data["process_exit_code"]' in part.split("          PYTHON", 1)[0]]
        self.assertEqual(len(wrappers), 2)
        for block in wrappers:
            code = "\n".join(line[10:] for line in block.splitlines())
            filename = re.search(r'path = pathlib.Path\("([a-z-]+\.json)"\)', code).group(1)
            self.assertIn(filename, {"browser-results.json", "production-tls-results.json"})
            with tempfile.TemporaryDirectory() as temporary:
                path = Path(temporary) / filename
                for exit_code in (0, 1):
                    path.write_text('{"status":"passed"}')
                    subprocess.run([sys.executable, "-c", code, str(exit_code)], cwd=temporary, check=True)
                    result = json.loads(path.read_text())
                    self.assertEqual(result["process_exit_code"], exit_code)
                    self.assertEqual(result["status"], "passed" if exit_code == 0 else "failed")

class BrowserURLPredicates(unittest.TestCase):
    def test_wait_url_predicates_accept_playwright_strings(self):
        import ast
        from pathlib import Path
        from urllib.parse import urlparse
        tree = ast.parse((Path(__file__).parent / "run.py").read_text())
        predicates = [node.args[0] for node in ast.walk(tree)
                      if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute)
                      and node.func.attr == "wait_for_url" and len(node.args) == 1
                      and isinstance(node.args[0], ast.Lambda)
                      and "APP" not in ast.unparse(node.args[0]) and "AUTH" not in ast.unparse(node.args[0])]
        self.assertGreaterEqual(len(predicates), 4)
        for predicate in predicates:
            function = eval(compile(ast.Expression(predicate), "predicate", "eval"), {"urlparse": urlparse})
            self.assertFalse(function("http://auth.test.localhost:8088/_login"))
            self.assertTrue(function("http://auth.test.localhost:8088/_account"))


class CallbackAuthority(unittest.TestCase):
    def test_exact_url_and_authority(self):
        target = "http://hello.test.localhost:8088"
        for raw in (target, target + "/", "hello.test.localhost:8088/"):
            self.assertTrue(callback_matches(raw, target))
        for raw in ("http://hello.test.localhost:9090", "http://other.test.localhost:8088", "http://hello.test.localhost:8088.evil.test", "https://hello.test.localhost:8088", "http://user@hello.test.localhost:8088", target + "/secret", target + "?q=1"):
            self.assertFalse(callback_matches(raw, target))


class HelloFormat(unittest.TestCase):
    def test_explicit_format_preserves_example_origin(self):
        from urllib.parse import urlparse, parse_qs
        for base in ("http://hello.test.localhost:8088", "https://app.example.com:8443/"):
            parsed = urlparse(hello_text_url(base))
            self.assertEqual(parsed.netloc, urlparse(base).netloc)
            self.assertEqual(parsed.scheme, urlparse(base).scheme)
            self.assertEqual(parsed.path, "/")
            self.assertEqual(parse_qs(parsed.query), {"format": ["text"]})
