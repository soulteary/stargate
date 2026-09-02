#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

git -C "$repo_root" archive HEAD | tar -x -C "$temp_dir"
# The test script is commonly run before its own changes are committed. Copy
# the working-tree checker and every file covered by self-tests so they cannot
# accidentally exercise stale content from HEAD and report a false pass.
cp "$repo_root/.github/scripts/check-doc-contracts.sh" "$temp_dir/.github/scripts/check-doc-contracts.sh"
cp "$repo_root/CHANGELOG.md" "$temp_dir/CHANGELOG.md"
cp "$repo_root/docker-compose.yml" "$temp_dir/docker-compose.yml"
for file in "$repo_root"/docs/*/API.md "$repo_root"/docs/*/CONFIG.md "$repo_root"/docs/*/SECURITY.md "$repo_root"/docs/*/DEPLOYMENT.md; do
  relative=${file#"$repo_root"/}
  cp "$file" "$temp_dir/$relative"
done
git -C "$temp_dir" init -q
git -C "$temp_dir" config user.name "doc-contract-test"
git -C "$temp_dir" config user.email "doc-contract-test@example.invalid"
git -C "$temp_dir" add .
git -C "$temp_dir" commit -qm "test baseline"
base_sha=$(git -C "$temp_dir" rev-parse HEAD)

# A failing baseline would make every negative mutation below look successful.
(cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null)

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

# Both browser-supported form encodings must remain documented in every locale.
perl -0pi -e 's/(^### `POST \/_send_verify_code`.*?)(`multipart\/form-data`)/$1`unsupported-form-type`/ms' "$temp_dir/docs/deDE/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a localized send-form encoding contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/deDE/API.md

perl -0pi -e 's/(^### `POST \/totp\/enroll\/confirm`.*?)(`multipart\/form-data`)/$1`unsupported-form-type`/ms' "$temp_dir/docs/frFR/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a localized TOTP form encoding contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/frFR/API.md

# TOTP confirmation must reject application/json request bodies.
perl -0pi -e 's/(^### `POST \/totp\/enroll\/confirm`.*?\| `application\/json` \|) ❌ \|/$1 ✅ |/ms' "$temp_dir/docs/frFR/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a TOTP confirmation application/json contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/frFR/API.md

# The send matrix must remain under the marked request-body heading, not another subsection.
perl -0pi -e 's/<!-- api-contract: send-verify-code-request-body -->\n//; s/(^### `POST \/_send_verify_code`.*?)(^#### Response)/$1<!-- api-contract: send-verify-code-request-body -->\n$2/ms' "$temp_dir/docs/enUS/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a misplaced send request-body contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/API.md

# Warden fields must be documented as table rows, not only appear in examples.
perl -0pi -e 's/^\| `challenge_id` .*\n//m' "$temp_dir/docs/itIT/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a localized Warden field contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/itIT/API.md

# The Warden TOTP variant needs its own conditional fields and executable example.
perl -0pi -e 's/^\| `otp_code` .*\n//m' "$temp_dir/docs/frFR/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a localized Warden TOTP field contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/frFR/API.md

perl -0pi -e 's/(auth_method=warden&mail=user\@example\.com&)use_otp=true&otp_code=123456/$1use_otp=false&missing_otp_code=123456/' "$temp_dir/docs/jaJP/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a localized Warden TOTP example contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/jaJP/API.md

# Both Warden identifiers may be supplied together; do not regress to exactly-one validation.
perl -0pi -e 's/(^### `POST \/_login`.*?)(`phone` \+ `mail`)/$1`phone` only/ms' "$temp_dir/docs/deDE/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a combined Warden identifier contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/deDE/API.md

# A supported-JSON claim in the request-body matrix must fail the form contract.
perl -0pi -e 's/(^### `POST \/_send_verify_code`.*?\| `application\/json` \|) ❌ \|/$1 ✅ |/ms' "$temp_dir/docs/koKR/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a supported application/json request contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/koKR/API.md

# Explicitly naming an application/json response must not fail the request-body contract.
perl -0pi -e 's/(^### `POST \/_send_verify_code`.*?\*\*Success Response \(200 OK\)\*\*)/$1\n\nResponse media type: `application\/json`./ms' "$temp_dir/docs/enUS/API.md"
if ! (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an application/json response mention to remain valid" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/API.md

# DingTalk delivery must remain explicitly opt-in.
perl -0pi -e 's/(^### `POST \/_send_verify_code`.*?)(`deliver_via=dingtalk`)/$1`deliver_via=implicit`/ms' "$temp_dir/docs/jaJP/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an explicit DingTalk delivery contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/jaJP/API.md

# Revoke requires an authenticated session plus password or TOTP reauthentication.
perl -0pi -e 's/(^### `POST \/totp\/revoke`.*?)(`401 Unauthorized`)/$1`authentication failure`/ms' "$temp_dir/docs/koKR/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a localized TOTP reauthentication contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/koKR/API.md

# Configuration checks must work in both directions.
printf '\n| `REMOVED_RUNTIME_SETTING` | string | - | test-only invalid setting |\n' >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a removed runtime setting contract failure" >&2
  exit 1
fi

git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Preserve the legacy logical key so Compose can reuse an existing named network.
perl -0pi -e 's/^  traefik:\n    name: stargate-traefik$/  stargate-traefik:\n    name: stargate-traefik/m' "$temp_dir/docker-compose.yml"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a bundled Compose logical-key compatibility failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docker-compose.yml

# Traefik must select the actual Docker network name, not the logical Compose key.
perl -0pi -e 's/traefik\.docker\.network=stargate-traefik/traefik.docker.network=traefik/' "$temp_dir/docker-compose.yml"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a bundled Traefik label network failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docker-compose.yml

# Every localized deployment guide must create a new external network with the trusted subnet.
perl -0pi -e 's/--subnet 172\.30\.0\.0\/24/--subnet 172.31.0.0\/24/' "$temp_dir/docs/jaJP/DEPLOYMENT.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a localized external-network subnet contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/jaJP/DEPLOYMENT.md

# Existing external networks must be inspected rather than silently recreated.
perl -0pi -e 's/docker network inspect "\$TRAEFIK_NETWORK_NAME"/docker network show "\$TRAEFIK_NETWORK_NAME"/g' "$temp_dir/docs/enUS/DEPLOYMENT.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an existing external-network inspection contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/DEPLOYMENT.md

echo "Documentation contract self-tests passed."
