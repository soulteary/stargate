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

# Batch mode must also be rejected when options with arguments precede -b.
printf '\n```bash\nhtpasswd -C 10 -bn "" password\n```\n' >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a reordered unsafe htpasswd option contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# A shell line continuation must not hide a later batch-password option.
printf '\n```bash\nhtpasswd -C 10 \\\n  -bn "" password\n```\n' >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a continued unsafe htpasswd option contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Quoting an option does not change the argv value received by htpasswd.
printf '\n```bash\nhtpasswd -C 10 "-bn" "" password\n```\n' >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a quoted unsafe htpasswd option contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Backslashes are literal inside single quotes, including immediately before
# the closing quote; a later parse failure must not hide an earlier -b option.
printf '%s\n' \
  '' \
  '```bash' \
  "htpasswd -bn \"\" password > 'output\\'" \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd option before a quoted backslash failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Unsafe invocations in explicit inline-code contexts remain protected.
printf '%s\n' '' 'Never run `htpasswd -bn "" password`.' >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe inline htpasswd command contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# File-descriptor and combined-output redirections stay inside the command;
# neither form may hide a batch-password option which follows it.
printf '%s\n' \
  '' \
  '```bash' \
  'htpasswd -C 10 2>&1 -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd option after descriptor redirection failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

printf '%s\n' \
  '' \
  '```bash' \
  'htpasswd -C 10 &>/tmp/htpasswd.log -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd option after combined redirection failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Shell grammar prefixes and leading redirections do not change the command
# word; each form must still expose htpasswd options to the contract check.
printf '%s\n' \
  '' \
  '```bash' \
  'if htpasswd -bn "" password; then echo ok; fi' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd option after if failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

printf '%s\n' \
  '' \
  '```bash' \
  '! htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd option after negation failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

printf '%s\n' \
  '' \
  '```bash' \
  '2>/dev/null htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd option after leading redirection failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Wrapper options which take a separate operand precede, but do not replace,
# the wrapped command word.
printf '%s\n' \
  '' \
  '```bash' \
  'sudo -u root htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command after a sudo user option failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

printf '%s\n' \
  '' \
  '```bash' \
  'env -u OLD_VALUE htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command after an env unset option failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Backtick substitutions execute in both unquoted and double-quoted contexts.
printf '%s\n' \
  '' \
  '```bash' \
  'echo `htpasswd -bn "" password`' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd backtick substitution failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

printf '%s\n' \
  '' \
  '```bash' \
  'echo "result: `htpasswd -bn \"\" password`"' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe quoted htpasswd backtick substitution failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Modern command substitutions are executable even inside double quotes.
printf '%s\n' \
  '' \
  '```bash' \
  'echo "$(htpasswd -bn "" password)"' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe modern htpasswd command substitution failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# A hash prompt in console-like fences represents an executed root command.
printf '%s\n' \
  '' \
  '```console' \
  '# htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command at a console root prompt failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# An option mentioned after a Markdown or shell command boundary does not
# belong to the preceding safe htpasswd invocation and must not be a violation.
printf '%s\n' \
  '' \
  'Run `htpasswd -nBC 10 stargate`; never add `-b` to that command.' \
  '```bash' \
  'htpasswd -nBC 10 stargate && printf "%s\\n" "-b"' \
  'htpasswd -nBC 10 stargate | sed -n "-b"' \
  'htpasswd -nBC 10 stargate; printf "%s\\n" "-b"' \
  'htpasswd -nBC 10 stargate & printf "%s\\n" "-b"' \
  'htpasswd -nBC 10 stargate # never add -b' \
  'htpasswd -nBC 10 stargate > -b' \
  '# Do not invoke htpasswd with -b because that exposes the password.' \
  'printf "%s\\n" "htpasswd -b is unsafe"' \
  'if printf "%s\\n" "htpasswd -b"; then echo ok; fi' \
  '```' \
  'Do not invoke htpasswd with -b because that exposes the password.' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if ! (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected safe htpasswd command-boundary examples to pass" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Wrapper operands, literal backticks in single quotes, and comments in a
# non-console fence are not executed htpasswd command positions.
printf '%s\n' \
  '' \
  '```bash' \
  'sudo -u htpasswd printf "%s\n" "-b"' \
  "printf '%s\\n' '\`htpasswd -bn \"\" password\`'" \
  '```' \
  '```' \
  '# htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if ! (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected non-command htpasswd wrapper, quoting, and comment forms to pass" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Once a console transcript uses prompts, unprompted lines are output rather
# than shell commands. Modern substitutions in a single-quoted string are
# literal as well.
printf '%s\n' \
  '' \
  '```console' \
  '$ printf "%s\n" "htpasswd -C 10 -bn is unsafe"' \
  'htpasswd -C 10 -bn is unsafe' \
  '```' \
  '```bash' \
  "printf '%s\\n' '\$(htpasswd -bn \"\" password)'" \
  '# `htpasswd -bn "" password` is not executed in a comment' \
  '# $(htpasswd -bn "" password) is not executed in a comment' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if ! (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected console output and literal command substitutions to pass" >&2
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

echo "Documentation contract self-tests passed."
