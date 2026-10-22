#!/bin/bash
# Startup purge: at boot the agent removes every container and network labeled
# with its agent_id — the single-agent invariant means no other process owns
# resources with the same agent_id, so blanket removal is safe. Pre-seed a
# stray container, an orphan network with no container attached and a network
# of another agent, start the agent, assert the first two are gone and the
# foreign network remains.
set -euo pipefail

source "$(dirname "$0")/lib.sh"

WORK="/tmp/agent-work-purge"
AGENT_ID="66666666-6666-6666-6666-666666666666"
STRAY_NAME="databasus-verif-purge-stray"
STRAY_LABEL="databasus.verification.agent_id=${AGENT_ID}"
ORPHAN_NETWORK="databasus-verif-purge-orphan"
FOREIGN_NETWORK="databasus-verif-purge-foreign"
FOREIGN_LABEL="databasus.verification.agent_id=67676767-6767-6767-6767-676767676767"

cleanup() {
  docker network rm "$FOREIGN_NETWORK" "$ORPHAN_NETWORK" >/dev/null 2>&1 || true
}
trap cleanup EXIT

rm -rf "$WORK"
mkdir -p "$WORK"
cd "$WORK"

reset_mock_state
reset_mock_version

docker rm -f "$STRAY_NAME" >/dev/null 2>&1 || true
docker run -d --name "$STRAY_NAME" --label "$STRAY_LABEL" alpine sleep 600 >/dev/null

if ! docker ps -a --filter "label=$STRAY_LABEL" --format '{{.Names}}' | grep -q "$STRAY_NAME"; then
  echo "FAIL: pre-seed stray container not present"
  exit 1
fi
echo "Pre-seeded stray container $STRAY_NAME with label $STRAY_LABEL"

cleanup
docker network create --label "$STRAY_LABEL" "$ORPHAN_NETWORK" >/dev/null
docker network create --label "$FOREIGN_LABEL" "$FOREIGN_NETWORK" >/dev/null
echo "Pre-seeded orphan network $ORPHAN_NETWORK and foreign network $FOREIGN_NETWORK"

cp "$ARTIFACTS/agent-v1" ./databasus-verification-agent
chmod +x ./databasus-verification-agent

start_agent "$AGENT_ID"

GONE=0
for _ in $(seq 1 30); do
  # The purge removes containers before networks, so wait for both.
  if [ -z "$(docker ps -a --filter "label=$STRAY_LABEL" --format '{{.Names}}')" ] &&
     [ -z "$(docker network ls --filter "name=${ORPHAN_NETWORK}" --format '{{.Name}}')" ]; then
    GONE=1
    break
  fi
  sleep 1
done

if [ "$GONE" -ne 1 ]; then
  echo "FAIL: stray container or orphan network still present after agent start"
  docker ps -a --filter "label=$STRAY_LABEL"
  docker network ls --filter "label=$STRAY_LABEL"
  echo "---- agent.out (last 30) ----"
  tail -30 agent.out
  stop_agent
  exit 1
fi
echo "Stray container and orphan network purged by agent startup"

if ! docker network ls --filter "name=${FOREIGN_NETWORK}" --format '{{.Name}}' | grep -qx "$FOREIGN_NETWORK"; then
  echo "FAIL: foreign network $FOREIGN_NETWORK was removed by the purge"
  stop_agent
  exit 1
fi
echo "Foreign network left in place"

stop_agent

if ! leak_check "$AGENT_ID"; then
  exit 1
fi

echo "Verification agent startup-purge e2e passed"
