#!/bin/bash
# Restore e2e (correct dump) for one Postgres major: mock serves the matching
# -Fc archive of a tiny seeded schema; the agent must run pg_restore inside
# the spawned postgres:N container, then the verifier (out-of-container pgx
# via the container's published host port) must collect tier stats, and the
# agent must POST COMPLETED with non-null dbSizeBytesAfterRestore /
# tableCount / pgRestoreExitCode==0 / non-empty tableStats. The PG major is
# the first positional arg (default 16); run-all.sh loops it over 12..18.
set -euo pipefail

source "$(dirname "$0")/lib.sh"

PG_VERSION="${1:-16}"

WORK="/tmp/agent-work-restore-success-pg${PG_VERSION}"
AGENT_ID="44444444-4444-4444-4444-444444444444"
VERIFICATION_ID="44444444-aaaa-aaaa-aaaa-4444444444${PG_VERSION}"
BACKUP_ID="44444444-bbbb-bbbb-bbbb-4444444444${PG_VERSION}"

rm -rf "$WORK"
mkdir -p "$WORK"
cd "$WORK"

reset_mock_state
reset_mock_version

cp "$ARTIFACTS/agent-v1" ./databasus-verification-agent
chmod +x ./databasus-verification-agent

start_agent "$AGENT_ID"

curl -sf -X POST "$MOCK/mock/set-backup-fixture" \
  -H 'Content-Type: application/json' \
  -d "{\"path\":\"/artifacts/good-pg${PG_VERSION}.dump\"}"

curl -sf -X POST "$MOCK/mock/set-claim" \
  -H 'Content-Type: application/json' \
  -d "{\"verificationId\":\"$VERIFICATION_ID\",\"backupId\":\"$BACKUP_ID\",\"backupSizeMb\":1,\"maxContainerDiskMb\":2048,\"database\":{\"type\":\"POSTGRES_LOGICAL\",\"postgresqlLogical\":{\"version\":\"${PG_VERSION}\"}}}"

wait_for_report '"status":"COMPLETED"' 240 '"status":"FAILED"'

assert_report "$VERIFICATION_ID" '.pgRestoreExitCode == 0'
assert_report "$VERIFICATION_ID" '.dbSizeBytesAfterRestore > 0'
assert_report "$VERIFICATION_ID" '.tableCount == 2'
assert_report "$VERIFICATION_ID" '.schemaCount == 1'
assert_report "$VERIFICATION_ID" '(.tableStats | map(.name) | sort) == ["t_a","t_b"]'

echo "Success report OK (pg${PG_VERSION}): COMPLETED with tableCount=2 schemaCount=1 t_a+t_b present"

stop_agent

if ! leak_check "$AGENT_ID"; then
  echo "---- agent.out ----"
  cat agent.out
  exit 1
fi

echo "Verification agent restore-success e2e (pg${PG_VERSION}) passed"
