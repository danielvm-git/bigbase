#!/usr/bin/env bash
set -euo pipefail

# observatory-check.sh — HTTP security header audit
#
# Audits HTTP security headers served by TARGET_URL using curl.
# Reports which headers are present/missing and their values.
#
# Usage:
#   TARGET_URL=https://example.com bash scripts/observatory-check.sh
#   TARGET_URL=http://localhost:9999 bash scripts/observatory-check.sh
#
# Environment:
#   TARGET_URL — Required. URL to audit (e.g. http://localhost:9999)

usage() {
  cat <<EOF
Usage: TARGET_URL=<url> bash $0

Audits HTTP security headers served by TARGET_URL.

Environment:
  TARGET_URL   Required. URL to audit.

Exit codes:
  0 — Always (informational only)
EOF
  exit 0
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
fi

if [[ -z "${TARGET_URL:-}" ]]; then
  echo "ERROR: TARGET_URL is required."
  echo "Usage: TARGET_URL=<url> bash $0"
  exit 1
fi

echo "=== Mozilla Observatory-Style HTTP Header Audit ==="
echo "Target: $TARGET_URL"
echo ""

# Fetch response headers
HEADERS=$(curl -sIL --max-time 10 "$TARGET_URL" 2>&1) || {
  echo "ERROR: Failed to fetch headers from $TARGET_URL"
  exit 1
}

# Expected security headers
declare -A EXPECTED=(
  ["Content-Security-Policy"]="present"
  ["Strict-Transport-Security"]="present"
  ["X-Frame-Options"]="present"
  ["X-Content-Type-Options"]="present"
  ["Referrer-Policy"]="present"
  ["Permissions-Policy"]="present"
)

declare -A OPTIONAL=(
  ["Cache-Control"]="recommended for API routes"
  ["X-Request-ID"]="recommended for tracing"
)

echo "--- Required Headers ---"
PASS=0
FAIL=0
for header in "${!EXPECTED[@]}"; do
  value=$(echo "$HEADERS" | grep -i "^$header:" | sed 's/^[^:]*: //' | head -1)
  if [[ -n "$value" ]]; then
    echo "  ✅ $header: $value"
    PASS=$((PASS + 1))
  else
    echo "  ❌ $header: MISSING"
    FAIL=$((FAIL + 1))
  fi
done

echo ""
echo "--- Optional Headers ---"
for header in "${!OPTIONAL[@]}"; do
  value=$(echo "$HEADERS" | grep -i "^$header:" | sed 's/^[^:]*: //' | head -1)
  if [[ -n "$value" ]]; then
    echo "  ✅ $header: $value"
  else
    echo "  ⚠️  $header: MISSING (${OPTIONAL[$header]})"
  fi
done

echo ""
echo "--- Additional Headers ---"
echo "$HEADERS" | grep -iE '^(X-|CF-|Server|via)' | sort -u || echo "  (none found)"

echo ""
echo "=== Summary ==="
echo "  Required headers: $PASS present / $((PASS + FAIL)) total"
if [[ $FAIL -gt 0 ]]; then
  echo "  ⚠️  $FAIL required header(s) missing!"
else
  echo "  ✅ All required security headers present."
fi

exit 0
