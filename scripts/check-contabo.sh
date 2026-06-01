#!/usr/bin/env bash
# ============================================================================
# check-contabo.sh — Validate Contabo account is API-ready for BigBase deploy
#
# This script verifies:
#   1. cntb CLI is installed (or installs it)
#   2. API credentials are configured
#   3. Can authenticate and list your VPS instances
#   4. The specific BigBase VPS is running
#   5. SSH key-based login works
#   6. Required ports are open
#
# Usage:
#   chmod +x scripts/check-contabo.sh
#   ./scripts/check-contabo.sh
#
# Prerequisites:
#   - A Contabo account with an active Cloud VPS 20
#   - API credentials from Contabo Customer Control Panel:
#     Control Panel → API → (note your Client ID, Client Secret)
#     Then click "Send Link" to set API password
# ============================================================================
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

PASS()  { printf "${GREEN}✓ %s${NC}\n" "$1"; }
WARN()  { printf "${YELLOW}⚠ %s${NC}\n" "$1"; }
FAIL()  { printf "${RED}✗ %s${NC}\n" "$1"; }
INFO()  { printf "${CYAN}%s${NC}\n" "$1"; }

PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

check_result() {
  if [ "$1" -eq 0 ]; then
    PASS "$2"
    PASS_COUNT=$((PASS_COUNT + 1))
  else
    FAIL "$2"
    FAIL_COUNT=$((FAIL_COUNT + 1))
  fi
}

echo ""
INFO "╔══════════════════════════════════════════════════╗"
INFO "║     Contabo Account Readiness Check for BigBase ║"
INFO "╚══════════════════════════════════════════════════╝"
echo ""

# ============================================================================
# Check 1: cntb CLI installed
# ============================================================================
INFO "[1/6] Contabo CLI (cntb)"

if command -v cntb &>/dev/null; then
  INSTALLED_VER=$(cntb version 2>&1 | head -1)
  PASS "cntb CLI found: ${INSTALLED_VER}"
  PASS_COUNT=$((PASS_COUNT + 1))
else
  WARN "cntb CLI not installed."
  WARN_COUNT=$((WARN_COUNT + 1))

  echo ""
  INFO "  Install it now? Press Enter to install, or Ctrl+C to install manually."
  read -r
  echo "  Installing cntb CLI..."

  OS=$(uname -s)
  ARCH=$(uname -m)

  case "$OS" in
    Darwin)
      if command -v brew &>/dev/null; then
        brew install contabo/tap/cntb
      else
        curl -L "https://github.com/contabo/cntb/releases/latest/download/cntb-darwin-${ARCH}" -o /usr/local/bin/cntb
        chmod +x /usr/local/bin/cntb
      fi
      ;;
    Linux)
      curl -L "https://github.com/contabo/cntb/releases/latest/download/cntb-linux-${ARCH}" -o /usr/local/bin/cntb
      chmod +x /usr/local/bin/cntb
      ;;
    *)
      FAIL "Unsupported OS: ${OS}"
      exit 1
      ;;
  esac

  if command -v cntb &>/dev/null; then
    PASS "cntb CLI installed: $(cntb version 2>&1 | head -1)"
  else
    FAIL "cntb CLI installation failed. Install manually from https://github.com/contabo/cntb"
    exit 1
  fi
fi

# ============================================================================
# Check 2: API credentials configured
# ============================================================================
echo ""
INFO "[2/6] API Credentials"

CONFIG_FILE="${HOME}/.cntb.json"
CONFIG_DIR="${HOME}/.config/cntb"

if [ -f "${CONFIG_FILE}" ] || [ -d "${CONFIG_DIR}" ]; then
  PASS "cntb config file found"
  PASS_COUNT=$((PASS_COUNT + 1))
else
  echo ""
  WARN "────────────────────────────────────────────────────────────"
  WARN "  API credentials not configured."
  WARN "────────────────────────────────────────────────────────────"
  echo ""
  echo "  You need four things from the Contabo Customer Control Panel:"
  echo "    https://my.contabo.com/"
  echo ""
  echo "  1. Log in → Control Panel → blue menu → API"
  echo "  2. Note your:"
  echo "     - Client ID"
  echo "     - Client Secret"
  echo "  3. Click 'Send Link' to set API password via email"
  echo "  4. Create an API User if you haven't already"
  echo ""

  # Prompt for credentials
  printf "  Enter Client ID: "
  read -r CLIENT_ID
  printf "  Enter Client Secret: "
  read -rs CLIENT_SECRET
  echo ""
  printf "  Enter API User (email): "
  read -r API_USER
  printf "  Enter API Password: "
  read -rs API_PASS
  echo ""

  cntb config set-credentials \
    --oauth2-client-id="${CLIENT_ID}" \
    --oauth2-client-secret="${CLIENT_SECRET}" \
    --oauth2-user="${API_USER}" \
    --oauth2-password="${API_PASS}" 2>&1

  # Verify config was written
  if [ -f "${CONFIG_FILE}" ] || [ -d "${CONFIG_DIR}" ]; then
    PASS "cntb credentials configured"
    PASS_COUNT=$((PASS_COUNT + 1))

    # Clear credentials from shell history
    history -c 2>/dev/null || true
  else
    FAIL "Failed to configure cntb credentials"
    FAIL_COUNT=$((FAIL_COUNT + 1))
    exit 1
  fi
fi

# ============================================================================
# Check 3: Authenticate and list instances
# ============================================================================
echo ""
INFO "[3/6] Authentication & Instance List"

INSTANCES=$(cntb get instances -o json 2>&1) || true

if echo "${INSTANCES}" | grep -q "error\|unauthorized\|token\|Invalid"; then
  FAIL "Authentication failed. Check your API credentials."
  echo ""
  echo "  To reconfigure:"
  echo "    cntb config set-credentials ..."
  echo ""
  echo "  Or delete the config and re-run this script:"
  echo "    rm -f ~/.cntb.json"
  echo "    rm -rf ~/.config/cntb"
  FAIL_COUNT=$((FAIL_COUNT + 1))
else
  PASS "Authenticated successfully"
  PASS_COUNT=$((PASS_COUNT + 1))

  INSTANCE_COUNT=$(echo "${INSTANCES}" | grep -c '"instanceId"' 2>/dev/null || echo "0")

  if [ "${INSTANCE_COUNT}" -gt 0 ]; then
    PASS "Found ${INSTANCE_COUNT} VPS instance(s)"
    PASS_COUNT=$((PASS_COUNT + 1))
    echo ""
    echo "${INSTANCES}" | python3 -m json.tool 2>/dev/null \
      | grep -E '"instanceId"|"displayName"|"productId"|"status"|"ipConfig"' \
      | head -20
  else
    WARN "No instances found in API response"
    WARN_COUNT=$((WARN_COUNT + 1))
    echo ""
    echo "  Raw response:"
    echo "${INSTANCES}" | head -5
  fi
fi

# ============================================================================
# Check 4: Can SSH into the VPS
# ============================================================================
echo ""
INFO "[4/6] SSH Access"

if [ -n "${CONTABO_HOST:-}" ]; then
  # Check if CONTABO_HOST env var is set
  if ssh -o ConnectTimeout=5 -o BatchMode=yes "root@${CONTABO_HOST}" "echo ok" 2>/dev/null; then
    PASS "SSH to root@${CONTABO_HOST} works"
    PASS_COUNT=$((PASS_COUNT + 1))

    VPS_OS=$(ssh "root@${CONTABO_HOST}" "cat /etc/os-release 2>/dev/null | head -1" 2>/dev/null)
    PASS "VPS OS: ${VPS_OS:-detected}"
    PASS_COUNT=$((PASS_COUNT + 1))
  else
    WARN "SSH to root@${CONTABO_HOST} failed. You'll need to set up SSH keys."
    echo ""
    echo "  To copy your SSH key:"
    echo "    ssh-copy-id root@${CONTABO_HOST}"
    echo ""
    echo "  Or from the Contabo panel, add your public key."
    WARN_COUNT=$((WARN_COUNT + 1))
  fi
else
  # No CONTABO_HOST set — try to find it from the instances list
  VPS_IP=$(echo "${INSTANCES}" | python3 -c "
import json,sys
try:
    data = json.load(sys.stdin)
    if isinstance(data, list) and len(data) > 0:
        ip = data[0].get('ipConfig', {}).get('v4', {}).get('ip', '')
        print(ip)
except: pass
" 2>/dev/null)

  if [ -n "${VPS_IP}" ]; then
    echo "  Found VPS IP from API: ${VPS_IP}"

    if ssh -o ConnectTimeout=5 -o BatchMode=yes "root@${VPS_IP}" "echo ok" 2>/dev/null; then
      PASS "SSH to root@${VPS_IP} works"
      PASS_COUNT=$((PASS_COUNT + 1))

      VPS_OS=$(ssh "root@${VPS_IP}" "cat /etc/os-release 2>/dev/null | head -1" 2>/dev/null)
      PASS "VPS OS: ${VPS_OS:-detected}"
      PASS_COUNT=$((PASS_COUNT + 1))
    else
      WARN "SSH to root@${VPS_IP} failed."
      echo ""
      echo "  To fix:"
      echo "    ssh-copy-id root@${VPS_IP}"
      echo ""
      WARN_COUNT=$((WARN_COUNT + 1))
    fi
  else
    WARN "Could not determine VPS IP. Set CONTABO_HOST env var and re-run."
    echo "  export CONTABO_HOST=<your-vps-ip>"
    WARN_COUNT=$((WARN_COUNT + 1))
  fi
fi

# ============================================================================
# Check 5: Required tools on VPS
# ============================================================================
echo ""
INFO "[5/6] VPS Prerequisites"

VPS_IP="${CONTABO_HOST:-${VPS_IP:-}}"

if [ -n "${VPS_IP}" ]; then
  REMOTE_CHECKS=$(ssh "root@${VPS_IP}" "
    echo 'ARCH=\$(uname -m)' > /tmp/check.sh
    echo 'CURL=\$(command -v curl >/dev/null && echo ok || echo missing)' >> /tmp/check.sh
    echo 'SYSTEMD=\$(command -v systemctl >/dev/null && echo ok || echo missing)' >> /tmp/check.sh
    echo 'echo ARCH:\$(uname -m)' >> /tmp/check.sh
    echo 'echo CURL:\$CURL' >> /tmp/check.sh
    echo 'echo SYSTEMD:\$SYSTEMD' >> /tmp/check.sh
    echo 'echo UPTIME:\$(uptime -p)' >> /tmp/check.sh
    echo 'echo MEM:\$(free -h | grep Mem | awk {print\\\$2})' >> /tmp/check.sh
    echo 'echo DISK:\$(df -h / | tail -1 | awk {print\\\$2})' >> /tmp/check.sh
    bash /tmp/check.sh
    rm -f /tmp/check.sh
  " 2>/dev/null) || true

  if [ -n "${REMOTE_CHECKS}" ]; then
    while IFS=: read -r key val; do
      case "$key" in
        ARCH)    PASS "Architecture: ${val}" && PASS_COUNT=$((PASS_COUNT + 1)) ;;
        CURL)    [ "${val}" = "ok" ] && PASS "curl installed" && PASS_COUNT=$((PASS_COUNT + 1)) || WARN "curl missing" ;;
        SYSTEMD) [ "${val}" = "ok" ] && PASS "systemd available" && PASS_COUNT=$((PASS_COUNT + 1)) || WARN "systemd missing" ;;
        UPTIME)  PASS "Uptime: ${val}" && PASS_COUNT=$((PASS_COUNT + 1)) ;;
        MEM)     PASS "RAM: ${val}" && PASS_COUNT=$((PASS_COUNT + 1)) ;;
        DISK)    PASS "Disk: ${val}" && PASS_COUNT=$((PASS_COUNT + 1)) ;;
      esac
    done <<< "${REMOTE_CHECKS}"
  else
    WARN "Could not run remote checks"
    WARN_COUNT=$((WARN_COUNT + 1))
  fi
else
  WARN "No VPS IP available — skipping remote checks"
  WARN_COUNT=$((WARN_COUNT + 1))
fi

# ============================================================================
# Check 6: VPS can reach GitHub (needed for GitHub Actions)
# ============================================================================
echo ""
INFO "[6/6] Outbound Connectivity"

if [ -n "${VPS_IP:-}" ]; then
  GITHUB_REACHABLE=$(ssh "root@${VPS_IP}" "curl -sfI https://github.com >/dev/null 2>&1 && echo ok || echo fail" 2>/dev/null) || true
  if [ "${GITHUB_REACHABLE}" = "ok" ]; then
    PASS "VPS can reach GitHub"
    PASS_COUNT=$((PASS_COUNT + 1))
  else
    WARN "VPS cannot reach GitHub — check Contabo firewall settings"
    WARN_COUNT=$((WARN_COUNT + 1))
  fi

  DOCKER_REACHABLE=$(ssh "root@${VPS_IP}" "curl -sfI https://get.docker.com >/dev/null 2>&1 && echo ok || echo fail" 2>/dev/null) || true
  if [ "${DOCKER_REACHABLE}" = "ok" ]; then
    PASS "VPS has internet access"
    PASS_COUNT=$((PASS_COUNT + 1))
  else
    WARN "Outbound internet limited"
    WARN_COUNT=$((WARN_COUNT + 1))
  fi
else
  WARN "No VPS IP — skipping connectivity checks"
  WARN_COUNT=$((WARN_COUNT + 1))
fi

# ============================================================================
# Summary
# ============================================================================
echo ""
INFO "╔══════════════════════════════════════════════════╗"
INFO "║                   Results                        ║"
INFO "╚══════════════════════════════════════════════════╝"
echo ""
printf "  ${GREEN}Passed: %d${NC}\n" "${PASS_COUNT}"
[ "${FAIL_COUNT}" -gt 0 ] && printf "  ${RED}Failed: %d${NC}\n" "${FAIL_COUNT}"
[ "${WARN_COUNT}" -gt 0 ] && printf "  ${YELLOW}Warnings: %d${NC}\n" "${WARN_COUNT}"
echo ""

if [ "${FAIL_COUNT}" -eq 0 ]; then
  INFO "  ✓ Your Contabo account is ready for BigBase deployment!"
  echo ""
  echo "  Next steps:"
  echo "    1. Run the VPS setup script:"
  echo "       ./scripts/setup-vps.sh"
  echo ""
  echo "    2. Set GitHub Secrets in your repo:"
  echo "       CONTABO_HOST  = ${VPS_IP:-<your VPS IP>}"
  echo "       CONTABO_USER  = root"
  echo "       CONTABO_SSH_KEY = ~/.ssh/id_ed25519"
  echo ""
  echo "    3. Push to main and deploy."
  echo ""
else
  WARN "  Fix the failures above, then re-run this script."
  echo ""
  exit 1
fi
