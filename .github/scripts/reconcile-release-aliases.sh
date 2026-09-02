#!/usr/bin/env bash
set -euo pipefail

tag=${1:?usage: reconcile-release-aliases.sh TAG IMAGE}
image=${2:?usage: reconcile-release-aliases.sh TAG IMAGE}
repo=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}

temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT
release_tags="$temp_dir/release-tags.txt"
plan="$temp_dir/alias-plan.tsv"

gh api --paginate "repos/${repo}/releases?per_page=100" \
  --jq '.[] | select(.draft == false and .prerelease == false) | .tag_name' \
  > "$release_tags"

bash "$(dirname "$0")/plan-release-aliases.sh" "$tag" "$release_tags" > "$plan"
if [[ ! -s "$plan" ]]; then
  echo "Prerelease $tag keeps only its immutable full-version image."
  exit 0
fi

latest_release=""
while IFS=$'\t' read -r alias source_tag; do
  source_ref="${image}:${source_tag#v}"
  alias_ref="${image}:${alias}"
  digest=$(docker buildx imagetools inspect "$source_ref" | awk '$1 == "Digest:" { print $2; exit }')
  if [[ ! "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
    echo "Unable to resolve an immutable digest for $source_ref" >&2
    exit 1
  fi
  docker buildx imagetools create --tag "$alias_ref" "${image}@${digest}"
  if [[ "$alias" == "latest" ]]; then
    latest_release=$source_tag
  fi
done < "$plan"

# Keep GitHub's Latest Release marker consistent with the image alias. Every
# reconciler computes the high-water SemVer, so completion order cannot cause
# an older release to win.
gh release edit "$latest_release" --repo "$repo" --latest
