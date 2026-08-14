#!/usr/bin/env bash
# check-version-drift.sh — Assert that Go and Node version claims in docs
# match the actual go.mod toolchain and .nvmrc declarations.
#
# Fails with a non-zero exit and a list of mismatches. Used in CI verify job.

set -euo pipefail

FAILED=false

# ── Source of truth ───────────────────────────────────────────────────────────
GOMOD_VERSION=$(grep '^go ' go.mod | awk '{print $2}')        # e.g. 1.26.3
GOMOD_MAJOR_MINOR=$(echo "$GOMOD_VERSION" | cut -d. -f1,2)    # e.g. 1.26
NVMRC_VERSION=$(cat .nvmrc | tr -d 'v\n')                     # e.g. 24

fail() {
  echo "  DRIFT: $1"
  FAILED=true
}

ok() {
  echo "  OK:    $1"
}

echo "check-version-drift: go.mod=$GOMOD_VERSION  .nvmrc=v$NVMRC_VERSION"
echo ""

# ── Go version checks ─────────────────────────────────────────────────────────
check_go() {
  local file="$1"
  local pattern="$2"  # regex that should NOT match (i.e., stale version)
  if [[ ! -f "$file" ]]; then return; fi
  # Check that the file doesn't claim a Go version older than GOMOD_MAJOR_MINOR
  if grep -qP "$pattern" "$file" 2>/dev/null || grep -qE "$pattern" "$file" 2>/dev/null; then
    fail "$file claims stale Go version (expected $GOMOD_MAJOR_MINOR)"
  else
    ok "$file Go version"
  fi
}

# Files that must not reference Go < 1.26
for f in README.md CONTRIBUTING.md; do
  if [[ -f "$f" ]]; then
    if grep -qE 'Go [0-9]+\.[0-9]+' "$f"; then
      CLAIMED=$(grep -oE 'Go [0-9]+\.[0-9]+(\.[0-9]+)?' "$f" | head -1 | awk '{print $2}')
      CLAIMED_MM=$(echo "$CLAIMED" | cut -d. -f1,2)
      if [[ "$CLAIMED_MM" != "$GOMOD_MAJOR_MINOR" ]]; then
        fail "$f: Go $CLAIMED (expected $GOMOD_MAJOR_MINOR)"
      else
        ok "$f: Go $CLAIMED"
      fi
    else
      ok "$f: no explicit Go version claim"
    fi
  fi
done

# CLAUDE.md and GEMINI.md are thin adapters — check they reference go.mod version if they claim one
for f in CLAUDE.md GEMINI.md; do
  if [[ -f "$f" ]]; then
    if grep -qE 'Go [0-9]+\.[0-9]+' "$f"; then
      CLAIMED=$(grep -oE 'Go [0-9]+\.[0-9]+(\.[0-9]+)?' "$f" | head -1 | awk '{print $2}')
      CLAIMED_MM=$(echo "$CLAIMED" | cut -d. -f1,2)
      if [[ "$CLAIMED_MM" != "$GOMOD_MAJOR_MINOR" ]]; then
        fail "$f: Go $CLAIMED (expected $GOMOD_MAJOR_MINOR)"
      else
        ok "$f: Go $CLAIMED"
      fi
    else
      ok "$f: no Go version claim"
    fi
  fi
done

# ── Node version checks ───────────────────────────────────────────────────────
for f in README.md CONTRIBUTING.md; do
  if [[ -f "$f" ]]; then
    if grep -qiE 'node [0-9]+' "$f"; then
      CLAIMED=$(grep -oiE 'node [0-9]+' "$f" | head -1 | awk '{print $2}')
      if [[ "$CLAIMED" != "$NVMRC_VERSION" ]]; then
        fail "$f: Node $CLAIMED (expected $NVMRC_VERSION per .nvmrc)"
      else
        ok "$f: Node $CLAIMED"
      fi
    else
      ok "$f: no explicit Node version claim"
    fi
  fi
done

# ── CI workflow filename references ───────────────────────────────────────────
DELETED_WORKFLOWS="ci-cd.yml"
for f in README.md CONTRIBUTING.md CLAUDE.md GEMINI.md AGENTS.md; do
  if [[ -f "$f" ]]; then
    for dead in $DELETED_WORKFLOWS; do
      if grep -q "$dead" "$f"; then
        fail "$f: references deleted workflow $dead"
      else
        ok "$f: no reference to $dead"
      fi
    done
  fi
done

# ── Result ────────────────────────────────────────────────────────────────────
echo ""
if [[ "$FAILED" == "true" ]]; then
  echo "ERROR: Version drift detected. Update the files listed above."
  exit 1
else
  echo "All version claims are consistent."
fi
