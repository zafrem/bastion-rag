#!/usr/bin/env bash
# Start the Vault module (Go) in the foreground. REST :8081 · gRPC :9091.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

cd "$ROOT/vault"
# Vault refuses to start without a real JWT secret. For local runs, generate an
# ephemeral one; set VAULT_AUTH_JWT_SECRET yourself to keep tokens valid across restarts.
if [[ -z "${VAULT_AUTH_JWT_SECRET:-}" ]]; then
  VAULT_AUTH_JWT_SECRET="$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
  export VAULT_AUTH_JWT_SECRET
  echo "[vault] VAULT_AUTH_JWT_SECRET not set — generated an ephemeral secret for this run"
fi
echo "[vault] building ./cmd/vault ..."
go build -o bin/vault ./cmd/vault
echo "[vault] starting server (REST :8081, gRPC :9091) — Ctrl-C to stop"
exec ./bin/vault server -c configs/vault.yaml
