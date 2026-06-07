#!/usr/bin/env bash
# Start the Vault module (Go) in the foreground. REST :8081 · gRPC :9091.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

cd "$ROOT/vault"
echo "[vault] building ./cmd/vault ..."
go build -o bin/vault ./cmd/vault
echo "[vault] starting server (REST :8081, gRPC :9091) — Ctrl-C to stop"
exec ./bin/vault server -c configs/vault.yaml
