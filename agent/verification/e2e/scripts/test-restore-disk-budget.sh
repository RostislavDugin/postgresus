#!/bin/bash
# Disk-budget enforcement (real Docker): tiny maxContainerDiskMb (1 MB) against
# the standard postgres:16 init footprint (~50 MB) makes the disk watcher trip
# on its first poll. The agent must POST FAILED with failureKind:
# DISK_LIMIT_EXCEEDED and the container/anonymous PGDATA volume must be torn
# down. The watcher polls every 3 s so the verdict lands within ~10 s of spawn.
set -euo pipefail

source "$(dirname "$0")/lib.sh"

WORK="/tmp/agent-work-disk-budget"
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
  -d '{"path":"/artifacts/good-pg16.dump"}'

curl -sf -X POST "$MOCK/mock/set-claim" \
  -H 'Content-Type: application/json' \
  -d "{\"verificationId\":\"$VERIFICATION_ID\",\"backupId\":\"$BACKUP_ID\",\"backupSizeMb\":1,\"maxContainerDiskMb\":1,\"database\":{\"type\":\"POSTGRES_LOGICAL\",\"postgresqlLogical\":{\"version\":\"16\"}}}"

wait_for_report '"failureKind":"DISK_LIMIT_EXCEEDED"' 180 '"status":"COMPLETED"'

assert_report "$VERIFICATION_ID" '.status == "FAILED"'

echo "Disk-budget verdict OK: FAILED with failureKind:DISK_LIMIT_EXCEEDED"

stop_agent

if ! leak_check "$AGENT_ID"; then
  exit 1
fi

echo "Verification agent restore-disk-budget e2e passed"
