#!/usr/bin/env python3
"""Keep current architecture/deployment version descriptions tied to source."""
from pathlib import Path
import re
import sys

root = Path(__file__).resolve().parents[2]
module = (root / "go.mod").read_text()
go_version = re.search(r"^go (\S+)$", module, re.M).group(1)


def dependency(name):
    return re.search(r"^\s*" + re.escape(name) + r"\s+v(\S+)$", module, re.M).group(1)


fiber_version = dependency("github.com/gofiber/fiber/v3")
template_version = dependency("github.com/gofiber/template/html/v3")
rules = [
    (r"\bGo (\d+\.\d+\.\d+)", go_version),
    (r"golang:(\d+\.\d+\.\d+)", go_version),
    (r"\[Fiber v([^\]]+)\]", fiber_version),
    (r"\[Fiber (?:HTML )?Template v([^\]]+)\]", template_version),
]
errors = []
checked = 0
for name in ("ARCHITECTURE.md", "DEPLOYMENT.md"):
    for path in sorted((root / "docs").glob("*/" + name)):
        checked += 1
        for line_number, line in enumerate(path.read_text().splitlines(), 1):
            for pattern, expected in rules:
                for match in re.finditer(pattern, line):
                    if match.group(1) != expected:
                        errors.append(f"{path.relative_to(root)}:{line_number}: {match.group(1)} != {expected}")
if errors:
    print("\n".join(errors))
    sys.exit(1)
print(f"Runtime version documentation: {checked} guides match go.mod")
