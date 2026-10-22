#!/bin/bash
# Privileges e2e: checks what the kernel applied to the restore container, not
# what the engine reports back. The mock holds the backup stream for 30s, so
# the job's container is up and waiting for its backup while this script reads
# /proc/1/status through `docker exec`. The capability bounding set must hold
# exactly CHOWN, DAC_OVERRIDE, FOWNER, SETGID and SETUID (0xcb), and
# no-new-privileges must be set. CapEff is not checked: PID 1 is the Postgres
# server running as a non-root user, so it is zero under any configuration.
# The job must then finish COMPLETED and leave nothing behind.
set -euo pipefail

source "$(dirname "$0")/lib.sh"

WORK="/tmp/agent-work-container-privileges"
AGENT_ID="77777777-7777-7777-7777-777777777777"
VERIFICATION_ID="77777777-aaaa-aaaa-aaaa-777777777777"
BACKUP_ID="77777777-bbbb-bbbb-bbbb-777777777777"

EXPECTED_CAP_BND="00000000000000cb"
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

PROC_STATUS="$(docker exec "$CONTAINER_ID" cat /proc/1/status)"
CAP_BND="$(echo "$PROC_STATUS" | awk '/^CapBnd:/ {print $2}')"
NO_NEW_PRIVS="$(echo "$PROC_STATUS" | awk '/^NoNewPrivs:/ {print $2}')"

if [ "$CAP_BND" != "$EXPECTED_CAP_BND" ] || [ "$NO_NEW_PRIVS" != "1" ]; then
  echo "FAIL: restore container privileges: CapBnd=${CAP_BND} (want ${EXPECTED_CAP_BND}), NoNewPrivs=${NO_NEW_PRIVS} (want 1)"
  echo "$PROC_STATUS"
  stop_agent
  exit 1
fi

echo "Restore container privileges OK: CapBnd=${CAP_BND} NoNewPrivs=${NO_NEW_PRIVS}"

wait_for_report '"status":"COMPLETED"' 240 '"status":"FAILED"'

assert_report "$VERIFICATION_ID" '.pgRestoreExitCode == 0'

stop_agent

if ! leak_check "$AGENT_ID"; then
  echo "---- agent.out ----"
  cat agent.out
  exit 1
fi

echo "Verification agent container-privileges e2e passed"
