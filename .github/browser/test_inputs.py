import unittest
from validate_inputs import validate


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
        block = workflow.rsplit("<<'PYTHON'\n", 1)[1].split("          PYTHON", 1)[0]
        code = "\n".join(line[10:] for line in block.splitlines())
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / "browser-results.json"
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
                      and node.func.attr == "wait_for_url"]
        self.assertGreaterEqual(len(predicates), 4)
        for predicate in predicates:
            function = eval(compile(ast.Expression(predicate), "predicate", "eval"), {"urlparse": urlparse})
            self.assertFalse(function("http://auth.test.localhost:8088/_login"))
            self.assertTrue(function("http://auth.test.localhost:8088/_account"))
