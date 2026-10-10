#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
source "$root/scripts/fedora.sh"

readonly backend_port=4005

# `stop`, not `down`: the compose file also holds the test fixtures other worktrees share, and the
# dev database keeps its data in an anonymous volume that `down` would drop.
stop_database() {
    docker compose -f "$root/docker-compose.yml" stop backend-db
}

# A stale backend keeps holding the port and answers with the previously built code; a new one that
# fails to bind only logs the error, so requests silently hit the old binary. Killing the listener
# also makes its parent `go run` exit.
free_backend_port() {
    local port_hex inode pid
    port_hex=$(printf ':%04X' "$backend_port")
    inode=$(awk -v p="$port_hex" '$4 == "0A" && $2 ~ p"$" {print $10}' /proc/net/tcp /proc/net/tcp6 | head -1)
    [[ -n "$inode" ]] || return 0
    for pid in $(ls /proc | grep -E '^[0-9]+$'); do
        if ls -l "/proc/$pid/fd" 2>/dev/null | grep -q "socket:\[$inode\]"; then
            echo "freeing port $backend_port: killing stale backend pid $pid"
            kill -9 "$pid" 2>/dev/null || true
        fi
    done
}

[[ -f "$root/.env" ]] || {
    echo "$root/.env is missing, copy .env.example to .env" >&2
    exit 1
}

apply_libedit_shim

trap stop_database EXIT
docker compose -f "$root/docker-compose.yml" up -d backend-db
timeout 60 bash -c 'until docker exec backend-db pg_isready -U postgres >/dev/null; do sleep 2; done'

(cd "$root/backend" && swag init -d . -g cmd/main.go -o swagger --quiet)
pnpm --dir "$root/frontend" install --frozen-lockfile
free_backend_port

cd "$root"
process-compose --config process-compose.yaml --no-server --disable-dotenv up "$@"
