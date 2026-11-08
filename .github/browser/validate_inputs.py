"""Validate source identities before checkout or shell interpolation."""
import os
import re


def validate(values):
    for key in ("FULLSTACK_REF", "GATEWAY_REF"):
        if not re.fullmatch(r"[0-9a-f]{40}", values.get(key, "")):
            raise ValueError("Source inputs must be full lowercase commit hashes")
    if not re.fullmatch(r"ghcr\.io/soulteary/herald@sha256:[0-9a-f]{64}", values.get("HERALD_IMAGE", "")):
        raise ValueError("Herald must use its verified canonical image digest")


if __name__ == "__main__":
    validate(os.environ)
