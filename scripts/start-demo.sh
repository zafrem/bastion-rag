#!/bin/bash

# Port Configuration
API_PORT=8090
UI_PORT=3000

echo "================================================================================"
echo "BASTION RAG: INTERACTIVE DEMO LAUNCHER"
echo "================================================================================"

# 1. Start Demo API
echo "[1/2] Starting Demo API on port $API_PORT..."
go run cmd/demo-api/main.go > demo_api.log 2>&1 &
API_PID=$!

# 2. Start Frontend
echo "[2/2] Starting UI Development Server on port $UI_PORT..."
cd ui && npm run dev -- -p $UI_PORT > ../demo_ui.log 2>&1 &
UI_PID=$!

# Trap to ensure processes are killed on exit
trap "kill $API_PID $UI_PID 2>/dev/null" EXIT

echo "--------------------------------------------------------------------------------"
echo "Demo is running!"
echo "API: http://localhost:$API_PORT"
echo "UI:  http://localhost:$UI_PORT"
echo "--------------------------------------------------------------------------------"
echo "Press Ctrl+C to stop the demo."

# Wait for processes
wait
