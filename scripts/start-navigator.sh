#!/usr/bin/env bash
# Start the Navigator module (Python) in the foreground. REST :8082 · gRPC :9092.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

cd "$ROOT/navigator"
PY="$(python_for "$ROOT/navigator")"
echo "[navigator] using interpreter: $PY"
echo "[navigator] starting server (REST :8082, gRPC :9092) — Ctrl-C to stop"
echo "[navigator] note: first run downloads BAAI/bge-m3 (~1.4 GB) + reranker (~900 MB)"
exec "$PY" -m navigator.main --config config/config.yaml
