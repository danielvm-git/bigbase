#!/bin/bash
# Fails if any commit between origin/main and HEAD doesn't follow Conventional
# Commits format (feat/fix/docs/style/refactor/perf/test/build/ci/chore/revert).
# Usage: bash scripts/check-conventional-commits.sh
# Extracted from .github/workflows/ci-cd.yml's verify job (BigBase v3 CI/CD restructure).

set -euo pipefail

NON_CC=$(git log origin/main..HEAD --no-merges --format="%s" | grep -v -c -E "^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(.+\))?!?: " || true)

if [ "$NON_CC" -gt 0 ]; then
  echo "FAIL: $NON_CC commits do not follow Conventional Commits"
  git log origin/main..HEAD --no-merges --format="%s" | grep -v -E "^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(.+\))?!?: "
  exit 1
fi

echo "PASS: All commits follow Conventional Commits"
