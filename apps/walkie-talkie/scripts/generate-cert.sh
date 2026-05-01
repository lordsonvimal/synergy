#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CERT_DIR="$SCRIPT_DIR/../certs"

mkdir -p "$CERT_DIR"

LOCAL_IP=$(ipconfig getifaddr en0 2>/dev/null || echo "192.168.1.1")

echo "Generating certs for localhost and $LOCAL_IP"

mkcert -cert-file "$CERT_DIR/cert.pem" -key-file "$CERT_DIR/key.pem" \
  localhost 127.0.0.1 "$LOCAL_IP"

echo "Certificates generated in $CERT_DIR"
echo "Local IP: $LOCAL_IP"
