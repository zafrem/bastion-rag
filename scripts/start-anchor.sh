#!/usr/bin/env bash
# Start the Anchor module (Python) in the foreground. REST :8083 · gRPC :9093.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

cd "$ROOT/anchor"
PY="$(python_for "$ROOT/anchor")"
echo "[anchor] using interpreter: $PY"
echo "[anchor] starting server (REST :8083, gRPC :9093) — Ctrl-C to stop"
exec "$PY" -m anchor.main --config config/config.yaml
