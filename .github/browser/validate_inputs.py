"""Validate source identities before checkout or shell interpolation."""
import os
import re
from urllib.parse import urlparse


def validate(values):
    for key in ("FULLSTACK_REF", "GATEWAY_REF"):
        if not re.fullmatch(r"[0-9a-f]{40}", values.get(key, "")):
            raise ValueError("Source inputs must be full lowercase commit hashes")
    if not re.fullmatch(r"ghcr\.io/soulteary/herald@sha256:[0-9a-f]{64}", values.get("HERALD_IMAGE", "")):
        raise ValueError("Herald must use its verified canonical image digest")


def hello_text_url(base):
    return base.rstrip("/") + "/?format=text"


def callback_matches(raw, target):
    try:
        expected = urlparse(target)
        candidate = urlparse(raw if "://" in raw else "//" + raw)
        return (not candidate.username and not candidate.password
                and candidate.scheme in ("", expected.scheme)
                and candidate.hostname == expected.hostname
                and candidate.port == expected.port
                and candidate.path in ("", "/")
                and not candidate.query and not candidate.fragment)
    except ValueError:
        return False


if __name__ == "__main__":
    validate(os.environ)
