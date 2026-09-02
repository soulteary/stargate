#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

git -C "$repo_root" archive HEAD | tar -x -C "$temp_dir"
# The test script is commonly run before its own changes are committed. Copy
# the working-tree checker so the self-test cannot accidentally exercise the
# old checker from HEAD and report a false pass.
cp "$repo_root/.github/scripts/check-doc-contracts.sh" "$temp_dir/.github/scripts/check-doc-contracts.sh"
cp "$repo_root/CHANGELOG.md" "$temp_dir/CHANGELOG.md"
for file in "$repo_root"/docs/*/CONFIG.md "$repo_root"/docs/*/SECURITY.md; do
  relative=${file#"$repo_root"/}
  cp "$file" "$temp_dir/$relative"
done
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

# Merely mentioning a route in prose must not replace its API section heading.
perl -0pi -e 's/^### `GET \/healthz`$/The `GET \/healthz` route is available./m' "$temp_dir/docs/enUS/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a route-heading contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/API.md

# Password-generation examples must never put a password in htpasswd argv.
printf '\n```bash\nhtpasswd -bnBC 10 "" password\n```\n' >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd option contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Both high-impact upgrade requirements must remain in the scoped v1 breaking list.
perl -0pi -e 's/^- The official container now listens on port .*\n//m' "$temp_dir/CHANGELOG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a container-port breaking-change contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- CHANGELOG.md

perl -0pi -e 's/^- `Stargate-Password` request-header authentication .*\n//m' "$temp_dir/CHANGELOG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a password-header breaking-change contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- CHANGELOG.md

# TOTP fields must occur in the confirmation section, not merely somewhere in the file.
perl -0pi -e 's/(^### `POST \/totp\/enroll\/confirm`.*?)(`enroll_id`)/$1`enrollment_token`/ms' "$temp_dir/docs/enUS/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a section-specific TOTP contract failure" >&2
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
