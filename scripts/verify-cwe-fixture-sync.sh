#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
fixture_root="${SECURITY_REVIEW_SKILLS_ROOT:-$HOME/.claude/skills/security-review}/fixtures"
if [[ ! -d "$fixture_root" ]]; then
  fixture_root="$root/skills/security-review/fixtures"
fi

required=(
  CWE-89-sqli-positive.py CWE-89-sqli-negative.py
  CWE-79-xss-positive.js CWE-79-xss-negative.js
  CWE-639-idor-positive.go CWE-639-idor-negative.go
  CWE-fail-open-verify-positive.sh CWE-fail-open-verify-negative.sh
)
for fixture in "${required[@]}"; do
  test -f "$fixture_root/$fixture" || { echo "missing fixture: $fixture" >&2; exit 1; }
done
printf 'CWE fixture sync: OK (%s fixtures)\n' "${#required[@]}"
