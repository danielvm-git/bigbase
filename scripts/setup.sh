#!/usr/bin/env bash
set -euo pipefail

# Idempotent one-command setup for the BigBase development environment (e90s05).
# Fresh clone -> ./scripts/setup.sh -> developer ready.
# Safe to run repeatedly. `scripts/setup-dev.sh` is a thin alias for this script.

BIGBASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_VERSION_MIN="1.26"   # keep in sync with go.mod `go` directive

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info() { printf "${GREEN}%s${NC}\n" "$1"; }
warn() { printf "${YELLOW}WARNING: %s${NC}\n" "$1"; }

cd "$BIGBASE_DIR"

# --- Go ----------------------------------------------------------------------
info "Checking Go..."
if ! command -v go &>/dev/null; then
  warn "Go is not installed. Install Go $GO_VERSION_MIN+ from https://go.dev/dl/"; exit 1
fi
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
info "Go $GO_VERSION found (min $GO_VERSION_MIN)"

# --- golangci-lint -----------------------------------------------------------
info "Checking golangci-lint..."
if ! command -v golangci-lint &>/dev/null; then
  warn "golangci-lint not found. Install: brew install golangci-lint (or 'go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.0')"
else
  info "golangci-lint: $(golangci-lint --version | head -1)"
fi

# --- git hooks (versioned in .githooks/) -------------------------------------
if [ -d "$BIGBASE_DIR/.githooks" ]; then
  if [ "$(git config core.hooksPath 2>/dev/null || true)" != ".githooks" ]; then
    git config core.hooksPath .githooks
    info "Configured git hooksPath=.githooks"
  else
    info "git hooksPath already .githooks"
  fi
fi

# --- BIGBASE_ROOT in .envrc (LOCAL, gitignored) ------------------------------
# The concrete absolute repo path is a per-machine concern, so it lives in the
# local .envrc (direnv), never in committed config. Committed config references
# it portably as ${BIGBASE_ROOT} (see .mcp.json).
ENVRC="$BIGBASE_DIR/.envrc"
if ! grep -qs 'BIGBASE_ROOT=' "$ENVRC" 2>/dev/null; then
  printf 'export BIGBASE_ROOT=%q\n' "$BIGBASE_DIR" >> "$ENVRC"
  info "Wrote BIGBASE_ROOT to .envrc (run 'direnv allow' if you use direnv)"
fi

# --- Generate MCP adapter configs from canonical .mcp.json -------------------
if [ -x "$BIGBASE_DIR/scripts/gen-mcp-configs.sh" ]; then
  info "Generating MCP adapter configs..."
  bash "$BIGBASE_DIR/scripts/gen-mcp-configs.sh"
fi

# --- Dependencies: Go modules + npm (root + ui) ------------------------------
info "Downloading Go modules..."
go mod download

if [ -f "$BIGBASE_DIR/package-lock.json" ]; then
  info "Installing root npm deps..."
  npm ci --no-audit --no-fund || warn "root npm ci failed (continuing)"
fi
if [ -d "$BIGBASE_DIR/ui" ]; then
  info "Building admin UI..."
  (cd "$BIGBASE_DIR/ui" && npm ci --no-audit --no-fund && npm run build)
  info "Admin UI build OK"
fi

# --- Build binary with VERSION injection -------------------------------------
BUILD_VERSION="0.0.0-dev"
[ -f "$BIGBASE_DIR/VERSION" ] && BUILD_VERSION=$(cat "$BIGBASE_DIR/VERSION")
info "Building bigbase (version: ${BUILD_VERSION})..."
go build -ldflags "-X github.com/danielvm/bigbase/kernel.Version=${BUILD_VERSION}" -o bigbase .
info "Build OK"

# --- Initial ctxo index (hooks keep it warm afterward) -----------------------
info "Building initial ctxo index (best-effort)..."
npx -y @ctxo/cli@0.11.4 index >/dev/null 2>&1 && info "ctxo index built" \
  || warn "ctxo index skipped (offline or ctxo unavailable) — hooks will build it on first commit"

# --- Doctor summary ----------------------------------------------------------
info ""
info "===== doctor ====="
printf "  go             %s\n" "$(go version 2>/dev/null | awk '{print $3}')"
printf "  node           %s\n" "$(command -v node >/dev/null && node --version || echo 'MISSING')"
printf "  npm            %s\n" "$(command -v npm  >/dev/null && npm --version  || echo 'MISSING')"
printf "  golangci-lint  %s\n" "$(command -v golangci-lint >/dev/null && golangci-lint --version | awk '{print $4}' || echo 'MISSING')"
printf "  jq             %s\n" "$(command -v jq >/dev/null && jq --version || echo 'MISSING')"
printf "  git hooksPath  %s\n" "$(git config core.hooksPath 2>/dev/null || echo 'unset')"
if [ -d "$BIGBASE_DIR/.ctxo" ]; then
  IDX_AGE=$(find "$BIGBASE_DIR/.ctxo" -name '*.json' -type f -newermt '-1 day' 2>/dev/null | head -1)
  printf "  ctxo index     %s\n" "$([ -n "$IDX_AGE" ] && echo 'fresh (<1d)' || echo 'stale or absent')"
fi
info "=================="
info ""
info "BigBase development environment is ready."
