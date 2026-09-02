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

if bash "$repo_root/.github/scripts/extract-release-notes.sh" v1.0.0 "$temp_dir/formal.md" "$repo_root/CHANGELOG.md"; then
  grep -Fq '## [1.0.0] - 2026-08-27' "$temp_dir/formal.md"
else
  echo "Formal release unexpectedly rejected a dated changelog entry" >&2
  exit 1
fi

unreleased_changelog="$temp_dir/unreleased-CHANGELOG.md"
sed 's/## \[1.0.0\] - 2026-08-27/## [1.0.0] - Unreleased/' "$repo_root/CHANGELOG.md" > "$unreleased_changelog"
if bash "$repo_root/.github/scripts/extract-release-notes.sh" v1.0.0 "$temp_dir/unreleased.md" "$unreleased_changelog" >/dev/null 2>&1; then
  echo "Formal release unexpectedly accepted an Unreleased changelog entry" >&2
  exit 1
fi
if bash "$repo_root/.github/scripts/extract-release-notes.sh" v1.0.0-rc.2 "$temp_dir/unreleased-rc.md" "$unreleased_changelog" >/dev/null 2>&1; then
  echo "Prerelease unexpectedly accepted an Unreleased changelog entry" >&2
  exit 1
fi

malformed_changelog="$temp_dir/malformed-CHANGELOG.md"
sed 's/## \[1.0.0\] - 2026-08-27/## [1.0.0] - 27 August 2026/' "$repo_root/CHANGELOG.md" > "$malformed_changelog"
if bash "$repo_root/.github/scripts/extract-release-notes.sh" v1.0.0 "$temp_dir/malformed.md" "$malformed_changelog" >/dev/null 2>&1; then
  echo "Malformed changelog heading unexpectedly succeeded" >&2
  exit 1
fi

if bash "$repo_root/.github/scripts/extract-release-notes.sh" v9.9.9 "$temp_dir/missing.md" "$repo_root/CHANGELOG.md" >/dev/null 2>&1; then
  echo "Missing changelog version unexpectedly succeeded" >&2
  exit 1
fi

if bash "$repo_root/.github/scripts/extract-release-notes.sh" v01.0.0 "$temp_dir/invalid-tag.md" "$repo_root/CHANGELOG.md" >/dev/null 2>&1; then
  echo "Invalid SemVer tag unexpectedly succeeded" >&2
  exit 1
fi

echo "Release-note extraction tests passed."
