#!/usr/bin/env bash
# Generate a throwaway CA, a server certificate and a client certificate for trying gRPC TLS locally.
#
#   scripts/gen-dev-certs.sh [output-dir] [extra-dns-names]
#
#   output-dir       default: certs   (git-ignored; never commit keys)
#   extra-dns-names  comma-separated names added to the server certificate, e.g. "sentinel,vault.internal"
#                    (localhost and 127.0.0.1 are always included)
#
# FOR LOCAL DEVELOPMENT ONLY. In production use certificates from your own CA or a service mesh.
set -euo pipefail

OUT="${1:-certs}"
EXTRA="${2:-}"
DAYS="${CERT_DAYS:-30}"

command -v openssl >/dev/null || { echo "openssl is required" >&2; exit 1; }
mkdir -p "$OUT"
cd "$OUT"

san="DNS:localhost,IP:127.0.0.1"
if [[ -n "$EXTRA" ]]; then
  IFS=',' read -ra names <<<"$EXTRA"
  for n in "${names[@]}"; do san+=",DNS:${n// /}"; done
fi

ec=(-newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -nodes)

# CA
openssl req -x509 "${ec[@]}" -keyout ca.key -out ca.pem -subj "/CN=bastion-dev-ca" -days "$DAYS" \
  -addext "basicConstraints=critical,CA:TRUE" -addext "keyUsage=critical,keyCertSign,cRLSign" 2>/dev/null

issue() { # name cn extension-line serial
  local name="$1" cn="$2" ext="$3" serial="$4"
  openssl req -new "${ec[@]}" -keyout "$name.key" -out "$name.csr" -subj "/CN=$cn" 2>/dev/null
  printf '%s\n' "$ext" >"$name.ext"
  openssl x509 -req -in "$name.csr" -CA ca.pem -CAkey ca.key -set_serial "$serial" \
    -out "$name.pem" -days "$DAYS" -extfile "$name.ext" 2>/dev/null
  rm -f "$name.csr" "$name.ext"
}

issue server bastion-dev-server $'subjectAltName='"$san"$'\nextendedKeyUsage=serverAuth' 101
issue client bastion-dev-client 'extendedKeyUsage=clientAuth' 102
chmod 600 ./*.key

abs="$(pwd)"
cat <<EOF
Wrote to $abs: ca.pem  server.pem/server.key  client.pem/client.key   (valid $DAYS days)

Server settings (each module's config; the Go modules use server.grpc_tls_*, Navigator/Anchor the same keys):
  grpc_tls_cert: $abs/server.pem
  grpc_tls_key:  $abs/server.key
  grpc_tls_client_ca: $abs/ca.pem        # optional: require client certificates (mutual TLS)

Check a running server:
  openssl s_client -connect localhost:9090 -alpn h2 -CAfile $abs/ca.pem </dev/null 2>&1 | grep 'Verify return code'

Navigator federation peers (navigator config.yaml, per peer):
  tls: true
  tls_ca_file: $abs/ca.pem
  tls_client_cert: $abs/client.pem       # only if the peer requires mutual TLS
  tls_client_key:  $abs/client.key
EOF
