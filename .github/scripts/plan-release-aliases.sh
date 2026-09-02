#!/usr/bin/env bash
set -euo pipefail

tag=${1:?usage: plan-release-aliases.sh TAG RELEASE_TAGS_FILE}
release_tags=${2:?usage: plan-release-aliases.sh TAG RELEASE_TAGS_FILE}

semver='(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)'
if [[ ! "$tag" =~ ^v${semver}(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$ ]]; then
  echo "Unsupported release tag: $tag" >&2
  exit 1
fi

# Prereleases always keep their full version image, but never move stable
# aliases. This also keeps a late RC from changing latest after a GA release.
if [[ "$tag" == *-* ]]; then
  exit 0
fi

major=${BASH_REMATCH[1]}
minor=${BASH_REMATCH[2]}

temp_file=$(mktemp)
trap 'rm -f "$temp_file"' EXIT

# Only published, stable vX.Y.Z releases are candidates. Include the release
# that triggered reconciliation to tolerate short API visibility delays.
while IFS= read -r candidate; do
  if [[ "$candidate" =~ ^v${semver}$ ]]; then
    printf '%s\n' "$candidate"
  fi
done < "$release_tags" > "$temp_file"
printf '%s\n' "$tag" >> "$temp_file"
sort -Vu -o "$temp_file" "$temp_file"

latest=$(tail -n 1 "$temp_file")
major_latest=$(awk -v prefix="v${major}." 'index($0, prefix) == 1' "$temp_file" | tail -n 1)
minor_latest=$(awk -v prefix="v${major}.${minor}." 'index($0, prefix) == 1' "$temp_file" | tail -n 1)

printf 'latest\t%s\n' "$latest"
printf '%s\t%s\n' "$major" "$major_latest"
printf '%s.%s\t%s\n' "$major" "$minor" "$minor_latest"
