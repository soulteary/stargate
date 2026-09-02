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
for file in "$repo_root"/docs/*/API.md "$repo_root"/docs/*/CONFIG.md \
  "$repo_root"/docs/*/DEPLOYMENT.md "$repo_root"/docs/*/MIGRATION_V1.md \
  "$repo_root"/docs/*/SECURITY.md; do
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

# Cross-domain support in one process must not regress to a blanket Redis requirement.
perl -0pi -e 's/SESSION_STORAGE_ENABLED=false/SESSION_STORAGE_IN_MEMORY=unknown/' "$temp_dir/docs/enUS/DEPLOYMENT.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a process-local session storage scope failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/DEPLOYMENT.md

# The v1 environment must be separate and must omit both retired Warden OTP names.
perl -0pi -e 's/--env-file \.\/stargate-v1\.env/--env-file .\/stargate.env/' "$temp_dir/docs/enUS/DEPLOYMENT.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a separate v1 environment-file contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/DEPLOYMENT.md

perl -0pi -e 's/`WARDEN_OTP_SECRET_KEY`/`WARDEN_OTP_KEY`/' "$temp_dir/docs/enUS/MIGRATION_V1.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a retired Warden OTP removal contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/MIGRATION_V1.md

# Herald-backed TOTP also requires Warden user resolution.
perl -0pi -e 's/`WARDEN_URL`/`WARDEN_ENDPOINT`/' "$temp_dir/docs/enUS/MIGRATION_V1.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a Herald TOTP Warden prerequisite contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/MIGRATION_V1.md

# Migration guidance may name retired variables, but must never make them active again.
printf '\n```bash\nWARDEN_OTP_ENABLED=true\nWARDEN_OTP_SECRET_KEY=unsafe\n```\n' \
  >> "$temp_dir/docs/enUS/DEPLOYMENT.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an active retired Warden OTP assignment failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/DEPLOYMENT.md

# Compose mapping syntax is also an active environment assignment.
printf '\n```yaml\nenvironment:\n  WARDEN_OTP_ENABLED: "true"\n  WARDEN_OTP_SECRET_KEY: "unsafe"\n```\n' \
  >> "$temp_dir/docs/enUS/MIGRATION_V1.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a retired Warden OTP mapping assignment failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/MIGRATION_V1.md

# A valueless export can forward a retired value inherited from the parent shell.
printf '\n```bash\nexport WARDEN_OTP_ENABLED\n```\n' \
  >> "$temp_dir/docs/enUS/MIGRATION_V1.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a valueless retired Warden OTP export failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/MIGRATION_V1.md

# Execute the documented transformation and verify both filtering and no-clobber behavior.
env_test_dir="$temp_dir/env-migration-test"
mkdir "$env_test_dir"
migration_script=$(perl -0ne 'print $1 if /```bash\n(set -eu\nold_env=\.\/stargate\.env.*?\n)```/s' \
  "$temp_dir/docs/enUS/MIGRATION_V1.md")
if [[ -z "$migration_script" ]]; then
  echo "Could not extract the documented environment migration commands" >&2
  exit 1
fi
printf '%s\n' \
  'KEEP_SETTING=kept' \
  'WARDEN_OTP_ENABLED' \
  'WARDEN_OTP_SECRET_KEY' \
  'WARDEN_OTP_ENABLED=false' \
  'WARDEN_OTP_SECRET_KEY=retired-secret' \
  'PORT=80' \
  > "$env_test_dir/stargate.env"
chmod 644 "$env_test_dir/stargate.env"
(cd "$env_test_dir" && sh -c "$migration_script")
cmp "$env_test_dir/stargate.env" "$env_test_dir/stargate-v0.12.0.env"
grep -q '^KEEP_SETTING=kept$' "$env_test_dir/stargate-v1.env"
grep -q '^PORT=80$' "$env_test_dir/stargate-v0.12.0.env"
grep -q '^PORT=8080$' "$env_test_dir/stargate-v1.env"
if grep -q '^PORT=80$' "$env_test_dir/stargate-v1.env"; then
  echo "Documented v1 environment retained the legacy container port" >&2
  exit 1
fi
if grep -q '^WARDEN_OTP_\(ENABLED\|SECRET_KEY\)\($\|=\)' "$env_test_dir/stargate-v1.env"; then
  echo "Documented v1 environment retained a retired Warden OTP setting" >&2
  exit 1
fi
for file in stargate.env stargate-v0.12.0.env stargate-v1.env; do
  if [[ "$(stat -c '%a' "$env_test_dir/$file")" != "600" ]]; then
    echo "Documented environment migration left $file with non-private permissions" >&2
    exit 1
  fi
done
rollback_checksum=$(cksum "$env_test_dir/stargate-v0.12.0.env")
v1_checksum=$(cksum "$env_test_dir/stargate-v1.env")
if (cd "$env_test_dir" && sh -c "$migration_script" >/dev/null 2>&1); then
  echo "Documented environment migration unexpectedly overwrote existing files" >&2
  exit 1
fi
[[ "$rollback_checksum" == "$(cksum "$env_test_dir/stargate-v0.12.0.env")" ]]
[[ "$v1_checksum" == "$(cksum "$env_test_dir/stargate-v1.env")" ]]

# A deployment without an existing env file must be recoverable from the old
# container without exposing secrets or leaving a partial output on failure.
export_test_dir="$temp_dir/env-export-test"
mkdir -p "$export_test_dir/bin"
printf '%s\n' \
  '#!/bin/sh' \
  'printf "%s\\n" KEEP_SETTING=from-container WARDEN_OTP_ENABLED=true PORT=80' \
  > "$export_test_dir/bin/docker"
chmod +x "$export_test_dir/bin/docker"
(cd "$export_test_dir" && PATH="$export_test_dir/bin:$PATH" sh -c "$migration_script")
cmp "$export_test_dir/stargate.env" "$export_test_dir/stargate-v0.12.0.env"
grep -q '^KEEP_SETTING=from-container$' "$export_test_dir/stargate-v1.env"
grep -q '^PORT=8080$' "$export_test_dir/stargate-v1.env"
if grep -q '^WARDEN_OTP_ENABLED\($\|=\)' "$export_test_dir/stargate-v1.env"; then
  echo "Exported v1 environment retained a retired Warden OTP setting" >&2
  exit 1
fi
for file in stargate.env stargate-v0.12.0.env stargate-v1.env; do
  if [[ "$(stat -c '%a' "$export_test_dir/$file")" != "600" ]]; then
    echo "Exported environment migration left $file with non-private permissions" >&2
    exit 1
  fi
done

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

# CommonMark fenced blocks allow up to three leading spaces after list or
# block-quote container prefixes, either marker character, and a closing
# marker at least as long as the opening marker.
fence_fixture="$temp_dir/fence-structure-test.md"
printf '%s\n' \
  '   ```bash' \
  'echo valid' \
  '   ```' \
  '~~~text' \
  'valid' \
  '~~~~' \
  'inline ``` markers are not fences' \
  '    ```four-space-indented-code' \
  '1. outer list' \
  '   - nested item' \
  '' \
  '     ```json' \
  '     {}' \
  '     ```' \
  '> ~~~text' \
  '> quoted content' \
  '> ~~~' \
  '- > ```json' \
  '  > {}' \
  '  > ```' \
  '> - ~~~text' \
  '>   quoted list content' \
  '>   ~~~' \
  '-   paragraph in a wide list item' \
  'lazy paragraph continuation' \
  '    ```text' \
  '    fenced content after lazy continuation' \
  '    ```' \
  'paragraph before a non-list marker' \
  '2. this remains paragraph text' \
  '   ```text' \
  'unindented root fence content' \
  '   ```' \
  > "$fence_fixture"
if ! (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected valid CommonMark fence forms to pass" >&2
  exit 1
fi

printf '%s\n' '   ```bash' 'echo unclosed' > "$fence_fixture"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an indented unclosed fence failure" >&2
  exit 1
fi

printf '%s\n' '```text' 'wrong marker' '~~~' > "$fence_fixture"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a mismatched fence-marker failure" >&2
  exit 1
fi

printf '%s\n' '````text' 'short close' '```' > "$fence_fixture"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a short closing-fence failure" >&2
  exit 1
fi

printf '%s\n' \
  '1. outer list' \
  '   - nested item' \
  '' \
  '     ```json' \
  '     {}' \
  > "$fence_fixture"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unclosed nested-list fence failure" >&2
  exit 1
fi

# A later top-level delimiter must not close a fence whose list container has
# already ended.
printf '%s\n' \
  '1. list item' \
  '   ```text' \
  '   missing close' \
  'top-level paragraph ends the list' \
  '   ```' \
  > "$fence_fixture"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a fence container-exit failure" >&2
  exit 1
fi

# Container markers can interleave. A quote opened after a list marker must be
# part of the fence context rather than hiding an unclosed fence.
printf '%s\n' \
  '- > ```json' \
  '  > {}' \
  > "$fence_fixture"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unclosed interleaved-container fence failure" >&2
  exit 1
fi

# A lazy paragraph continuation retains its list container, including the
# wider content column established by marker padding.
printf '%s\n' \
  '-   paragraph in a wide list item' \
  'lazy paragraph continuation' \
  '    ```text' \
  '    missing close' \
  > "$fence_fixture"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unclosed fence after a lazy list continuation failure" >&2
  exit 1
fi

# Exercise the same five-space list-container indentation used by the existing
# localized API response examples, so those real blocks cannot silently regress.
perl -0pi -e 's/^     ```\r?\n//m' "$temp_dir/docs/enUS/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a localized nested-list fence failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/API.md
rm -f "$fence_fixture"

echo "Documentation contract self-tests passed."
