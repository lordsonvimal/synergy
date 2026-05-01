#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_DIR="$SCRIPT_DIR/.."

echo "=== Walkie Talkie Setup ==="

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
echo "Run 'nx dev walkie-talkie' to start"
