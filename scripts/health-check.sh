#!/usr/bin/env bash
set -euo pipefail

# Health check script for BigBase.
# Starts the server, curls /health, validates JSON output, then cleans up.

BIGBASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BINARY="${BIGBASE_DIR}/bigbase"
PORT=${PORT:-19999}
DB="${DB:-:memory:}"

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

pass() { printf "${GREEN}PASS${NC} %s\n" "$1"; }
fail() { printf "${RED}FAIL${NC} %s\n" "$1"; exit 1; }

# Ensure binary exists
if [ ! -x "$BINARY" ]; then
  echo "Building bigbase binary..."
  (cd "$BIGBASE_DIR" && go build -o bigbase .)
fi

# Start server in background
$BINARY serve --port "$PORT" --db "$DB" &
PID=$!
trap 'kill $PID 2>/dev/null; wait $PID 2>/dev/null' EXIT

# Wait for server to be ready
for i in $(seq 1 10); do
  if curl -sf "http://localhost:$PORT/health" >/dev/null 2>&1; then
    break
  fi
  if [ "$i" -eq 10 ]; then
    fail "server did not start within 10 seconds"
  fi
  sleep 0.5
done

# Fetch health endpoint
BODY=$(curl -sf "http://localhost:$PORT/health")
if [ -z "$BODY" ]; then
  fail "empty response from /health"
fi
pass "/health returned non-empty response"

# Validate JSON
if ! echo "$BODY" | python3 -c "import json,sys; json.load(sys.stdin)" 2>/dev/null; then
  fail "response is not valid JSON"
fi
pass "response is valid JSON"

# Validate required fields
for field in status components running; do
  if ! echo "$BODY" | python3 -c "
import json,sys
data = json.load(sys.stdin)
assert '$field' in data, 'missing field: $field'
" 2>/dev/null; then
    fail "missing field '$field' in /health JSON"
  fi
done
pass "all required fields (status, components, running) present"

# Validate health is ok
STATUS=$(echo "$BODY" | python3 -c "import json,sys; print(json.load(sys.stdin)['status'])")
if [ "$STATUS" != "ok" ]; then
  fail "expected status=ok, got $STATUS"
fi
pass "status is ok"

# Check component_status_map exists (e24s03 feature)
if echo "$BODY" | python3 -c "
import json,sys
data = json.load(sys.stdin)
assert 'component_status_map' in data, 'missing component_status_map'
components = data['component_status_map']
assert len(components) > 0, 'component_status_map is empty'
" 2>/dev/null; then
  pass "component_status_map present with entries"
fi

echo ""
echo "All health checks passed."
