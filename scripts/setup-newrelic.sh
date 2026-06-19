#!/usr/bin/env bash
# Idempotent New Relic Infrastructure Agent setup for BigBase.
# Prerequisites: NEW_RELIC_API_KEY and NEW_RELIC_ACCOUNT_ID set as env vars
# (or stored in /etc/newrelic-infra.yml after first run).
#
# Usage:
#   NEW_RELIC_API_KEY=NRAK-... NEW_RELIC_ACCOUNT_ID=12345 bash scripts/setup-newrelic.sh
#
# Optional env vars:
#   NEW_RELIC_REGION    — US (default) or EU
#   NEW_RELIC_LICENSE_KEY — alternative to API key (legacy)

set -euo pipefail

BIGBASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()  { printf "${GREEN}%s${NC}\n" "$1"; }
warn()  { printf "${YELLOW}%s${NC}\n" "$1"; }
err()   { printf "${RED}ERROR: %s${NC}\n" "$1"; }

# --- Check for required env vars ---
API_KEY="${NEW_RELIC_API_KEY:-}"
ACCOUNT_ID="${NEW_RELIC_ACCOUNT_ID:-}"
REGION="${NEW_RELIC_REGION:-US}"

if [ -z "$API_KEY" ] || [ -z "$ACCOUNT_ID" ]; then
  err "NEW_RELIC_API_KEY and NEW_RELIC_ACCOUNT_ID must be set."
  echo ""
  echo "  Get these from https://one.newrelic.com > API Keys"
  echo "  Then run:"
  echo "    NEW_RELIC_API_KEY=NRAK-... NEW_RELIC_ACCOUNT_ID=12345 bash scripts/setup-newrelic.sh"
  exit 1
fi

# --- Check if infrastructure agent is already installed ---
if command -v newrelic-infra &>/dev/null; then
  info "New Relic infrastructure agent already installed: $(newrelic-infra -version 2>/dev/null || echo 'unknown')"
  info "Agent is running — no action needed."
  exit 0
fi

# --- Check if New Relic CLI is installed ---
NR_CLI="/usr/local/bin/newrelic"
if ! command -v "$NR_CLI" &>/dev/null; then
  info "Installing New Relic CLI..."
  curl -Ls https://download.newrelic.com/install/newrelic-cli/scripts/install.sh | bash
  info "New Relic CLI installed."
else
  info "New Relic CLI already installed."
fi

# --- Install infrastructure agent via guided install ---
info "Installing New Relic infrastructure agent..."
if [ "$REGION" = "EU" ]; then
  sudo env NEW_RELIC_API_KEY="$API_KEY" \
    NEW_RELIC_ACCOUNT_ID="$ACCOUNT_ID" \
    NEW_RELIC_REGION="$REGION" \
    "$NR_CLI" install -n infrastructure-agent-installer -y
else
  sudo env NEW_RELIC_API_KEY="$API_KEY" \
    NEW_RELIC_ACCOUNT_ID="$ACCOUNT_ID" \
    "$NR_CLI" install -n infrastructure-agent-installer -y
fi

info ""
info "New Relic infrastructure agent installed."
info "View your data at https://one.newrelic.com > Infrastructure > Hosts"
info ""
info "To add PostgreSQL monitoring (if using PostgreSQL):"
info "  sudo NEW_RELIC_API_KEY=NRAK-... NEW_RELIC_ACCOUNT_ID=12345 $NR_CLI install -n postgres-open-source-integration"
