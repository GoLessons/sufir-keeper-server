#!/usr/bin/env bash
set -euo pipefail

out_dir="$(dirname "$0")/../../.docker/nginx/certs"
mkdir -p "$out_dir"

ca_key="$out_dir/dev-ca.key"
ca_crt="$out_dir/dev-ca.crt"
srv_key="$out_dir/server.key"
srv_csr="$out_dir/server.csr"
srv_crt="$out_dir/server.crt"
san_ext="$out_dir/san.ext"

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl not found" >&2
  exit 1
fi

openssl genrsa -out "$ca_key" 2048
openssl req -x509 -new -key "$ca_key" -days 3650 -out "$ca_crt" -subj "/CN=dev-local-ca"

openssl req -new -newkey rsa:2048 -nodes -keyout "$srv_key" -out "$srv_csr" -subj "/CN=localhost"
cat > "$san_ext" <<EOF
subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1
EOF
openssl x509 -req -in "$srv_csr" -CA "$ca_crt" -CAkey "$ca_key" -CAcreateserial -out "$srv_crt" -days 3650 -extfile "$san_ext"

rm -f "$srv_csr" "$san_ext"
echo "Dev CA: $ca_crt"
echo "Server cert: $srv_crt"
