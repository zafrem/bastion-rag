#!/usr/bin/env bash
# Start the Tracker module (Go) in the foreground.
# REST/UI :8080 · gRPC :9090 · WebSocket :8081 · dashboard http://localhost:8080.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

cd "$ROOT/tracker"
# Tracker refuses to start without an audit-signing key (the built-in demo key is
# public, so signatures made with it are forgeable). For local runs, generate an
# ephemeral one; set TRACKER_AUDIT_SECRET yourself to keep signatures verifiable.
if [[ -z "${TRACKER_AUDIT_SECRET:-}" ]]; then
  TRACKER_AUDIT_SECRET="$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
  export TRACKER_AUDIT_SECRET
  echo "[tracker] TRACKER_AUDIT_SECRET not set — generated an ephemeral key for this run"
fi
echo "[tracker] building ./cmd/tracker-cli ..."
go build -o bin/tracker-cli ./cmd/tracker-cli
echo "[tracker] starting server (dashboard http://localhost:8080) — Ctrl-C to stop"
exec ./bin/tracker-cli server
