#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
workflow="$repo_root/.github/workflows/release.yml"
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

fail() {
  echo "$*" >&2
  exit 1
}

line_number() {
  local pattern=$1
  grep -nF -- "$pattern" "$workflow" | head -n 1 | cut -d: -f1
}

# Distinct tags never contend for one pending slot. Same-tag retries and the
# global alias reconciler both opt in to GitHub's supported 100-run queue.
grep -Fq "group: \${{ github.repository }}-release-\${{ github.event_name == 'workflow_dispatch' && inputs.release_tag || github.ref_name }}" "$workflow" \
  || fail "Release concurrency is not scoped by tag"
[[ $(grep -c 'queue: max' "$workflow") -eq 2 ]] \
  || fail "Expected queue:max for tag publication and alias reconciliation"
grep -Fq 'group: ${{ github.repository }}-release-aliases' "$workflow" \
  || fail "Mutable aliases do not have a dedicated concurrency group"

# Release notes must fail before the state machine reaches its first external
# write. The workflow must never delete an existing Release as an intermediate
# overwrite step.
notes_line=$(line_number 'name: Prepare and validate curated release notes')
immutability_line=$(line_number 'name: Resolve existing immutable image')
publish_boundary=$(line_number 'name: Attest release artifacts')
[[ -n "$notes_line" && -n "$publish_boundary" && "$notes_line" -lt "$publish_boundary" ]] \
  || fail "Release notes are not validated before external publication"
[[ -n "$immutability_line" && "$immutability_line" -lt "$publish_boundary" ]] \
  || fail "Image immutability is not checked before external publication"
if grep -Fq 'gh release delete' "$workflow"; then
  fail "Release overwrite still deletes the existing Release"
fi
if grep -Fq 'release delete' "$repo_root/.github/scripts/publish-github-release.sh"; then
  fail "Release publisher still deletes the existing Release"
fi

metadata=$(sed -n '/name: Extract metadata/,/name: Build amd64 image/p' "$workflow")
grep -Fq 'pattern={{version}}' <<< "$metadata" \
  || fail "Immutable full-version image tag is missing"
if grep -Eq 'pattern=\{\{major|value=latest' <<< "$metadata"; then
  fail "Mutable aliases are still published by the immutable image step"
fi

# A late older tag must reconcile from the published SemVer high-water mark,
# not from its own version or from queue completion order.
release_tags="$temp_dir/releases.txt"
printf '%s\n' v1.2.1 v1.3.0 v2.0.0 v2.1.0-rc.1 invalid > "$release_tags"
actual_plan="$temp_dir/actual-plan.tsv"
bash "$repo_root/.github/scripts/plan-release-aliases.sh" v1.2.0 "$release_tags" > "$actual_plan"
expected_plan="$temp_dir/expected-plan.tsv"
printf 'latest\tv2.0.0\n1\tv1.3.0\n1.2\tv1.2.1\n' > "$expected_plan"
cmp "$expected_plan" "$actual_plan" \
  || fail "A late older release can downgrade a mutable alias"

prerelease_plan="$temp_dir/prerelease-plan.tsv"
bash "$repo_root/.github/scripts/plan-release-aliases.sh" v2.1.0-rc.2 "$release_tags" > "$prerelease_plan"
[[ ! -s "$prerelease_plan" ]] \
  || fail "Prerelease unexpectedly updates stable aliases"

# Model both manual overwrite branches with a fake gh client. Existing
# Releases stay visible and are repaired with upload --clobber + edit.
fake_bin="$temp_dir/bin"
mkdir -p "$fake_bin" "$temp_dir/dist"
printf 'notes\n' > "$temp_dir/notes.md"
printf 'artifact\n' > "$temp_dir/dist/stargate-linux-amd64"
fake_gh="$fake_bin/gh"
printf '%s\n' \
  '#!/usr/bin/env bash' \
  'printf "%s " "$@" >> "$GH_TEST_LOG"' \
  'printf "\n" >> "$GH_TEST_LOG"' \
  'if [[ "$1 $2" == "release view" ]]; then exit "${GH_VIEW_STATUS:-0}"; fi' \
  'exit 0' > "$fake_gh"
chmod +x "$fake_gh"

existing_log="$temp_dir/existing.log"
PATH="$fake_bin:$PATH" GH_TEST_LOG="$existing_log" GH_VIEW_STATUS=0 \
  GITHUB_REPOSITORY=soulteary/stargate \
  bash "$repo_root/.github/scripts/publish-github-release.sh" \
    v1.2.0 "$temp_dir/notes.md" "$temp_dir/dist"
grep -Fq 'release upload' "$existing_log" || fail "Existing Release assets were not replaced"
grep -Fq -- '--clobber' "$existing_log" || fail "Existing Release upload is not idempotent"
grep -Fq 'release edit' "$existing_log" || fail "Existing Release metadata was not updated"
if grep -Fq 'release delete' "$existing_log"; then
  fail "Manual overwrite deleted the existing Release"
fi

missing_log="$temp_dir/missing.log"
PATH="$fake_bin:$PATH" GH_TEST_LOG="$missing_log" GH_VIEW_STATUS=1 \
  GITHUB_REPOSITORY=soulteary/stargate \
  bash "$repo_root/.github/scripts/publish-github-release.sh" \
    v1.2.0 "$temp_dir/notes.md" "$temp_dir/dist"
grep -Fq 'release create' "$missing_log" || fail "Missing Release was not created"

invalid_log="$temp_dir/invalid.log"
if PATH="$fake_bin:$PATH" GH_TEST_LOG="$invalid_log" GH_VIEW_STATUS=0 \
  GITHUB_REPOSITORY=soulteary/stargate \
  bash "$repo_root/.github/scripts/publish-github-release.sh" \
    v1.2.0 "$temp_dir/missing-notes.md" "$temp_dir/dist" >/dev/null 2>&1; then
  fail "Missing release notes unexpectedly reached publication"
fi
[[ ! -e "$invalid_log" ]] || fail "Validation failure modified an existing Release"

echo "Release workflow state-machine tests passed."
