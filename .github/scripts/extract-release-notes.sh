#!/usr/bin/env bash
set -euo pipefail

tag=${1:?usage: extract-release-notes.sh TAG OUTPUT [CHANGELOG]}
output=${2:?usage: extract-release-notes.sh TAG OUTPUT [CHANGELOG]}
changelog=${3:-CHANGELOG.md}
repository=${GITHUB_REPOSITORY:-soulteary/stargate}

if [[ ! "$tag" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$ ]]; then
  echo "Unsupported release tag: $tag" >&2
  exit 1
fi

version=${tag#v}
base_version=${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.${BASH_REMATCH[3]}

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
if [[ ! "$header" =~ ^##\ \[$base_version\]\ -\ [0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
  echo "Release $tag requires an exact dated CHANGELOG heading for $base_version" >&2
  exit 1
fi

if ! tail -n +2 "$temp_file" | grep -Eq '^[[:space:]]*[^[:space:]]'; then
  echo "CHANGELOG.md section for $base_version is empty" >&2
  exit 1
fi

if [[ "$version" == *-* ]]; then
  sed -i "1s/.*/## [$version] - Prerelease/" "$temp_file"
fi

sed "s#](docs/enUS/MIGRATION_V1.md)#](https://github.com/${repository}/blob/${tag}/docs/enUS/MIGRATION_V1.md)#" \
  "$temp_file" > "$output"

printf '\nUpgrade instructions: [v1.0.0 migration guide](https://github.com/%s/blob/%s/docs/enUS/MIGRATION_V1.md).\n' \
  "$repository" "$tag" >> "$output"
