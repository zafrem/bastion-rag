#!/usr/bin/env bash
# Start the Anchor module (Python) in the foreground. REST :8083 · gRPC :9093.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

cd "$ROOT/anchor"
# Anchor refuses to start without a caller auth token. For local runs, generate an ephemeral
# one; set ANCHOR_AUTH_TOKEN yourself (and give the same value to callers) to keep it stable.
if [[ -z "${ANCHOR_AUTH_TOKEN:-}" ]]; then
  ANCHOR_AUTH_TOKEN="$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
  export ANCHOR_AUTH_TOKEN
  echo "[anchor] ANCHOR_AUTH_TOKEN not set — generated an ephemeral token for this run"
fi
PY="$(python_for "$ROOT/anchor")"
echo "[anchor] using interpreter: $PY"
echo "[anchor] starting server (REST :8083, gRPC :9093) — Ctrl-C to stop"
exec "$PY" -m anchor.main --config config/config.yaml
