# Chess App — Oracle Cloud Deployment

**VM IP:** `140.245.207.184`  
**SSH key:** `~/Downloads/ssh-key-2026-05-16.key`  
**Username:** `ubuntu`

The app runs on an Oracle Cloud Always Free VM (VM.Standard.E2.1.Micro — 1 OCPU, 1 GB RAM, Ubuntu 22.04).
Nginx sits in front of the Go server as a reverse proxy, handling HTTP and SSE buffering.
SQLite is stored at `/var/lib/chess/chess.db` on the VM's persistent disk.

---

## Step 1 — Fix SSH key permissions (run once)

SSH refuses to use a private key that is readable by others. This sets it to owner-read-only.

```bash
chmod 400 ~/Downloads/ssh-key-2026-05-16.key
```

---

## Step 2 — Open ports in Oracle Console (run once)

Oracle's network-level firewall blocks all ports except 22 (SSH) by default.
These rules allow HTTP and HTTPS traffic to reach the VM.

1. Oracle Console → Instances → instance-chessleap → **Networking** tab
2. Click **subnet-20260516-1501**
3. Click **Security Lists** → default security list → **Add Ingress Rules**
4. Add rule 1 — allows HTTP traffic:
   - Source CIDR: `0.0.0.0/0`, Protocol: TCP, Destination Port: `80`
5. Add rule 2 — allows HTTPS traffic:
   - Source CIDR: `0.0.0.0/0`, Protocol: TCP, Destination Port: `443`

---

## Step 3 — Set up the server (run once)

Copies the systemd service definition and Nginx config to the VM, then runs `setup-server.sh`
which installs Nginx and Stockfish (used by the game-analysis worker), creates the `chess`
system user, sets up directories, enables the service, and configures the OS-level firewall
(ufw + iptables) to allow ports 80 and 443.

Run from `apps/chess/`:

```bash
# Copy the systemd service file — defines how the OS starts/restarts the chess process
scp -i ~/Downloads/ssh-key-2026-05-16.key deploy/chess.service ubuntu@140.245.207.184:/tmp/chess.service

# Copy the Nginx config — sets up the reverse proxy and SSE streaming settings
scp -i ~/Downloads/ssh-key-2026-05-16.key deploy/nginx.conf ubuntu@140.245.207.184:/tmp/chess-nginx.conf

# Run the setup script on the VM over SSH
ssh -i ~/Downloads/ssh-key-2026-05-16.key ubuntu@140.245.207.184 < deploy/setup-server.sh
```

---

## Step 4 — Create .env on the server (run once)

The app reads configuration from `/opt/chess/.env` at startup.
`SESSION_SECRET` is the HMAC key used to sign play session JWTs — must be a strong random value.

Generate a secret on your Mac:

```bash
# Generates a cryptographically random 32-byte string encoded as base64
openssl rand -base64 32
```

SSH into the VM and create the file:

```bash
# Open an interactive SSH session on the VM
ssh -i ~/Downloads/ssh-key-2026-05-16.key ubuntu@140.245.207.184

# Create the app directory (if not already created by setup-server.sh)
sudo mkdir -p /opt/chess

# Open the .env file in the nano text editor
sudo nano /opt/chess/.env
```

Paste the following (replace `<secret>` with the openssl output):

```
GIN_MODE=release        # Disables Gin's debug logging and enables production optimisations
DATA_DIR=/var/lib/chess # Directory where chess.db is stored — persists across restarts
SESSION_SECRET=<secret> # HMAC signing key for JWTs — keep this private
```

Save: `Ctrl+O` → `Enter` → `Ctrl+X`. Then type `exit` to leave the SSH session.

---

## Step 5 — Deploy the app (run on every update)

`deploy.sh` does the full build and upload in one command:
1. Runs `templ generate` to compile Go HTML templates
2. Builds Tailwind CSS and minifies it into `dist/style.css`
3. Cross-compiles the Go binary for `linux/amd64`
4. Uploads the binary, `dist/`, and `assets/` to the VM via scp/rsync
5. Restarts the systemd service

Run from `apps/chess/`:

```bash
# Pass the SSH key as the second argument so scp and rsync can authenticate
./deploy/deploy.sh ubuntu@140.245.207.184 ~/Downloads/ssh-key-2026-05-16.key
```

First deploy only — the service is installed but not yet started by setup-server.sh, so start it manually:

```bash
ssh -i ~/Downloads/ssh-key-2026-05-16.key ubuntu@140.245.207.184 "sudo systemctl start chess"
```

---

## Step 6 — Verify

Check that the systemd service is running and there are no startup errors:

```bash
ssh -i ~/Downloads/ssh-key-2026-05-16.key ubuntu@140.245.207.184 "sudo systemctl status chess --no-pager"
```

App is live at: http://140.245.207.184

---

## Useful commands

Run these after SSH-ing into the VM (`ssh -i ~/Downloads/ssh-key-2026-05-16.key ubuntu@140.245.207.184`):

```bash
# Stream live app logs (Ctrl+C to stop)
sudo journalctl -u chess -f

# Restart the app — needed after changing .env
sudo systemctl restart chess

# Stop the app
sudo systemctl stop chess

# Stream Nginx error logs — useful if the site is unreachable
sudo tail -f /var/log/nginx/error.log

# Stream Nginx access logs — shows all incoming HTTP requests
sudo tail -f /var/log/nginx/access.log

# Check how much disk the SQLite database is using
du -sh /var/lib/chess/

# Check available RAM
free -h
```
