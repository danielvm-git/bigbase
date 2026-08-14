#!/usr/bin/env bash
set -euo pipefail

# Idempotent setup script for BigBase development environment.
# Can be run multiple times without side effects.

BIGBASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_VERSION_MIN="1.26.3"
GOPLS_VERSION="v0.23.0"
GO_BIN_DIR="$(go env GOPATH 2>/dev/null || printf '%s' "$HOME/go")/bin"
export PATH="$GO_BIN_DIR:$PATH"
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { printf "${GREEN}%s${NC}\n" "$1"; }
warn()  { printf "${YELLOW}WARNING: %s${NC}\n" "$1"; }

# --- Go version check ---
info "Checking Go version..."
if ! command -v go &>/dev/null; then
  warn "Go is not installed. Install Go $GO_VERSION_MIN+ from https://go.dev/dl/"
  exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
info "Go $GO_VERSION found"

# --- gopls ---
info "Checking gopls..."
if ! command -v gopls &>/dev/null; then
  info "Installing gopls ${GOPLS_VERSION}..."
  go install "golang.org/x/tools/gopls@${GOPLS_VERSION}"
fi
info "gopls found: $(gopls version | head -1)"

# --- npm dependencies ---
info "Installing root npm dependencies..."
(cd "$BIGBASE_DIR" && npm ci --ignore-scripts)
info "typescript-language-server found: $(cd "$BIGBASE_DIR" && npx --no-install typescript-language-server --version)"

# --- golangci-lint ---
info "Checking golangci-lint..."
if ! command -v golangci-lint &>/dev/null; then
  warn "golangci-lint not found. Install it with your platform package manager."
  exit 1
fi
info "golangci-lint found: $(golangci-lint --version | head -1)"
# --- Go module dependencies ---
info "Downloading Go module dependencies..."
(cd "$BIGBASE_DIR" && go mod download)

# --- Build admin UI ---
if [ -d "$BIGBASE_DIR/ui" ]; then
  info "Building admin UI..."
  (cd "$BIGBASE_DIR/ui" && npm ci --legacy-peer-deps && npm run build)
  info "Admin UI build OK"
fi

# --- Build binary with VERSION injection ---
VERSION_FILE="$BIGBASE_DIR/VERSION"
if [ -f "$VERSION_FILE" ]; then
  BUILD_VERSION=$(cat "$VERSION_FILE")
else
  BUILD_VERSION="0.0.0-dev"
fi
info "Building bigbase (version: ${BUILD_VERSION})..."
(cd "$BIGBASE_DIR" && go build -ldflags "-X github.com/danielvm/bigbase/kernel.Version=${BUILD_VERSION}" -o bigbase .)
info "Build OK"

# --- Create config/ if missing ---
if [ ! -d "$BIGBASE_DIR/config" ]; then
  mkdir -p "$BIGBASE_DIR/config"
  info "Created config/ directory"
fi

# --- Create specs/ if missing ---
if [ ! -d "$BIGBASE_DIR/specs" ]; then
  mkdir -p "$BIGBASE_DIR/specs"
  info "Created specs/ directory"
fi

# --- Verify pre-commit hook ---
HOOKS_DIR="$BIGBASE_DIR/.githooks"
if [ -d "$HOOKS_DIR" ]; then
  HOOK_PATH=$(git -C "$BIGBASE_DIR" config core.hooksPath 2>/dev/null || true)
  if [ "$HOOK_PATH" != ".githooks" ]; then
    git -C "$BIGBASE_DIR" config core.hooksPath .githooks
    info "Configured git hooksPath=.githooks"
  fi
fi

info ""
info "BigBase development environment is ready."

# --- ctxo codebase index (optional) ---
# Warm the index so MCP clients (AGENTS.md ctxo workflow) have data on a
# fresh clone instead of waiting for the first source commit. Failure is
# non-fatal — pre-commit/post-merge rebuild it automatically.
info "Checking ctxo..."
if command -v ctxo &>/dev/null; then
  info "Warming ctxo index (first build may take a minute)..."
  ctxo index >/dev/null 2>&1 || warn "ctxo index failed — it will rebuild on the next source commit"
else
  warn "ctxo not installed — skip index warm-up (npm i -g @ctxo/cli to enable)"
fi
