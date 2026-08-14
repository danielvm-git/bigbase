#!/usr/bin/env bash
# gen-mcp-configs.sh — Generate per-tool MCP adapter files from .mcp.json (canonical source).
#
# Usage:
#   scripts/gen-mcp-configs.sh          # generate all adapters
#   scripts/gen-mcp-configs.sh --check  # verify adapters match; exit 1 if stale
#
# Adapters are written with a GENERATED header. Edit .mcp.json — not the adapters.
# Run this script (or `make mcp`) after any .mcp.json change, and on first clone.

set -euo pipefail

CANONICAL=".mcp.json"
CHECK_MODE=false
FAILED=false

if [[ "${1:-}" == "--check" ]]; then
  CHECK_MODE=true
fi

HEADER='// GENERATED — do not edit. Source: .mcp.json. Run: scripts/gen-mcp-configs.sh'

generate() {
  local dest="$1"
  local content="$2"
  local generated="${HEADER}"$'\n'"${content}"

  if [[ "$CHECK_MODE" == "true" ]]; then
    if [[ ! -f "$dest" ]]; then
      echo "  STALE (missing): $dest"
      FAILED=true
      return
    fi
    existing=$(cat "$dest")
    if [[ "$existing" != "$generated" ]]; then
      echo "  STALE (differs): $dest"
      FAILED=true
    else
      echo "  OK: $dest"
    fi
  else
    mkdir -p "$(dirname "$dest")"
    printf '%s\n' "$generated" > "$dest"
    echo "  wrote: $dest"
  fi
}

if ! command -v jq > /dev/null 2>&1; then
  echo "ERROR: jq is required. Install with: brew install jq"
  exit 1
fi

echo "gen-mcp-configs: source = $CANONICAL"

# ── .claude/mcp.json ────────────────────────────────────────────────────────
# Claude Code uses the same schema as .mcp.json; copy verbatim (minus header).
# Note: .claude/ is gitignored — local only, not CI-checked.
CLAUDE_CONTENT=$(cat "$CANONICAL")
if [[ "$CHECK_MODE" == "false" ]]; then
  generate ".claude/mcp.json" "$CLAUDE_CONTENT"
fi

# ── .cursor/mcp.json ────────────────────────────────────────────────────────
# Note: .cursor/ is gitignored — local only, not CI-checked.
if [[ "$CHECK_MODE" == "false" ]]; then
  generate ".cursor/mcp.json" "$CLAUDE_CONTENT"
fi

# ── opencode.jsonc ──────────────────────────────────────────────────────────
# opencode.jsonc IS tracked in git — CI checks this one for drift.
OPENCODE_SERVERS=$(jq '{mcp: {servers: .mcpServers}}' "$CANONICAL")
generate "opencode.jsonc" "$OPENCODE_SERVERS"

if [[ "$CHECK_MODE" == "true" ]]; then
  if [[ "$FAILED" == "true" ]]; then
    echo ""
    echo "ERROR: One or more adapter files are stale. Run: scripts/gen-mcp-configs.sh"
    exit 1
  else
    echo "All adapters are up-to-date."
  fi
else
  echo "Done."
fi
