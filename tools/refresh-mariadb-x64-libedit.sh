#!/usr/bin/env bash
# Rebuild assets/tools/x64/mariadb/mariadb-12.1/bin/ using the libedit-linked
# client from MariaDB's official APT repo. Mirrors how the arm tree is built.
#
# Run from the databasus repo root. Requires Docker with linux/amd64 support
# (works natively on amd64 hosts, or via qemu on arm hosts).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$REPO_ROOT/assets/tools/x64/mariadb/mariadb-12.1/bin"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

docker run --rm --platform linux/amd64 -v "$WORK:/out" debian:bookworm-slim bash -eu <<'INNER'
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y --no-install-recommends \
  ca-certificates curl gnupg lsb-release apt-transport-https >/dev/null
curl -fsSL https://r.mariadb.com/downloads/mariadb_repo_setup \
  | bash -s -- --mariadb-server-version=12.1 >/dev/null 2>&1
apt-get update -qq
apt-get install -y --no-install-recommends mariadb-client >/dev/null
cp /usr/bin/mariadb       /out/mariadb
cp /usr/bin/mariadb-dump  /out/mariadb-dump
chmod +x /out/mariadb /out/mariadb-dump
/out/mariadb --version
/out/mariadb-dump --version
INNER

install -m 0755 "$WORK/mariadb"      "$DEST/mariadb"
install -m 0755 "$WORK/mariadb-dump" "$DEST/mariadb-dump"

echo
echo "Installed into $DEST"
sha256sum "$DEST/mariadb" "$DEST/mariadb-dump"
"$DEST/mariadb" --version
