#!/bin/bash

# Configuration
MOCK_PORT=11435
LOG_FILE="test_run.log"

echo "================================================================================"
echo "BASTION RAG: INTEGRATION TEST RUNNER"
echo "================================================================================"

# 1. Initialize environment
echo "[1/4] Initializing submodules and workspace..."
git submodule update --init --recursive > /dev/null 2>&1
go mod tidy > /dev/null 2>&1

# 2. Start Mock LLM in background
echo "[2/4] Starting Mock LLM on port $MOCK_PORT..."
go run tests/mock-llm/main.go > "$LOG_FILE" 2>&1 &
MOCK_PID=$!

# Trap to ensure background process is killed on exit
trap "kill $MOCK_PID 2>/dev/null" EXIT

# Wait for server to be ready
MAX_RETRIES=5
COUNT=0
while ! curl -s "http://localhost:$MOCK_PORT/health" > /dev/null; do
    sleep 1
    COUNT=$((COUNT + 1))
    if [ $COUNT -ge $MAX_RETRIES ]; then
        echo "Error: Mock LLM failed to start within $MAX_RETRIES seconds."
        echo "Check $LOG_FILE for details."
        exit 1
    fi
done
echo "      Mock LLM is ready."

# 3. Run Integration Tests
echo "[3/4] Running automated integration tests..."
go test -v tests/integration_test.go
INTEG_STATUS=$?

# 4. Run Bidirectional Flow Demonstration
echo "[4/4] Running bidirectional flow demonstration..."
go test -v tests/flow_test.go
FLOW_STATUS=$?

echo "================================================================================"
if [ $INTEG_STATUS -eq 0 ] && [ $FLOW_STATUS -eq 0 ]; then
    echo "RESULT: SUCCESS - All integration tests passed."
    echo "Logs saved to $LOG_FILE"
    exit 0
else
    echo "RESULT: FAILURE - One or more tests failed."
    echo "Logs saved to $LOG_FILE"
    exit 1
fi
