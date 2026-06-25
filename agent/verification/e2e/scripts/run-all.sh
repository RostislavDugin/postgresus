#!/bin/bash
set -uo pipefail

SCRIPT_DIR="$(dirname "$0")"

TESTS=(
  "test-heartbeat-and-upgrade.sh"
  "test-start-stop-status.sh"
  "test-startup-purge.sh"
  "test-restore-broken.sh"
  "test-restore-disk-budget.sh"
  "test-restore-timescale.sh"
)

PG_VERSIONS=(12 13 14 15 16 17 18)

echo "========================================"
echo "  Verification agent e2e"
echo "========================================"

PASSED=0
FAILED=0

for test in "${TESTS[@]}"; do
  echo ""
  echo "---- $test ----"
  if bash "$SCRIPT_DIR/$test"; then
    PASSED=$((PASSED + 1))
  else
    FAILED=$((FAILED + 1))
  fi
done

for V in "${PG_VERSIONS[@]}"; do
  echo ""
  echo "---- test-restore-success.sh $V ----"
  if bash "$SCRIPT_DIR/test-restore-success.sh" "$V"; then
    PASSED=$((PASSED + 1))
  else
    FAILED=$((FAILED + 1))
  fi
done

echo ""
echo "========================================"
echo "  Results: $PASSED passed, $FAILED failed"
echo "========================================"

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi
