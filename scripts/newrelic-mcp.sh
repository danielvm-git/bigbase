#!/usr/bin/env bash
set -euo pipefail

# New Relic MCP wrapper — injects API key from environment.
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
MCP_REMOTE="${MCP_REMOTE_BIN:-$(command -v mcp-remote || true)}"
if [ -z "$MCP_REMOTE" ] && [ -x "$ROOT_DIR/node_modules/.bin/mcp-remote" ]; then
  MCP_REMOTE="$ROOT_DIR/node_modules/.bin/mcp-remote"
fi
if [ -z "$MCP_REMOTE" ]; then
  echo "mcp-remote is not installed; run scripts/setup.sh first" >&2
  exit 1
fi

exec "$MCP_REMOTE" \
  "https://mcp.newrelic.com/mcp/" \
  --header "Api-Key: ${NEW_RELIC_API_KEY:?NEW_RELIC_API_KEY not set}"
