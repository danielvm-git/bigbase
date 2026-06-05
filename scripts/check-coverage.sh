#!/bin/bash
# Coverage gate for BigBase. Fails if total coverage is below threshold.
# Usage: bash scripts/check-coverage.sh [threshold]
# Default threshold: 60 (percent)

set -euo pipefail

THRESHOLD="${1:-60}"
COVERAGE_FILE=$(mktemp)

trap 'rm -f "$COVERAGE_FILE"' EXIT

go test -coverprofile="$COVERAGE_FILE" -count=1 ./... > /dev/null 2>&1

TOTAL=$(go tool cover -func="$COVERAGE_FILE" | grep '^total:' | awk '{print $NF}' | sed 's/%//')

echo "Total coverage: ${TOTAL}% (threshold: ${THRESHOLD}%)"

if (( $(echo "$TOTAL < $THRESHOLD" | bc -l) )); then
    echo "FAIL: Coverage ${TOTAL}% is below threshold ${THRESHOLD}%"
    exit 1
fi

echo "PASS: Coverage meets threshold"
