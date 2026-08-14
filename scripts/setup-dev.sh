#!/usr/bin/env bash
# setup-dev.sh — One-command developer onboarding for a fresh BigBase clone.
#
# Usage: ./scripts/setup-dev.sh
#
# Idempotent: safe to re-run. Prints a status summary at the end.

set -euo pipefail

OK="✓"
WARN="⚠"
FAIL="✗"
PASS=true

hr() { printf '%0.s─' {1..60}; echo; }
step() { echo; echo "▸ $1"; }

hr
echo "  BigBase dev setup"
hr

# ── 1. Git hooks ─────────────────────────────────────────────────────────────
step "Git hooks (core.hooksPath)"
git config core.hooksPath .githooks
echo "  $OK core.hooksPath = .githooks"
chmod +x .githooks/* 2>/dev/null || true

# ── 2. MCP adapter files ──────────────────────────────────────────────────────
step "MCP adapter files"
bash scripts/gen-mcp-configs.sh

# ── 3. Go toolchain ───────────────────────────────────────────────────────────
step "Go toolchain"
REQUIRED_GO="1.26"
if command -v go > /dev/null 2>&1; then
  GOVERSION=$(go version | awk '{print $3}' | sed 's/go//')
  echo "  $OK go $GOVERSION"
else
  echo "  $FAIL go not found — install from https://go.dev/dl/"
  PASS=false
fi

# ── 4. Node toolchain ─────────────────────────────────────────────────────────
step "Node toolchain"
REQUIRED_NODE="24"
if command -v node > /dev/null 2>&1; then
  NODEVERSION=$(node --version | sed 's/v//')
  NODEMAJOR=$(echo "$NODEVERSION" | cut -d. -f1)
  if [[ "$NODEMAJOR" -ge "$REQUIRED_NODE" ]]; then
    echo "  $OK node v$NODEVERSION"
  else
    echo "  $WARN node v$NODEVERSION (expected >=$REQUIRED_NODE — use .nvmrc: nvm use)"
  fi
else
  echo "  $FAIL node not found — install via nvm or https://nodejs.org"
  PASS=false
fi

# ── 5. Root npm install ───────────────────────────────────────────────────────
step "npm install (root)"
npm install --silent --legacy-peer-deps
echo "  $OK root dependencies installed"

# ── 6. UI npm install ─────────────────────────────────────────────────────────
step "npm install (ui/)"
(cd ui && npm install --silent --legacy-peer-deps)
echo "  $OK ui/ dependencies installed"

# ── 7. Optional tools ─────────────────────────────────────────────────────────
step "Optional tools"
for tool in ctxo golangci-lint bts opensrc sqz-mcp; do
  if command -v "$tool" > /dev/null 2>&1; then
    echo "  $OK $tool"
  else
    echo "  $WARN $tool not found (optional)"
  fi
done

# ── 8. ctxo index warm-up ────────────────────────────────────────────────────
step "ctxo index"
if command -v ctxo > /dev/null 2>&1; then
  echo "  Indexing codebase (background)…"
  ctxo index > /dev/null 2>&1 &
  echo "  $OK ctxo index started"
else
  echo "  $WARN ctxo not installed — skipping index warm-up"
fi

# ── Summary ──────────────────────────────────────────────────────────────────
hr
if [[ "$PASS" == "true" ]]; then
  echo "  $OK Setup complete — you're ready to code."
else
  echo "  $WARN Setup complete with warnings. Fix the $FAIL items above."
fi
hr
