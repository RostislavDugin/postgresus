#!/bin/bash
# Privileges e2e: checks what the kernel applied to the restore container, not
# what the engine reports back. The mock holds the backup stream for 30s, so
# the job's container is up and waiting for its backup while this script reads
# /proc/1/status through `docker exec`. PID 1 must run as a non-root user, and
# the capability bounding set must be empty, so even a process that reaches
# user ID 0 inside the container holds no capabilities. CapEff is not checked:
# for a non-root process it is zero under any configuration.
# The job must then finish COMPLETED and leave nothing behind.
set -euo pipefail

source "$(dirname "$0")/lib.sh"

WORK="/tmp/agent-work-container-privileges"
AGENT_ID="77777777-7777-7777-7777-777777777777"
VERIFICATION_ID="77777777-aaaa-aaaa-aaaa-777777777777"
BACKUP_ID="77777777-bbbb-bbbb-bbbb-777777777777"

EXPECTED_CAP_BND="0000000000000000"
STREAM_DELAY_SECONDS=30

rm -rf "$WORK"
mkdir -p "$WORK"
cd "$WORK"

reset_mock_state
reset_mock_version

cp "$ARTIFACTS/agent-v1" ./databasus-verification-agent
chmod +x ./databasus-verification-agent

start_agent "$AGENT_ID"

curl -sf -X POST "$MOCK/mock/set-stream-delay" \
  -H 'Content-Type: application/json' \
  -d "{\"seconds\":${STREAM_DELAY_SECONDS}}"

curl -sf -X POST "$MOCK/mock/set-backup-fixture" \
  -H 'Content-Type: application/json' \
  -d '{"path":"/artifacts/good-pg18.dump"}'

curl -sf -X POST "$MOCK/mock/set-claim" \
  -H 'Content-Type: application/json' \
  -d "{\"verificationId\":\"$VERIFICATION_ID\",\"backupId\":\"$BACKUP_ID\",\"backupSizeMb\":1,\"maxContainerDiskMb\":2048,\"database\":{\"type\":\"POSTGRES_LOGICAL\",\"postgresqlLogical\":{\"version\":\"18\"}}}"

CONTAINER_ID=""
deadline=$((SECONDS + 120))
while [ $SECONDS -lt $deadline ]; do
  CONTAINER_ID="$(docker ps --filter "label=databasus.verification.agent_id=${AGENT_ID}" \
    --filter status=running --format '{{.ID}}' | head -n1)"
  if [ -n "$CONTAINER_ID" ]; then
    break
  fi
  sleep 1
done

if [ -z "$CONTAINER_ID" ]; then
  echo "FAIL: no running restore container labeled with agent ${AGENT_ID} within 120s"
  dump_diagnostics
  stop_agent
  exit 1
fi

if ! PROC_STATUS="$(docker exec "$CONTAINER_ID" cat /proc/1/status)"; then
  echo "FAIL: could not read /proc/1/status of restore container ${CONTAINER_ID}"
  dump_diagnostics
  stop_agent
  exit 1
fi
CAP_BND="$(echo "$PROC_STATUS" | awk '/^CapBnd:/ {print $2}')"
UID_REAL="$(echo "$PROC_STATUS" | awk '/^Uid:/ {print $2}')"

if [ -z "$UID_REAL" ] || [ "$UID_REAL" = "0" ] || [ "$CAP_BND" != "$EXPECTED_CAP_BND" ]; then
  echo "FAIL: restore container privileges: Uid=${UID_REAL} (want non-zero), CapBnd=${CAP_BND} (want ${EXPECTED_CAP_BND})"
  echo "$PROC_STATUS"
  stop_agent
  exit 1
fi

echo "Restore container privileges OK: Uid=${UID_REAL} CapBnd=${CAP_BND}"

wait_for_report '"status":"COMPLETED"' 240 '"status":"FAILED"'

assert_report "$VERIFICATION_ID" '.pgRestoreExitCode == 0'

stop_agent

if ! leak_check "$AGENT_ID"; then
  echo "---- agent.out ----"
  cat agent.out
  exit 1
fi

echo "Verification agent container-privileges e2e passed"
