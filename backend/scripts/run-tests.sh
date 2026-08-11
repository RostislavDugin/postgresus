#!/usr/bin/env bash
set -euo pipefail

# Wait for a TCP endpoint given as host:port. Uses bash /dev/tcp so we make
# no assumptions about what tools third-party images ship with.
wait_for_tcp() {
  local addr="$1"
  local host="${addr%%:*}"
  local port="${addr##*:}"
  local attempts=60

  echo "waiting for ${host}:${port}"
  for _ in $(seq 1 "${attempts}"); do
    if (echo > "/dev/tcp/${host}/${port}") 2>/dev/null; then
      echo "${host}:${port} is up"
      return 0
    fi
    sleep 1
  done

  echo "timed out waiting for ${host}:${port}" >&2
  return 1
}

for addr in \
  "${TEST_MINIO_ADDR}" \
  "${TEST_AZURITE_ADDR}" \
  "${TEST_NAS_ADDR}" \
  "${TEST_FTP_ADDR}"; do
  wait_for_tcp "${addr}"
done

echo "running migrations"
cd /app/backend/migrations
goose up
cd /app/backend

echo "running tests"
exec go test -p=1 -count=1 "$@" ./internal/...
