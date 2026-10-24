#!/bin/bash
# Shared helpers for verification-agent e2e scripts (sourced by each
# scripts/test-*.sh). The runner uses network_mode: host so 127.0.0.1:4050
# reaches the published mock-server port AND the random host ports the
# spawned Postgres containers bind 5432 to — same setup as production
# (agent on the VM, sibling containers).

MOCK="http://127.0.0.1:${E2E_MOCK_SERVER_PORT:-4050}"
# shellcheck disable=SC2034 # read by the test-*.sh scripts that source this file
ARTIFACTS="/opt/agent/artifacts"

hb_count() {
  curl -sf "$MOCK/mock/heartbeats" | sed -n 's/.*"count":\([0-9]*\).*/\1/p'
}

reports_json() {
  curl -sf "$MOCK/mock/reports"
}

# reset_mock_version pins the mock to v1.0.0 / agent-v1 so the agent's
# background upgrader finds no newer version and the process stays up for the
# whole test.
reset_mock_version() {
  curl -sf -X POST "$MOCK/mock/set-version" \
    -H 'Content-Type: application/json' -d '{"version":"v1.0.0"}'
  curl -sf -X POST "$MOCK/mock/set-binary-path" \
    -H 'Content-Type: application/json' -d '{"binaryPath":"/artifacts/agent-v1"}'
}

# reset_mock_state wipes per-test state (reports, claim, fixture, abort/stream
# flags, heartbeat counter). The mock is shared across all e2e scripts in one
# run; without this, a later script sees an earlier script's report and bails.
reset_mock_state() {
  curl -sf -X POST "$MOCK/mock/reset" >/dev/null
}

# leak_check is non-zero if any container/network labeled with the given agent
# ID survives. The anonymous PGDATA volume carries no label and is removed
# alongside the container by RemoveOptions{RemoveVolumes:true}, so a leaked
# container is the canonical signal of a leaked PGDATA.
leak_check() {
  local agent_id="$1"
  local label="databasus.verification.agent_id=${agent_id}"
  local rc=0

  local containers
  containers="$(docker ps -a --filter "label=${label}" --format '{{.ID}} {{.Names}}')"
  if [ -n "$containers" ]; then
    echo "FAIL: leaked containers labeled ${label}:"
    echo "$containers"
    rc=1
  fi

  local networks
  networks="$(docker network ls --filter "label=${label}" --format '{{.ID}} {{.Name}}')"
  if [ -n "$networks" ]; then
    echo "FAIL: leaked networks labeled ${label}:"
    echo "$networks"
    rc=1
  fi

  return $rc
}

# start_agent <agent_id> [extra_flag...] — runs `databasus-verification-agent run`
# with the standard launch block in the current dir, redirects stdout+stderr to
# ./agent.out, and sets AGENT_PID. Extra flags forwarded to the binary.
start_agent() {
  local agent_id="$1"
  shift
  ./databasus-verification-agent run \
    --databasus-host "$MOCK" \
    --agent-id "$agent_id" \
    --token test-token \
    --max-cpu 4 \
    --max-ram-mb 2048 \
    --max-disk-gb 20 \
    --max-concurrent-jobs 2 \
    --allow-insecure-http \
    --skip-update \
    "$@" > agent.out 2>&1 &
  AGENT_PID=$!
}

stop_agent() {
  kill "$AGENT_PID" 2>/dev/null || true
  wait "$AGENT_PID" 2>/dev/null || true
}

dump_diagnostics() {
  echo "---- agent.out (last 30) ----"
  tail -30 agent.out 2>/dev/null || echo "(no agent.out)"
  echo "---- databasus-verification.log (last 60) ----"
  tail -60 databasus-verification.log 2>/dev/null || echo "(no log file)"
}

# wait_for_report <pattern> <deadline_seconds> [fatal_pattern] — polls
# /mock/reports every 2s until grep -q "$pattern" matches. If fatal_pattern is
# set and matches first, exits 1 with diagnostics. On timeout, exits 1 with
# diagnostics. On success, REPORTS holds the body for follow-up asserts.
wait_for_report() {
  local want="$1"
  local seconds="$2"
  local fatal="${3:-}"
  local deadline=$((SECONDS + seconds))
  REPORTS=""
  while [ $SECONDS -lt $deadline ]; do
    REPORTS="$(reports_json)"
    if echo "$REPORTS" | grep -q "$want"; then
      return 0
    fi
    if [ -n "$fatal" ] && echo "$REPORTS" | grep -q "$fatal"; then
      echo "FAIL: saw fatal pattern $fatal before expected $want"
      echo "reports: $REPORTS"
      dump_diagnostics
      exit 1
    fi
    sleep 2
  done
  echo "FAIL: no report matching $want within ${seconds}s"
  echo "reports: $REPORTS"
  dump_diagnostics
  exit 1
}

# assert_report <verification_id> <jq_expr> — selects the report whose
# verificationId matches and asserts jq_expr is truthy. On failure prints the
# matching report pretty-printed.
assert_report() {
  local vid="$1"
  local expr="$2"
  if ! reports_json | jq -e --arg vid "$vid" \
       "(.reports[] | select(.verificationId == \$vid)) | $expr" >/dev/null; then
    echo "FAIL: report assertion failed for $vid: $expr"
    reports_json | jq --arg vid "$vid" '.reports[] | select(.verificationId == $vid)'
    exit 1
  fi
}
