#!/usr/bin/env bash
# Run from apps/chess/ — builds the app and uploads it to the Oracle VM.
# Usage: ./deploy/deploy.sh <user@host> [path-to-ssh-key]
# Example: ./deploy/deploy.sh ubuntu@140.245.207.184 ~/.ssh/id_rsa
# Example: ./deploy/deploy.sh ubuntu@140.245.207.184 ~/Downloads/ssh-key-2026-05-16.key

set -euo pipefail

REMOTE="${1:?Usage: $0 user@host [ssh-key]}"
SSH_KEY="${2:-}"
REMOTE_DIR="/opt/chess"

# Build SSH and rsync flags with the key if provided
if [[ -n "$SSH_KEY" ]]; then
  SSH="ssh -i $SSH_KEY"
  RSYNC_SSH="rsync -avz --delete -e 'ssh -i $SSH_KEY'"
else
  SSH="ssh"
  RSYNC_SSH="rsync -avz --delete"
fi

echo "==> Generating templ files..."
templ generate

echo "==> Building Tailwind CSS..."
npx @tailwindcss/cli -i ./ui/styles/style.css -o ./dist/style.css --minify

echo "==> Cross-compiling for linux/amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /tmp/chess-server .

echo "==> Uploading binary and static assets..."
$SSH "$REMOTE" "sudo mkdir -p $REMOTE_DIR && sudo chown \$(whoami) $REMOTE_DIR"
scp ${SSH_KEY:+-i "$SSH_KEY"} /tmp/chess-server "$REMOTE:$REMOTE_DIR/chess-server"
eval "$RSYNC_SSH" dist/  "$REMOTE:$REMOTE_DIR/dist/"
eval "$RSYNC_SSH" assets/ "$REMOTE:$REMOTE_DIR/assets/"

echo "==> Restarting service..."
$SSH "$REMOTE" "sudo systemctl restart chess"

echo "==> Done. Checking status..."
$SSH "$REMOTE" "sudo systemctl status chess --no-pager"
