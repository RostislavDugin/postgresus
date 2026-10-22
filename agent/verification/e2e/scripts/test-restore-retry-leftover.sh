#!/bin/bash
# Retry over a leftover network (issue #832): the backend requeues a failed
# attempt under the same verification ID, and the agent names the attempt's
# network after that ID. A network the earlier attempt left behind used to
# fail every retry with "network with name databasus-verif-<id> already
# exists". Seed that leftover after the startup purge has run, so only the
# cleanup before the attempt can remove it, then assert the restore completes
# and nothing leaks.
set -euo pipefail

source "$(dirname "$0")/lib.sh"

PG_VERSION=16

WORK="/tmp/agent-work-restore-retry-leftover"
AGENT_ID="77777777-7777-7777-7777-777777777777"
VERIFICATION_ID="77777777-aaaa-aaaa-aaaa-777777777777"
BACKUP_ID="77777777-bbbb-bbbb-bbbb-777777777777"
LEFTOVER_NETWORK="databasus-verif-${VERIFICATION_ID}"

cleanup() {
  docker network rm "$LEFTOVER_NETWORK" >/dev/null 2>&1 || true
}
trap cleanup EXIT

rm -rf "$WORK"
mkdir -p "$WORK"
cd "$WORK"

reset_mock_state
reset_mock_version

cleanup

cp "$ARTIFACTS/agent-v1" ./databasus-verification-agent
chmod +x ./databasus-verification-agent

start_agent "$AGENT_ID"

# The heartbeat loop starts only after the startup purge returns, so the first
# heartbeat means the purge can no longer remove the seeded network.
STARTED=0
for _ in $(seq 1 30); do
  if [ "$(hb_count)" -ge 1 ] 2>/dev/null; then
    STARTED=1
    break
  fi
  sleep 1
done

if [ "$STARTED" -ne 1 ]; then
  echo "FAIL: agent sent no heartbeat within 30s"
  dump_diagnostics
  stop_agent
  exit 1
fi

docker network create \
  --label "databasus.verification.agent_id=${AGENT_ID}" \
  --label "databasus.verification.verification_id=${VERIFICATION_ID}" \
  "$LEFTOVER_NETWORK" >/dev/null

if ! docker network ls --filter "name=${LEFTOVER_NETWORK}" --format '{{.Name}}' | grep -qx "$LEFTOVER_NETWORK"; then
  echo "FAIL: seeded leftover network $LEFTOVER_NETWORK not present"
  stop_agent
  exit 1
fi
echo "Seeded leftover network $LEFTOVER_NETWORK after the startup purge"

curl -sf -X POST "$MOCK/mock/set-backup-fixture" \
  -H 'Content-Type: application/json' \
  -d "{\"path\":\"/artifacts/good-pg${PG_VERSION}.dump\"}"

curl -sf -X POST "$MOCK/mock/set-claim" \
  -H 'Content-Type: application/json' \
  -d "{\"verificationId\":\"$VERIFICATION_ID\",\"backupId\":\"$BACKUP_ID\",\"backupSizeMb\":1,\"maxContainerDiskMb\":2048,\"database\":{\"type\":\"POSTGRES_LOGICAL\",\"postgresqlLogical\":{\"version\":\"${PG_VERSION}\"}}}"

wait_for_report '"status":"COMPLETED"' 240 '"status":"FAILED"'

assert_report "$VERIFICATION_ID" '.pgRestoreExitCode == 0'

echo "Retry report OK: COMPLETED despite the leftover network"

stop_agent

if ! leak_check "$AGENT_ID"; then
  echo "---- agent.out ----"
  cat agent.out
  exit 1
fi

echo "Verification agent restore-retry-leftover e2e passed"
