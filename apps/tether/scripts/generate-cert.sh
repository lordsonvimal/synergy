#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CERT_DIR="$SCRIPT_DIR/../certs"

mkdir -p "$CERT_DIR"

LOCAL_IP=$(ipconfig getifaddr en0 2>/dev/null || echo "192.168.1.1")

# Allow extra SANs via arguments (e.g. Tailscale IP, custom hostname)
SANS=("localhost" "127.0.0.1" "$LOCAL_IP")
for arg in "$@"; do
  SANS+=("$arg")
done

echo "Generating certs for: ${SANS[*]}"

mkcert -cert-file "$CERT_DIR/cert.pem" -key-file "$CERT_DIR/key.pem" \
  "${SANS[@]}"

echo ""
echo "Certificates generated in $CERT_DIR"
echo "Local IP: $LOCAL_IP"
echo ""
echo "Connect from your phone at https://$LOCAL_IP:5100"
