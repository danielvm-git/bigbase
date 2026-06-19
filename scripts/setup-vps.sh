#!/usr/bin/env bash
# ============================================================================
# setup-vps.sh — One-time Contabo VPS setup for BigBase
# Run this ONCE on your fresh Contabo VPS before the first GitHub Actions deploy.
#
# Usage:
#   ssh root@<VPS_IP> 'bash -s' < scripts/setup-vps.sh
#
# Or copy it to the VPS and run:
#   ./setup-vps.sh
#
# Idempotent — safe to run multiple times.
# ============================================================================
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()  { printf "${GREEN}%s${NC}\n" "$1"; }
warn()  { printf "${YELLOW}WARNING: %s${NC}\n" "$1"; }
err()   { printf "${RED}ERROR: %s${NC}\n" "$1"; }

if [ "$(id -u)" -ne 0 ]; then
  err "This script must be run as root (use sudo)"
  exit 1
fi

# --- Detect architecture ---
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    err "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac
info "Architecture: $ARCH"

# --- Configuration (edit these if needed) ---
BIGBASE_USER="${BIGBASE_USER:-bigbase}"
BIGBASE_GROUP="${BIGBASE_GROUP:-bigbase}"
BIGBASE_HOME="/opt/bigbase"
BIGBASE_PORT="${BIGBASE_PORT:-8080}"
BIGBASE_DB="${BIGBASE_HOME}/data/bigbase.db"

# ============================================================================
# Step 1: System packages
# ============================================================================
info "[1/7] Installing system packages..."
apt-get update -qq
apt-get install -y -qq \
  curl \
  gnupg \
  ufw \
  caddy \
  rsync \
  ca-certificates

# Node.js 24 LTS for Vite/npm site builds (Vite 8: Node >= 20.19; platform standard is 24 LTS)
NODE_MAJOR=0
if command -v node >/dev/null 2>&1; then
  NODE_MAJOR=$(node -v 2>/dev/null | sed -E 's/^v([0-9]+).*/\1/')
fi
if [ "${NODE_MAJOR:-0}" -lt 24 ]; then
  info "  Installing Node.js 24 LTS (NodeSource)..."
  curl -fsSL https://deb.nodesource.com/setup_24.x | bash -
  apt-get install -y -qq nodejs
fi
info "  Node $(node -v), npm $(npm -v)"

# ============================================================================
# Step 2: Create bigbase user
# ============================================================================
info "[2/7] Creating '${BIGBASE_USER}' system user..."
if ! id -u "${BIGBASE_USER}" &>/dev/null; then
  useradd --system --no-create-home --shell /usr/sbin/nologin "${BIGBASE_USER}"
  info "  Created user: ${BIGBASE_USER}"
else
  info "  User already exists: ${BIGBASE_USER}"
fi

# ============================================================================
# Step 3: Create directory structure
# ============================================================================
info "[3/7] Creating directory structure..."
mkdir -p "${BIGBASE_HOME}/bin"
mkdir -p "${BIGBASE_HOME}/data"
mkdir -p "${BIGBASE_HOME}/backups"
mkdir -p "${BIGBASE_HOME}/releases"
mkdir -p "${BIGBASE_HOME}/.npm"
chown -R "${BIGBASE_USER}:${BIGBASE_GROUP}" "${BIGBASE_HOME}"
chmod 700 "${BIGBASE_HOME}/.npm"

# ============================================================================
# Step 4: Configure firewall (UFW)
# ============================================================================
info "[4/7] Configuring firewall..."
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp    comment 'SSH'
ufw allow 80/tcp    comment 'HTTP'
ufw allow 443/tcp   comment 'HTTPS'
ufw --force enable
info "  Firewall rules applied: SSH(22) HTTP(80) HTTPS(443)"

# ============================================================================
# Step 5: Set up Caddy reverse proxy
# ============================================================================
info "[5/7] Configuring Caddy reverse proxy..."

cat > /etc/caddy/Caddyfile << CADDYEOF
# BigBase — Caddy reverse proxy configuration
# ============================================
#
# Apex: automatic HTTPS (HTTP-01).
# Site subdomains: on-demand per-host certs (HTTP-01), allowed only when BigBase
# registers the host for a running deployment (see /api/internal/caddy-allow).

{
    email admin@bigbase.click
    on_demand_tls {
        ask http://127.0.0.1:${BIGBASE_PORT}/api/internal/caddy-allow
        interval 2m
        burst 5
    }
}

bigbase.click {
    reverse_proxy 127.0.0.1:${BIGBASE_PORT} {
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }

    # Security headers
    header {
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
        -Server
    }

    # Structured JSON access logs
    log {
        output file /var/log/caddy/bigbase-access.log
        format json
    }
}

# Deployed sites: https://<site-slug>.bigbase.click (DNS wildcard *.bigbase.click → VPS)
*.bigbase.click {
    tls {
        on_demand
    }

    reverse_proxy 127.0.0.1:${BIGBASE_PORT} {
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }

    header {
        X-Content-Type-Options "nosniff"
        X-Frame-Options "SAMEORIGIN"
        Referrer-Policy "strict-origin-when-cross-origin"
        -Server
    }

    log {
        output file /var/log/caddy/bigbase-sites-access.log
        format json
    }
}
CADDYEOF

systemctl enable caddy
if caddy validate --config /etc/caddy/Caddyfile; then
  systemctl reload caddy 2>/dev/null || systemctl restart caddy
  info "  Caddy reloaded — on-demand TLS for site subdomains"
else
  warn "  Caddyfile validation failed — fix /etc/caddy/Caddyfile and run: systemctl reload caddy"
fi
info "  Caddy configured — https://bigbase.click and https://<slug>.bigbase.click → localhost:${BIGBASE_PORT}"

# ============================================================================
# Step 6: Create systemd service for BigBase
# ============================================================================
info "[6/7] Creating systemd service..."

cat > /etc/systemd/system/bigbase.service << SYSTEMDEOF
[Unit]
Description=BigBase BaaS Platform
Documentation=https://github.com/danielvm-git/bigbase
After=network.target caddy.service
Wants=caddy.service

[Service]
Type=simple
User=${BIGBASE_USER}
Group=${BIGBASE_GROUP}
WorkingDirectory=${BIGBASE_HOME}

# The binary is deployed by GitHub Actions
ExecStart=${BIGBASE_HOME}/bin/bigbase serve \\
    --port ${BIGBASE_PORT} \\
    --db ${BIGBASE_DB} \\
    --sites-domain bigbase.click

# Environment variables for Google OAuth (set from GitHub Secrets)
EnvironmentFile=-${BIGBASE_HOME}/.env

# npm builds (user has --no-create-home; Vite needs Node 24 LTS on PATH)
Environment=BIGBASE_HOME=${BIGBASE_HOME}
Environment=HOME=${BIGBASE_HOME}
Environment=NPM_CONFIG_CACHE=${BIGBASE_HOME}/.npm
Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Restart behavior
Restart=always
RestartSec=5
StartLimitIntervalSec=60
StartLimitBurst=3

# Graceful shutdown
TimeoutStopSec=30
KillSignal=SIGTERM

# Security hardening
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=full
ProtectHome=yes
ReadWritePaths=${BIGBASE_HOME}/data
ReadWritePaths=${BIGBASE_HOME}/backups
ReadWritePaths=${BIGBASE_HOME}/secrets
ReadWritePaths=${BIGBASE_HOME}/.npm

[Install]
WantedBy=multi-user.target
SYSTEMDEOF

systemctl daemon-reload
systemctl enable bigbase
info "  systemd service created and enabled: bigbase.service"

# ============================================================================
# Step 7: Create helper scripts
# ============================================================================
info "[7/7] Creating helper scripts..."

# Rollback script
cat > "${BIGBASE_HOME}/rollback.sh" << 'ROLLBACK'
#!/usr/bin/env bash
# ============================================================================
# rollback.sh — Roll back to the previous BigBase release
# ============================================================================
set -euo pipefail

BIGBASE_HOME="/opt/bigbase"
RELEASES_DIR="${BIGBASE_HOME}/releases"

if [ ! -f "${RELEASES_DIR}/previous/bigbase" ]; then
  echo "ERROR: No previous release found at ${RELEASES_DIR}/previous/"
  exit 1
fi

echo "Rolling back to previous release..."
cp "${RELEASES_DIR}/previous/bigbase" "${BIGBASE_HOME}/bin/bigbase"
chmod +x "${BIGBASE_HOME}/bin/bigbase"
chown bigbase:bigbase "${BIGBASE_HOME}/bin/bigbase"

systemctl restart bigbase
sleep 3

# Health check
if curl -sf http://localhost:8080/api/monitoring/health > /dev/null 2>&1; then
  echo "Rollback successful — BigBase is running."
else
  echo "ERROR: Rollback failed — health check did not pass."
  exit 1
fi
ROLLBACK
chmod +x "${BIGBASE_HOME}/rollback.sh"
chown "${BIGBASE_USER}:${BIGBASE_GROUP}" "${BIGBASE_HOME}/rollback.sh"

# Status script
cat > "${BIGBASE_HOME}/status.sh" << 'STATUS'
#!/usr/bin/env bash
# ============================================================================
# status.sh — Check BigBase deployment status
# ============================================================================
echo "=== BigBase Service Status ==="
systemctl status bigbase --no-pager 2>&1 | head -20

echo ""
echo "=== Binary ==="
ls -lh /opt/bigbase/bin/bigbase 2>/dev/null || echo "No binary found"

echo ""
echo "=== Database ==="
ls -lh /opt/bigbase/data/bigbase.db 2>/dev/null || echo "No database found"

echo ""
echo "=== Backups ==="
ls -lh /opt/bigbase/backups/ 2>/dev/null | tail -5 || echo "No backups found"

echo ""
echo "=== Listeners ==="
ss -tlnp | grep -E ':(80|443|8080)' || echo "No listeners found"

echo ""
echo "=== Health Check ==="
curl -sf http://localhost:8080/api/monitoring/health && echo "  OK" || echo "  FAILED"
STATUS
chmod +x "${BIGBASE_HOME}/status.sh"
chown "${BIGBASE_USER}:${BIGBASE_GROUP}" "${BIGBASE_HOME}/status.sh"

# ============================================================================
# Step 8: New Relic Infrastructure Agent (optional)
# ============================================================================
if [ -n "${NEW_RELIC_API_KEY:-}" ] && [ -n "${NEW_RELIC_ACCOUNT_ID:-}" ]; then
  info "[8/8] Installing New Relic infrastructure agent..."

  NR_CLI="/usr/local/bin/newrelic"
  if ! command -v "$NR_CLI" &>/dev/null; then
    curl -Ls https://download.newrelic.com/install/newrelic-cli/scripts/install.sh | bash
  fi

  NR_REGION="${NEW_RELIC_REGION:-US}"
  if [ "$NR_REGION" = "EU" ]; then
    sudo env NEW_RELIC_API_KEY="$NEW_RELIC_API_KEY" \
      NEW_RELIC_ACCOUNT_ID="$NEW_RELIC_ACCOUNT_ID" \
      NEW_RELIC_REGION="$NR_REGION" \
      "$NR_CLI" install -n infrastructure-agent-installer -y
  else
    sudo env NEW_RELIC_API_KEY="$NEW_RELIC_API_KEY" \
      NEW_RELIC_ACCOUNT_ID="$NEW_RELIC_ACCOUNT_ID" \
      "$NR_CLI" install -n infrastructure-agent-installer -y
  fi
  info "  New Relic agent installed. View at https://one.newrelic.com > Infrastructure > Hosts"
else
  info "[8/8] Skipping New Relic (set NEW_RELIC_API_KEY + NEW_RELIC_ACCOUNT_ID to enable)"
fi

echo ""
echo "================================================================================"
info "  BigBase VPS setup complete!"
echo ""
echo "  Binary directory:  ${BIGBASE_HOME}/bin/"
echo "  Data directory:    ${BIGBASE_HOME}/data/"
echo "  Backups directory: ${BIGBASE_HOME}/backups/"
echo "  Releases:          ${BIGBASE_HOME}/releases/"
echo ""
echo "  Service:           systemctl [start|stop|restart|status] bigbase"
echo "  Logs:              journalctl -u bigbase -f"
echo "  Status:            sudo ${BIGBASE_HOME}/status.sh"
echo "  Rollback:          sudo ${BIGBASE_HOME}/rollback.sh"
echo ""
echo "  Caddy logs:        journalctl -u caddy -f"
echo ""
echo "  NEXT STEPS:"
echo "  1. Set up GitHub Secrets in your repo:"
echo "       - CONTABO_HOST     → Your VPS IP address"
echo "       - CONTABO_USER     → $(whoami)"
echo "       - CONTABO_SSH_KEY  → Your SSH private key (deploy key)"
echo "       - GOOGLE_CLIENT_ID     → (optional) Google OAuth client ID"
echo "       - GOOGLE_CLIENT_SECRET → (optional) Google OAuth client secret"
echo "       - BIGBASE_GITHUB_APP_ID → GitHub App ID (repo secret; not GITHUB_* prefix)"
echo "       - BIGBASE_GITHUB_APP_SLUG → GitHub App slug (e.g. bigbaseguthubapp)"
echo "       - BIGBASE_GITHUB_APP_PRIVATE_KEY → GitHub App PEM (multiline)"
echo "       - BIGBASE_GITHUB_WEBHOOK_SECRET → GitHub App webhook secret"
echo "  2. Push to main — GitHub Actions will build and deploy"
echo "  3. Domain bigbase.click is pre-configured in the Caddyfile"
echo ""
echo "  Optional — New Relic monitoring:"
echo "    1. Get your API key at https://one.newrelic.com > API Keys"
echo "    2. Copy your Account ID from the URL"
echo "    3. Re-run: ssh root@VPS NEW_RELIC_API_KEY=NRAK-... NEW_RELIC_ACCOUNT_ID=12345 "bash -s" < scripts/setup-vps.sh"
echo "================================================================================"
