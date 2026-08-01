#!/bin/bash
# Fails if any commit between origin/main and HEAD carries a Co-authored-by
# footer from an AI agent other than Cursor (blocked by CONVENTIONS.md).
# Usage: bash scripts/check-no-ai-attribution.sh
# Extracted from .github/workflows/ci-cd.yml's verify job (BigBase v3 CI/CD restructure).

set -euo pipefail

NON_CURSOR=$(git log origin/main..HEAD --format="%B" | grep -iE '^co[- ]authored[- ]by:' | grep -v -i 'cursor' | wc -l | tr -d ' ' || true)

if [ "$NON_CURSOR" -gt 0 ]; then
  echo "FAIL: Non-Cursor Co-authored-by: footer found — blocked by CONVENTIONS.md"
  git log origin/main..HEAD --format="%B" | grep -iE '^co[- ]authored[- ]by:' | grep -v -i 'cursor'
  exit 1
fi

echo "PASS: No AI agent attribution"
