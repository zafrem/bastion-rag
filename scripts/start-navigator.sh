#!/usr/bin/env bash
# Start the Navigator module (Python) in the foreground. REST :8082 · gRPC :9092.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

cd "$ROOT/navigator"
# Navigator refuses to start without a caller auth token. For local runs, generate an ephemeral
# one; set NAVIGATOR_AUTH_TOKEN yourself (and give the same value to callers) to keep it stable.
if [[ -z "${NAVIGATOR_AUTH_TOKEN:-}" ]]; then
  NAVIGATOR_AUTH_TOKEN="$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
  export NAVIGATOR_AUTH_TOKEN
  echo "[navigator] NAVIGATOR_AUTH_TOKEN not set — generated an ephemeral token for this run"
fi
PY="$(python_for "$ROOT/navigator")"
echo "[navigator] using interpreter: $PY"
echo "[navigator] starting server (REST :8082, gRPC :9092) — Ctrl-C to stop"
echo "[navigator] note: first run downloads BAAI/bge-m3 (~1.4 GB) + reranker (~900 MB)"
exec "$PY" -m navigator.main --config config/config.yaml
