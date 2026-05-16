#!/usr/bin/env bash
# Run this ONCE on the Oracle VM after first login.
# ssh ubuntu@<your-vm-ip> < deploy/setup-server.sh

set -euo pipefail

APP_DIR="/opt/chess"
DATA_DIR="/var/lib/chess"
SERVICE_USER="chess"

echo "==> Updating packages..."
sudo apt-get update -q
sudo apt-get install -y nginx certbot python3-certbot-nginx ufw

echo "==> Creating app user and directories..."
sudo useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER" || true
sudo mkdir -p "$APP_DIR" "$DATA_DIR"
sudo chown "$SERVICE_USER:$SERVICE_USER" "$DATA_DIR"
sudo chown root:root "$APP_DIR"
sudo chmod 755 "$APP_DIR"

echo "==> Installing systemd service..."
sudo cp /tmp/chess.service /etc/systemd/system/chess.service
sudo systemctl daemon-reload
sudo systemctl enable chess

echo "==> Installing Nginx config..."
sudo cp /tmp/chess-nginx.conf /etc/nginx/sites-available/chess
sudo ln -sf /etc/nginx/sites-available/chess /etc/nginx/sites-enabled/chess
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx

echo "==> Configuring firewall (ufw + Oracle iptables)..."
# ufw
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'
sudo ufw --force enable

# Oracle Cloud VMs also block traffic via iptables rules added by the OS image.
# These rules allow HTTP and HTTPS through the OS-level firewall.
sudo iptables -I INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 443 -j ACCEPT
sudo netfilter-persistent save 2>/dev/null || sudo apt-get install -y iptables-persistent && sudo netfilter-persistent save

echo ""
echo "==> Setup complete."
echo "    Next steps:"
echo "    1. Run deploy.sh to upload the binary and static files."
echo "    2. Create /opt/chess/.env with your production env vars."
echo "    3. Run: sudo certbot --nginx -d your-domain.com"
echo "    4. sudo systemctl start chess"
