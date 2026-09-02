#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

git -C "$repo_root" archive HEAD | tar -x -C "$temp_dir"
# The test script is commonly run before its own changes are committed. Copy
# the working-tree checker and the files covered by network contracts so the
# self-test cannot accidentally exercise stale content from HEAD.
cp "$repo_root/.github/scripts/check-doc-contracts.sh" "$temp_dir/.github/scripts/check-doc-contracts.sh"
cp "$repo_root/docker-compose.yml" "$temp_dir/docker-compose.yml"
for file in "$repo_root"/docs/*/DEPLOYMENT.md; do
  locale=$(basename "$(dirname "$file")")
  cp "$file" "$temp_dir/docs/$locale/DEPLOYMENT.md"
done
git -C "$temp_dir" init -q
git -C "$temp_dir" config user.name "doc-contract-test"
git -C "$temp_dir" config user.email "doc-contract-test@example.invalid"
git -C "$temp_dir" add .
git -C "$temp_dir" commit -qm "test baseline"
base_sha=$(git -C "$temp_dir" rev-parse HEAD)

# An exact route check must not let /healthz satisfy the /health contract.
perl -0pi -e 's/`GET \/health`/`GET \/healthz`/' "$temp_dir/docs/enUS/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an exact-route contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/API.md

# Merely mentioning a route in prose must not replace its API section heading.
perl -0pi -e 's/^### `GET \/healthz`$/The `GET \/healthz` route is available./m' "$temp_dir/docs/enUS/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a route-heading contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/API.md

# TOTP fields must occur in the confirmation section, not merely somewhere in the file.
perl -0pi -e 's/(^### `POST \/totp\/enroll\/confirm`.*?)(`enroll_id`)/$1`enrollment_token`/ms' "$temp_dir/docs/enUS/API.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a section-specific TOTP contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/API.md

# Configuration checks must work in both directions.
printf '\n| `REMOVED_RUNTIME_SETTING` | string | - | test-only invalid setting |\n' >> "$temp_dir/docs/enUS/CONFIG.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a removed runtime setting contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/CONFIG.md

# The bundled Compose network key and Traefik labels must resolve to the same name.
perl -0pi -e 's/traefik\.docker\.network=stargate-traefik/traefik.docker.network=traefik/' "$temp_dir/docker-compose.yml"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a bundled Compose network contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docker-compose.yml

# Every localized deployment guide must create a new external network with the trusted subnet.
perl -0pi -e 's/--subnet 172\.30\.0\.0\/24/--subnet 172.31.0.0\/24/' "$temp_dir/docs/jaJP/DEPLOYMENT.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected a localized external-network subnet contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/jaJP/DEPLOYMENT.md

# Existing external networks must be inspected rather than silently recreated.
perl -0pi -e 's/docker network inspect "\$TRAEFIK_NETWORK_NAME"/docker network show "\$TRAEFIK_NETWORK_NAME"/g' "$temp_dir/docs/enUS/DEPLOYMENT.md"
if (cd "$temp_dir" && bash .github/scripts/check-doc-contracts.sh "$base_sha" >/dev/null 2>&1); then
  echo "Expected an existing external-network inspection contract failure" >&2
  exit 1
fi
git -C "$temp_dir" checkout -q -- docs/enUS/DEPLOYMENT.md

echo "Documentation contract self-tests passed."
