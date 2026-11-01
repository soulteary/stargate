#!/usr/bin/env bash
set -euo pipefail
if [ "$#" -ne 2 ]; then
  echo 'usage: test-release-image.sh IMAGE EXPECTED_BINARY' >&2
  exit 2
fi
image=$1
expected_binary=$2
test -f "$expected_binary"
expected=$(sha256sum "$expected_binary" | awk '{print $1}')
actual=$(docker run --rm --network none --entrypoint sha256sum "$image" /bin/stargate | awk '{print $1}')
test "$actual" = "$expected"
test "$(docker image inspect --format '{{.Config.User}}' "$image")" = '10001:10001'
docker run --rm --network none --entrypoint sh "$image" -c \
  'test -r /app/web/templates/login.html && test -r /app/web/templates/login.warden.html'
container_id=''
cleanup() {
  if [ -n "$container_id" ]; then docker rm -f "$container_id" >/dev/null; fi
}
trap cleanup EXIT
container_id=$(docker run --detach --network none --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=32m --cap-drop ALL \
  --security-opt no-new-privileges \
  -e AUTH_HOST=auth.example.invalid -e PASSWORDS=plaintext:ci-only-package-check "$image")
for attempt in $(seq 1 30); do
  if docker exec "$container_id" wget -q -O - http://127.0.0.1:8080/healthz >/dev/null 2>&1; then
    echo 'Release image binary, non-root user, templates and health verified.'
    exit 0
  fi
  sleep 1
done
docker logs "$container_id"
echo 'Release image did not become healthy.' >&2
exit 1
