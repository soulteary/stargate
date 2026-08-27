#!/usr/bin/env bash
set -euo pipefail

tag=${1:?usage: extract-release-notes.sh TAG OUTPUT [CHANGELOG]}
output=${2:?usage: extract-release-notes.sh TAG OUTPUT [CHANGELOG]}
changelog=${3:-CHANGELOG.md}
repository=${GITHUB_REPOSITORY:-soulteary/stargate}

version=${tag#v}
base_version=${version%%-*}
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "Unsupported release tag: $tag" >&2
  exit 1
fi

temp_file=$(mktemp)
trap 'rm -f "$temp_file"' EXIT

awk -v version="$base_version" '
  index($0, "## [" version "] - ") == 1 {
    found = 1
    printing = 1
  }
  printing && found && $0 ~ /^## \[/ && index($0, "## [" version "] - ") != 1 {
    exit
  }
  printing && $0 == "### Release verification" {
    exit
  }
  printing && $0 ~ /^\[[0-9]+\.[0-9]+\.[0-9]+\]:/ {
    exit
  }
  printing { print }
  END {
    if (!found) {
      exit 2
    }
  }
' "$changelog" > "$temp_file" || {
  echo "CHANGELOG.md has no section for $base_version" >&2
  exit 1
}

header=$(head -n 1 "$temp_file")
if [[ "$version" != *-* && "$header" == *Unreleased* ]]; then
  echo "Formal release $tag requires a dated CHANGELOG entry" >&2
  exit 1
fi

if [[ "$version" == *-* ]]; then
  sed -i "1s/.*/## [$version] - Prerelease/" "$temp_file"
fi

sed "s#](docs/enUS/MIGRATION_V1.md)#](https://github.com/${repository}/blob/${tag}/docs/enUS/MIGRATION_V1.md)#" \
  "$temp_file" > "$output"

printf '\nUpgrade instructions: [v1.0.0 migration guide](https://github.com/%s/blob/%s/docs/enUS/MIGRATION_V1.md).\n' \
  "$repository" "$tag" >> "$output"
