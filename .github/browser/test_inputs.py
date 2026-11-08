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
