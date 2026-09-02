#!/usr/bin/env bash
set -euo pipefail

tag=${1:?usage: publish-github-release.sh TAG NOTES_FILE DIST_DIR}
notes=${2:?usage: publish-github-release.sh TAG NOTES_FILE DIST_DIR}
dist=${3:?usage: publish-github-release.sh TAG NOTES_FILE DIST_DIR}
repo=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}

if [[ ! -s "$notes" ]]; then
  echo "Release notes are missing or empty: $notes" >&2
  exit 1
fi

shopt -s nullglob
assets=("$dist"/*)
if (( ${#assets[@]} == 0 )); then
  echo "Release assets are missing: $dist" >&2
  exit 1
fi

metadata=(--repo "$repo" --title "Release $tag" --notes-file "$notes" --latest=false)
if [[ "$tag" == *-* ]]; then
  prerelease=(--prerelease=true)
else
  prerelease=(--prerelease=false)
fi

if gh release view "$tag" --repo "$repo" >/dev/null 2>&1; then
  declare -A wanted_assets=()
  for asset in "${assets[@]}"; do
    wanted_assets["$(basename "$asset")"]=1
  done
  existing_assets_text=$(
    gh release view "$tag" --repo "$repo" --json assets --jq '.assets[].name'
  )
  mapfile -t existing_assets <<< "$existing_assets_text"

  # Keep the existing Release visible while clobbering its assets. In
  # particular, never delete a good Release before its replacement is ready.
  gh release upload "$tag" "${assets[@]}" --repo "$repo" --clobber
  for asset_name in "${existing_assets[@]}"; do
    if [[ -n "$asset_name" && ! -v "wanted_assets[$asset_name]" ]]; then
      gh release delete-asset "$tag" "$asset_name" --repo "$repo" --yes
    fi
  done
  gh release edit "$tag" "${metadata[@]}" "${prerelease[@]}" --draft=false
else
  gh release create "$tag" "${assets[@]}" "${metadata[@]}" "${prerelease[@]}" --verify-tag
fi
