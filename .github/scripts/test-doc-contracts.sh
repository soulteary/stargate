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

# A CommonMark code span can cross a physical line. Its normalized contents
# still form one displayed command and must be scanned as one context.
printf '%s\n' \
  '' \
  '`htpasswd -bn ""' \
  'password`' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe multiline inline htpasswd command failure" >&2
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

# The same wrapper grammar applies to unfenced standalone commands. Wrapper
# operands must not prevent the line from reaching the deeper shell parser.
printf '%s\n' \
  '' \
  'sudo -u root htpasswd -bn "" password' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe standalone command after a sudo option failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Capturing a wrapper-prefixed standalone line must not treat a mention passed
# as data to an unrelated command as an htpasswd invocation.
printf '%s\n' \
  '' \
  'sudo printf "%s\\n" "do not use htpasswd -b"' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if ! (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected wrapper command data mentioning htpasswd to pass" >&2
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

# Path-qualified wrappers retain their wrapper semantics and delegate to the
# following command just like an unqualified env invocation.
printf '%s\n' \
  '' \
  '```bash' \
  '/usr/bin/env htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command after path-qualified env failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# stdbuf delegates to the command following its buffering options. Both an
# attached MODE and the separately supplied short-option operand are valid.
printf '%s\n' \
  '' \
  '```bash' \
  'stdbuf -o0 htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command delegated through stdbuf failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

printf '%s\n' \
  '' \
  '```bash' \
  '/usr/bin/stdbuf -o 0 htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command after a stdbuf option operand failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Brace-group delimiters establish a new simple-command position.
printf '%s\n' \
  '' \
  '```bash' \
  '{ htpasswd -bn "" password; }' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command inside a brace group failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Bash coproc is a reserved-word prefix for an asynchronously launched command.
printf '%s\n' \
  '' \
  '```bash' \
  'coproc htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command launched through coproc failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# A named coprocess precedes a compound command; brace segmentation still
# exposes the command executed inside that compound command.
printf '%s\n' \
  '' \
  '```bash' \
  'coproc WORKER { htpasswd -bn "" password; }' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command inside a named coprocess failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Array-assignment elements are stored as data rather than executed as a
# simple command, even when they look like an unsafe htpasswd argv.
printf '%s\n' \
  '' \
  '```bash' \
  'args=(htpasswd -C 10 -bn "" password)' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if ! (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a Bash array literal containing htpasswd text to pass" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# A command substitution inside an array element still executes and remains
# subject to the batch-password contract.
printf '%s\n' \
  '' \
  '```bash' \
  'args=("$(htpasswd -bn "" password)")' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd substitution inside an array failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# The shell `time` keyword accepts options before the pipeline command.
printf '%s\n' \
  '' \
  '```bash' \
  'time -p htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe timed htpasswd command failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Arithmetic left shifts use the same characters as a heredoc operator but do
# not start a heredoc body or hide commands on following lines.
printf '%s\n' \
  '' \
  '```bash' \
  'echo $((1 << 2))' \
  'htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command after an arithmetic shift failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Arithmetic expansions may span physical lines; the parser must retain that
# state when the shift operator itself appears on a continuation line.
printf '%s\n' \
  '' \
  '```bash' \
  'echo $((1' \
  '  << 2))' \
  'htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command after a multiline arithmetic shift failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Bash still supports the legacy `$[...]` arithmetic form. A shift inside it
# is not a heredoc declaration and cannot hide a following command.
printf '%s\n' \
  '' \
  '```bash' \
  'echo $[1 << 2]' \
  'htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command after a legacy arithmetic shift failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Substring offsets and lengths inside parameter expansion are arithmetic
# expressions, so their shifts are not heredoc declarations.
printf '%s\n' \
  '' \
  '```bash' \
  'value=abcdef' \
  'echo ${value:1 << 2}' \
  'htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command after a parameter offset failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Indexed-array subscripts are arithmetic contexts too. Their left shifts are
# not heredoc operators and cannot consume a following documented command.
printf '%s\n' \
  '' \
  '```bash' \
  'arr[1 << 2]=x' \
  'htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command after an array-subscript shift failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Bracketed text which is not an assignment word does not gain array-subscript
# arithmetic semantics. Its `<<` remains a real heredoc and the later text is
# payload rather than an executed htpasswd command.
printf '%s\n' \
  '' \
  '```bash' \
  'echo arr[1 << 2]' \
  'htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if ! (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a real heredoc after bracketed command data to pass" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Quote state spans physical lines. A `<<` inside the continued single-quoted
# argument is literal data and must not consume the following real command.
printf '%s\n' \
  '' \
  '```bash' \
  "printf '%s\\n' 'not a heredoc" \
  "<<EOF'" \
  'htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe command after a multiline quoted string failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# xargs executes its command operand with both initial and input-derived
# arguments; a wrapped htpasswd invocation must therefore be inspected.
printf '%s\n' \
  '' \
  '```bash' \
  'printf password | xargs -n 1 htpasswd -bn ""' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command delegated through xargs failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Short xargs flags may be bundled before an option whose value is supplied
# in the next argv token; that operand is not the delegated command.
printf '%s\n' \
  '' \
  '```bash' \
  'printf password | xargs -rn 1 htpasswd -bn ""' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command after bundled xargs options failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# The GNU long forms of the deprecated eof/replace switches take only an
# optional attached value. Without `=VALUE`, the next token is the command.
printf '%s\n' \
  '' \
  '```bash' \
  'printf password | xargs --eof htpasswd -bn ""' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command after optional xargs eof failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# GNU env splits `-S`/`--split-string` values into the argv it executes.
printf '%s\n' \
  '' \
  '```bash' \
  "env -S 'htpasswd -bn \"\" password'" \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command in an env split string failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

printf '%s\n' \
  '' \
  '```bash' \
  "sudo env --split-string='htpasswd -C 10 -bn \"\" password'" \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command in a wrapped env split string failure" >&2
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

# An unquoted heredoc body is data, but its command substitutions still run.
printf '%s\n' \
  '' \
  '```bash' \
  'cat <<DOC' \
  '$(htpasswd -bn "" password)' \
  'DOC' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd substitution in an expanding heredoc failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Heredocs inside command substitutions must be filtered after extracting the
# substitution body. Literal payload text is data, not another command.
printf '%s\n' \
  '' \
  '```bash' \
  'echo "$(cat <<EOF' \
  'htpasswd -C 10 -bn "" password' \
  'EOF' \
  ')"' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if ! (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a heredoc payload inside command substitution to pass" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# An expanding nested heredoc still executes substitutions in its payload.
printf '%s\n' \
  '' \
  '```bash' \
  'echo "$(cat <<EOF' \
  '$(htpasswd -bn "" password)' \
  'EOF' \
  ')"' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe substitution in a nested heredoc failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Process substitutions execute their own command bodies, but they remain one
# word in the outer argv. A redirection target must not split later options
# away from the htpasswd command that receives them.
printf '%s\n' \
  '' \
  '```bash' \
  'htpasswd -C 10 > >(cat) -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd option after process substitution failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# The process-substitution body is an independently executable context too.
printf '%s\n' \
  '' \
  '```bash' \
  'cat <(htpasswd -bn "" password)' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe command inside process substitution failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Unquoted command substitutions likewise remain one outer argv word.
printf '%s\n' \
  '' \
  '```bash' \
  'htpasswd -C 10 $(printf file) -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe option after unquoted command substitution failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Shell `-c` operands are themselves executable command contexts, including
# when the interpreter is reached through another supported wrapper.
printf '%s\n' \
  '' \
  '```bash' \
  "bash -c 'htpasswd -bn \"\" password'" \
  "sudo -u root /bin/sh -c 'htpasswd -C 10 -bn \"\" password'" \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd shell command-string failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# eval joins its argv and executes the result as shell input.
printf '%s\n' \
  '' \
  '```bash' \
  "eval 'htpasswd -bn \"\" password'" \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command string passed to eval failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Bash eval accepts `--` before its command-string arguments.
printf '%s\n' \
  '' \
  '```bash' \
  "eval -- 'htpasswd -bn \"\" password'" \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command after eval double dash failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# timeout consumes its own options and duration before delegating a command.
printf '%s\n' \
  '' \
  '```bash' \
  'timeout --signal TERM 5 htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command delegated through timeout failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# nice consumes its adjustment option before delegating the remaining argv.
printf '%s\n' \
  '' \
  '```bash' \
  'nice -n 5 htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command delegated through nice failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# GNU find actions execute argv up to their `;` or `+` terminator.
printf '%s\n' \
  '' \
  '```bash' \
  'find . -maxdepth 0 -exec htpasswd -bn {} password \;' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command delegated through find failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

printf '%s\n' \
  '' \
  '```bash' \
  "find . -maxdepth 0 -execdir sh -c 'htpasswd -C 10 -bn \"\" password' \\;" \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe shell command delegated through find execdir failure" >&2
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

# Console prompts commonly include the current user, host, and directory.
printf '%s\n' \
  '' \
  '```terminal' \
  'root@host:~# htpasswd -bn "" password' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe htpasswd command at a hostname prompt failure" >&2
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
  "bash safe-script.sh -c 'htpasswd -bn \"\" password'" \
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

# An unprompted line immediately following an explicit backslash continuation
# is still part of the prompted command, not transcript output.
printf '%s\n' \
  '' \
  '```console' \
  '$ htpasswd -C 10 \' \
  '  -bn "" password' \
  'created password entry' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an unsafe continued console command failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# Heredoc payload is not a sequence of shell commands. A quoted delimiter also
# disables substitutions, while direct text remains data with either form.
printf '%s\n' \
  '' \
  '```bash' \
  "cat <<'LITERAL'" \
  'htpasswd -bn "" password' \
  '$(htpasswd -bn "" password)' \
  'LITERAL' \
  'cat <<PLAIN' \
  'htpasswd -bn "" password' \
  'PLAIN' \
  'printf "%s\n" {htpasswd,-bn}' \
  '```' \
  >> "$temp_dir/docs/enUS/CONFIG.md"
if ! (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected literal heredoc and brace-expansion text to pass" >&2
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
