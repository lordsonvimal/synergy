#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_DIR="$SCRIPT_DIR/.."

echo "=== Tether Setup ==="

if ! command -v node &>/dev/null; then
  echo "Error: Node.js is required. Install via: brew install node"
  exit 1
fi

if ! command -v mkcert &>/dev/null; then
  echo "Installing mkcert..."
  brew install mkcert
  mkcert -install
fi

echo "Installing dependencies..."
cd "$APP_DIR"
yarn install

echo "Generating certificates..."
bash "$SCRIPT_DIR/generate-cert.sh"

echo ""
echo "=== Setup complete ==="
echo "Run 'nx dev tether' to start"
echo ""
echo "=== Connecting from your phone ==="
echo "  1. Both devices must be on the same WiFi network"
echo "  2. Set a static IP for this Mac in your router's DHCP settings"
echo "  3. Open https://<mac-ip>:5100 on your phone"
echo "  4. Accept the self-signed certificate warning on first visit"
echo ""
echo "=== Remote Access (no WARP/corporate VPN) ==="
echo "If this machine does NOT have a corporate VPN enforced:"
echo ""
echo "  Option A: Cloudflare Tunnel"
echo "    1. brew install cloudflared"
echo "    2. cloudflared tunnel --url https://localhost:5100 --no-tls-verify"
echo "    3. Use the printed URL on your phone from any network"
echo ""
echo "  Option B: Tailscale"
echo "    1. Install Tailscale on Mac and phone, log in with same account"
echo "    2. bash scripts/generate-cert.sh \$(tailscale ip -4)"
echo "    3. Restart server, connect from phone using Tailscale IP"
