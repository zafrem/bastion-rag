#!/usr/bin/env bash
# End-to-end integration runner for Bastion-RAG.
#
# Brings up the deterministic Mock LLM, then runs the full tests/ package, which
# exercises the cross-module pipeline (Sentinel-IN → Vault → Navigator → Vault →
# Sentinel-OUT). Tests that spawn the Vault/Navigator subprocesses run when the
# tooling is available and skip themselves otherwise; set BASTION_SKIP_SUBPROCESS=1
# to force the in-process-only subset.
#
# Usage:
#   scripts/run-integration.sh                 # full integration package
#   scripts/run-integration.sh -run TestFlow   # extra args pass through to go test
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

MOCK_PORT=11435
ensure_dirs
MOCK_LOG="$LOG_DIR/mock-llm.log"

echo "================================================================================"
echo "BASTION RAG: INTEGRATION TEST RUNNER"
echo "================================================================================"

# 1. Workspace
echo "[1/4] Syncing submodules and workspace..."
( cd "$ROOT" && git submodule update --init --recursive >/dev/null 2>&1 || true )

# 2. Mock LLM
echo "[2/4] Starting Mock LLM on port $MOCK_PORT..."
( cd "$ROOT" && exec go run tests/mock-llm/main.go ) >"$MOCK_LOG" 2>&1 &
MOCK_PID=$!
trap 'kill "$MOCK_PID" 2>/dev/null || true' EXIT

for _ in $(seq 1 30); do
  if curl -sf "http://localhost:$MOCK_PORT/health" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
if ! curl -sf "http://localhost:$MOCK_PORT/health" >/dev/null 2>&1; then
  echo "Error: Mock LLM failed to start. See $MOCK_LOG"
  exit 1
fi
echo "      Mock LLM is ready."

# 3. Integration package
echo "[3/4] Running integration test package (tests/)..."
set +e
( cd "$ROOT" && go test ./tests/ -v -timeout 300s "$@" )
STATUS=$?
set -e

# 4. Result
echo "================================================================================"
if [[ $STATUS -eq 0 ]]; then
  echo "[4/4] RESULT: SUCCESS — integration tests passed."
else
  echo "[4/4] RESULT: FAILURE — see output above (Mock LLM log: $MOCK_LOG)"
fi
echo "================================================================================"
exit $STATUS
