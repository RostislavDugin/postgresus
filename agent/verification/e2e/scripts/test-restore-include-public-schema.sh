#!/bin/bash
# Restore e2e for an include-schemas backup (issue #726): the archive was dumped with -n public, so
# it carries CREATE SCHEMA public, and the spawned target already owns that schema from initdb. The
# agent must still report COMPLETED with pgRestoreExitCode==0 — --clean --if-exists drops the target
# schema before the archive recreates it.
set -euo pipefail

source "$(dirname "$0")/lib.sh"

PG_VERSION=18

WORK="/tmp/agent-work-restore-include-public-schema"
AGENT_ID="49494949-4949-4949-4949-494949494949"
VERIFICATION_ID="49494949-aaaa-aaaa-aaaa-494949494949"
BACKUP_ID="49494949-bbbb-bbbb-bbbb-494949494949"

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
  -d "{\"path\":\"/artifacts/good-pg${PG_VERSION}-public-schema.dump\"}"

curl -sf -X POST "$MOCK/mock/set-claim" \
  -H 'Content-Type: application/json' \
  -d "{\"verificationId\":\"$VERIFICATION_ID\",\"backupId\":\"$BACKUP_ID\",\"backupSizeMb\":1,\"maxContainerDiskMb\":2048,\"database\":{\"type\":\"POSTGRES_LOGICAL\",\"postgresqlLogical\":{\"version\":\"${PG_VERSION}\"}}}"

wait_for_report '"status":"COMPLETED"' 240 '"status":"FAILED"'

assert_report "$VERIFICATION_ID" '.pgRestoreExitCode == 0'
assert_report "$VERIFICATION_ID" '.tableCount == 2'
assert_report "$VERIFICATION_ID" '(.tableStats | map(.name) | sort) == ["t_a","t_b"]'

echo "Include-public-schema report OK: COMPLETED with exit code 0 and t_a+t_b present"

stop_agent

if ! leak_check "$AGENT_ID"; then
  echo "---- agent.out ----"
  cat agent.out
  exit 1
fi

echo "Verification agent restore include-public-schema e2e passed"
