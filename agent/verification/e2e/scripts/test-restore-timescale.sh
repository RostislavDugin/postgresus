#!/bin/bash
# TimescaleDB restore e2e: the mock serves a hypertable -Fc dump and a claim
# whose timescaledbVersion is set, so the agent must spawn a version-matched
# timescale/timescaledb:<ver>-pg<major> container, run timescaledb_pre_restore /
# timescaledb_post_restore around a single-threaded pg_restore, then the
# verifier must collect stats and the agent must POST COMPLETED with exit 0.
# Without the hooks (or with parallel -j) the _timescaledb_catalog restore fails
# and the agent would report FAILED instead.
set -euo pipefail

source "$(dirname "$0")/lib.sh"

PG_VERSION="17"
TS_VERSION="2.17.0"

WORK="/tmp/agent-work-restore-timescale"
AGENT_ID="55555555-5555-5555-5555-555555555555"
VERIFICATION_ID="55555555-aaaa-aaaa-aaaa-555555555555"
BACKUP_ID="55555555-bbbb-bbbb-bbbb-555555555555"

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
  -d '{"path":"/artifacts/good-timescale-pg17.dump"}'

curl -sf -X POST "$MOCK/mock/set-claim" \
  -H 'Content-Type: application/json' \
  -d "{\"verificationId\":\"$VERIFICATION_ID\",\"backupId\":\"$BACKUP_ID\",\"backupSizeMb\":1,\"maxContainerDiskMb\":4096,\"timescaledbVersion\":\"${TS_VERSION}\",\"database\":{\"type\":\"POSTGRES_LOGICAL\",\"postgresqlLogical\":{\"version\":\"${PG_VERSION}\"}}}"

wait_for_report '"status":"COMPLETED"' 300 '"status":"FAILED"'

assert_report "$VERIFICATION_ID" '.pgRestoreExitCode == 0'
assert_report "$VERIFICATION_ID" '.dbSizeBytesAfterRestore > 0'
# The hypertable's parent table is restored into public...
assert_report "$VERIFICATION_ID" '(.tableStats | map(.name) | any(. == "sensor_data"))'
# ...and its data lands in _timescaledb_internal chunks, proving the catalog +
# chunk restore (the part that fails without the pre/post hooks) succeeded.
assert_report "$VERIFICATION_ID" '(.tableStats | map(select(.name | startswith("_hyper_")) | .rowCount) | add) > 0'

echo "TimescaleDB restore report OK: COMPLETED with sensor_data hypertable + chunk rows restored"

stop_agent

if ! leak_check "$AGENT_ID"; then
  echo "---- agent.out ----"
  cat agent.out
  exit 1
fi

echo "Verification agent restore-timescale e2e passed"
