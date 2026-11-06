#!/usr/bin/env bash
# story: e90s02
# check-conflict-markers.sh — fail when an unresolved merge-conflict marker is
# present in tracked source or config files.
#
# Usage:
#   bash scripts/check-conflict-markers.sh            # scan git-tracked files
#   bash scripts/check-conflict-markers.sh FILE...    # scan explicit files
#
# CHANGELOG.md and specs/archive/** are exempt in git mode: setext H1
# underlines ("=======") are legitimate there. Marker-like text that is not at
# the start of a line is ignored, so documentation samples do not trip the gate.
#
# Set BIGBASE_CONFLICT_ROOT to scan a directory other than this repository
# (used by scripts/check-conflict-markers.test.cjs fixtures).
#
# Exit 0 = no markers found, 1 = at least one marker found.
set -euo pipefail

# Mirrors the CI contract: a marker must start its own line.
pattern='^(<<<<<<<|>>>>>>>)( |$)|^=======$'

if [ "$#" -gt 0 ]; then
  # Explicit paths need no git context (used by the fixture tests).
  matches=$(grep -nHIE -- "$pattern" "$@" || true)
else
  root="${BIGBASE_CONFLICT_ROOT:-$(git rev-parse --show-toplevel)}"
  cd "$root"
  matches=$(git grep -n -E -- "$pattern" -- ':!CHANGELOG.md' ':!specs/archive/**' || true)
fi

if [ -n "$matches" ]; then
  echo "❌ unresolved merge-conflict marker found:"
  echo "$matches"
  echo ""
  echo "Resolve the conflict, then re-run: bash scripts/check-conflict-markers.sh"
  exit 1
fi

echo "✅ No unresolved merge-conflict markers"
