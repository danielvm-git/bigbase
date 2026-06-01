#!/usr/bin/env bash
set -euo pipefail

# Idempotent setup script for BigBase development environment.
# Can be run multiple times without side effects.

BIGBASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_VERSION_MIN="1.22"

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

# --- golangci-lint ---
info "Checking golangci-lint..."
if ! command -v golangci-lint &>/dev/null; then
  warn "golangci-lint not found. Install: brew install golangci-lint"
  exit 1
fi
info "golangci-lint found: $(golangci-lint --version | head -1)"

# --- Go module dependencies ---
info "Downloading Go module dependencies..."
(cd "$BIGBASE_DIR" && go mod download)

# --- Build admin UI ---
if [ -d "$BIGBASE_DIR/ui" ]; then
  info "Building admin UI..."
  (cd "$BIGBASE_DIR/ui" && npm ci && npm run build)
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
