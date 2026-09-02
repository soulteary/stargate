#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
launcher="$repo_root/start-local.sh"
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

fail() {
  echo "$*" >&2
  exit 1
}

assert_contains() {
  local output=$1
  local expected=$2
  local description=$3
  grep -Fq -- "$expected" <<< "$output" || fail "$description: missing '$expected'"
}

bash -n "$launcher"
bash -n "$repo_root/.github/scripts/test-start-local.sh"

fake_bin="$temp_dir/bin"
mkdir -p "$fake_bin"
fake_go="$fake_bin/go"
printf '%s\n' \
  '#!/usr/bin/env bash' \
  'printf "EXPORTED_PORT=%s\n" "${PORT-}"' \
  'printf "EXPORTED_AUTH_HOST=%s\n" "${AUTH_HOST-}"' \
  'printf "EXPORTED_CALLBACK_ALLOWED_HOSTS=%s\n" "${CALLBACK_ALLOWED_HOSTS-}"' \
  > "$fake_go"
chmod +x "$fake_go"

run_case() {
  local description=$1
  local expected_port=$2
  local expected_host_port=$3
  shift 3

  local output
  if ! output=$(cd "$repo_root" && env -u PORT -u AUTH_HOST -u CALLBACK_ALLOWED_HOSTS \
      PATH="$fake_bin:$PATH" "$@" 2>&1); then
    fail "$description: launcher unexpectedly failed: $output"
  fi

  assert_contains "$output" "EXPORTED_PORT=$expected_port" "$description"
  assert_contains "$output" "EXPORTED_AUTH_HOST=localhost:$expected_host_port" "$description"
  assert_contains "$output" "EXPORTED_CALLBACK_ALLOWED_HOSTS=localhost:$expected_host_port" "$description"
  assert_contains "$output" "AUTH_HOST: localhost:$expected_host_port" "$description"
  assert_contains "$output" \
    "http://localhost:$expected_host_port/_login?callback=localhost:$expected_host_port" \
    "$description"

  if grep -Fq "localhost::$expected_host_port" <<< "$output"; then
    fail "$description: output contains a double-colon host"
  fi
}

assert_invalid() {
  local description=$1
  local expected=$2
  shift 2

  local output
  if output=$(cd "$repo_root" && env -u PORT -u AUTH_HOST -u CALLBACK_ALLOWED_HOSTS \
      PATH="$fake_bin:$PATH" "$@" 2>&1); then
    fail "$description: invalid port unexpectedly succeeded"
  fi
  assert_contains "$output" "$expected" "$description"
}

# Empty PORT keeps the documented default.
run_case "empty environment port" "8080" "8080" env PORT= bash "$launcher"

# Both supported service-listener forms produce the same URL/host port.
run_case "numeric environment port" "8080" "8080" env PORT=8080 bash "$launcher"
run_case "colon-prefixed environment port" ":8080" "8080" env PORT=:8080 bash "$launcher"

# CLI values override the environment and preserve the selected listener form.
run_case "numeric CLI port" "8080" "8080" env PORT=:9090 bash "$launcher" --port 8080
run_case "colon-prefixed CLI port" ":8080" "8080" env PORT=9090 bash "$launcher" --port :8080

# Invalid inputs fail before the launcher can generate or export malformed hosts.
assert_invalid "non-numeric environment port" "错误: 无效端口: nope" \
  env PORT=nope bash "$launcher"
assert_invalid "out-of-range CLI port" "错误: 无效端口: :65536" \
  bash "$launcher" --port :65536
assert_invalid "missing CLI port" "错误: --port 需要端口值" \
  bash "$launcher" --port

echo "Local launcher port tests passed."
