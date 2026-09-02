#!/usr/bin/env bash
set -euo pipefail

tag=${1:?usage: prepare-release-notes.sh TAG OUTPUT [CHANGELOG]}
output=${2:?usage: prepare-release-notes.sh TAG OUTPUT [CHANGELOG]}
changelog=${3:-CHANGELOG.md}
allow_existing=${ALLOW_EXISTING_RELEASE_NOTES:-false}
script_dir=$(dirname "$0")
diagnostics=$(mktemp)
temp_file=""
trap 'rm -f "$diagnostics" "$temp_file"' EXIT

if bash "$script_dir/extract-release-notes.sh" "$tag" "$output" "$changelog" 2> "$diagnostics"; then
  exit 0
fi

if [[ "$allow_existing" != "true" ]]; then
  cat "$diagnostics" >&2
  echo "Release-note validation failed for $tag" >&2
  exit 1
fi

repo=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required for release-note fallback}
temp_file=$(mktemp)

# Historical tags can predate the dated CHANGELOG contract. Manual republish
# may preserve notes from an already-published Release, but it must never invent
# notes for a new release or weaken the automatic tag-push gate.
if ! gh release view "$tag" --repo "$repo" --json body --jq .body > "$temp_file"; then
  echo "No existing GitHub Release notes are available for $tag" >&2
  exit 1
fi
if ! grep -Eq '[^[:space:]]' "$temp_file" || grep -Fxq 'null' "$temp_file"; then
  echo "Existing GitHub Release notes are empty for $tag" >&2
  exit 1
fi

mv "$temp_file" "$output"
echo "Reusing existing GitHub Release notes for historical tag $tag."
