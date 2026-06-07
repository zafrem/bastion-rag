#!/usr/bin/env bash
# Start the Tracker module (Go) in the foreground.
# REST/UI :8080 · gRPC :9090 · WebSocket :8081 · dashboard http://localhost:8080.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

cd "$ROOT/tracker"
echo "[tracker] building ./cmd/tracker-cli ..."
go build -o bin/tracker-cli ./cmd/tracker-cli
echo "[tracker] starting server (dashboard http://localhost:8080) — Ctrl-C to stop"
exec ./bin/tracker-cli server
