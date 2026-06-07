#!/usr/bin/env bash
# Start the Sentinel module (Go) in the foreground. REST :8080 · gRPC :9090.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

cd "$ROOT/sentinel"
echo "[sentinel] building ./cmd/sentinel ..."
go build -o bin/sentinel ./cmd/sentinel
echo "[sentinel] starting server (REST :8080, gRPC :9090) — Ctrl-C to stop"
exec ./bin/sentinel server
