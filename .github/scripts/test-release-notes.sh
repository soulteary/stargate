#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

rc_notes="$temp_dir/rc.md"
GITHUB_REPOSITORY=soulteary/stargate \
  bash "$repo_root/.github/scripts/extract-release-notes.sh" v1.0.0-rc.1 "$rc_notes" "$repo_root/CHANGELOG.md"
grep -Fq '## [1.0.0-rc.1] - Prerelease' "$rc_notes"
grep -Fq '### Breaking changes' "$rc_notes"
if grep -Fq '[1.0.0]:' "$rc_notes"; then
  echo "Release notes unexpectedly contain changelog reference definitions" >&2
  exit 1
fi

if bash "$repo_root/.github/scripts/extract-release-notes.sh" v1.0.0 "$temp_dir/formal.md" "$repo_root/CHANGELOG.md" >/dev/null 2>&1; then
  echo "Formal release unexpectedly accepted an Unreleased changelog entry" >&2
  exit 1
fi

dated_changelog="$temp_dir/CHANGELOG.md"
sed 's/## \[1.0.0\] - Unreleased/## [1.0.0] - 2026-08-27/' "$repo_root/CHANGELOG.md" > "$dated_changelog"
bash "$repo_root/.github/scripts/extract-release-notes.sh" v1.0.0 "$temp_dir/formal.md" "$dated_changelog"
grep -Fq '## [1.0.0] - 2026-08-27' "$temp_dir/formal.md"

if bash "$repo_root/.github/scripts/extract-release-notes.sh" v9.9.9 "$temp_dir/missing.md" "$repo_root/CHANGELOG.md" >/dev/null 2>&1; then
  echo "Missing changelog version unexpectedly succeeded" >&2
  exit 1
fi

echo "Release-note extraction tests passed."
