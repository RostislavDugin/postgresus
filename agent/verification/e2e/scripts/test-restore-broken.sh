#!/bin/bash
# Restore e2e (broken dump): mock serves a file that is NOT a valid -Fc
# archive; pg_restore -Fc inside the spawned Postgres container must reject
# it and exit non-zero, and the agent must POST FAILED with a real
# pgRestoreExitCode. Independent of the verifier-conn fix — the failure
# happens at restore, never reaches the verifier.
set -euo pipefail

source "$(dirname "$0")/lib.sh"

WORK="/tmp/agent-work-restore-broken"
AGENT_ID="33333333-3333-3333-3333-333333333333"
VERIFICATION_ID="33333333-aaaa-aaaa-aaaa-333333333333"
BACKUP_ID="33333333-bbbb-bbbb-bbbb-333333333333"

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
  -d '{"path":"/artifacts/broken.dump"}'

curl -sf -X POST "$MOCK/mock/set-claim" \
  -H 'Content-Type: application/json' \
  -d "{\"verificationId\":\"$VERIFICATION_ID\",\"backupId\":\"$BACKUP_ID\",\"backupSizeMb\":1,\"maxContainerDiskMb\":2048,\"database\":{\"type\":\"POSTGRES_LOGICAL\",\"postgresqlLogical\":{\"version\":\"16\"}}}"

# Spawn + image pull (first run) + restore + report. Generous budget on cold cache.
wait_for_report '"status":"FAILED"' 240

assert_report "$VERIFICATION_ID" '.pgRestoreExitCode >= 1'

echo "Broken-dump report OK: FAILED with non-zero pgRestoreExitCode"

stop_agent

if ! leak_check "$AGENT_ID"; then
  echo "---- agent.out ----"
  cat agent.out
  exit 1
fi

echo "Verification agent restore-broken e2e passed"
