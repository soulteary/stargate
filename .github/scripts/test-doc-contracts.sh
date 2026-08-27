#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

git -C "$repo_root" archive HEAD | tar -x -C "$temp_dir"
git -C "$temp_dir" init -q
git -C "$temp_dir" config user.name "doc-contract-test"
git -C "$temp_dir" config user.email "doc-contract-test@example.invalid"
git -C "$temp_dir" add .
git -C "$temp_dir" commit -qm "test baseline"
base_sha=$(git -C "$temp_dir" rev-parse HEAD)

# An exact route check must not let /healthz satisfy the /health contract.
perl -0pi -e 's/`GET \/health`/`GET \/healthz`/' "$temp_dir/docs/enUS/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an exact-route contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/API.md

# Configuration checks must work in both directions.
printf '\n| `REMOVED_RUNTIME_SETTING` | string | - | test-only invalid setting |\n' >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a removed runtime setting contract failure" >&2
  exit 1
fi

echo "Documentation contract self-tests passed."
