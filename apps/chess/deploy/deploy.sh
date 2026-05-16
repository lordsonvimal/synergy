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

echo "==> Preparing remote directory..."
# Recursive chown so files left over from previous deploys (possibly owned by
# root or a different user) can be overwritten by rsync/scp under our user.
$SSH "$REMOTE" "sudo mkdir -p $REMOTE_DIR && sudo chown -R \$(whoami) $REMOTE_DIR"

echo "==> Uploading binary to staging path..."
# Upload to /tmp first, then sudo-move into place after stopping the service.
# This avoids ETXTBSY on running binaries and keeps the swap atomic.
scp ${SSH_KEY:+-i "$SSH_KEY"} /tmp/chess-server "$REMOTE:/tmp/chess-server.new"

echo "==> Uploading static assets..."
eval "$RSYNC_SSH" dist/  "$REMOTE:$REMOTE_DIR/dist/"
eval "$RSYNC_SSH" assets/ "$REMOTE:$REMOTE_DIR/assets/"

echo "==> Swapping binary and restarting service..."
$SSH "$REMOTE" "sudo systemctl stop chess && \
  sudo mv /tmp/chess-server.new $REMOTE_DIR/chess-server && \
  sudo chmod +x $REMOTE_DIR/chess-server && \
  sudo systemctl start chess"

echo "==> Done. Checking status..."
$SSH "$REMOTE" "sudo systemctl status chess --no-pager"
