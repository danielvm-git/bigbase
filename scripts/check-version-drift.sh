#!/usr/bin/env bash
# story: e90s07
# check-version-drift.sh — Assert that Go and Node version claims in docs
# match the actual go.mod toolchain and .nvmrc declarations.
#
# Fails with a non-zero exit and a list of mismatches. Used in CI verify job.
#
# Run from the repository root, or point BIGBASE_VERSION_ROOT at one. Outside a
# git checkout, this exits non-zero on purpose: a drift gate that silently
# passes when it cannot see the files is worse than a loud failure.

set -euo pipefail

# Set BIGBASE_VERSION_ROOT to check a directory other than this repository
# (used by scripts/check-version-drift.test.cjs fixtures).
cd "${BIGBASE_VERSION_ROOT:-$(git rev-parse --show-toplevel)}"

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

# ── CI version pins ───────────────────────────────────────────────────────────
# A CI pin that drifts from go.mod/.nvmrc builds with a different toolchain
# than the one the repository declares. Major.minor is enough for Go (go.mod
# carries the patch); Node pins the major only.
CI_WORKFLOW=".github/workflows/test-build-release.yml"
if [[ -f "$CI_WORKFLOW" ]]; then
  ci_pin() {
    sed -n "s/^[[:space:]]*$1:[[:space:]]*['\"]\{0,1\}\([0-9][0-9.]*\)['\"]\{0,1\}[[:space:]]*$/\1/p" \
      "$CI_WORKFLOW" | head -1
  }

  CI_GO=$(ci_pin GO_VERSION)
  if [[ -n "$CI_GO" ]]; then
    if [[ "$CI_GO" != "$GOMOD_MAJOR_MINOR" ]]; then
      fail "$CI_WORKFLOW: GO_VERSION $CI_GO (expected $GOMOD_MAJOR_MINOR per go.mod)"
    else
      ok "$CI_WORKFLOW: GO_VERSION $CI_GO"
    fi
  fi

  CI_NODE=$(ci_pin NODE_VERSION)
  if [[ -n "$CI_NODE" ]]; then
    if [[ "$CI_NODE" != "$(echo "$NVMRC_VERSION" | cut -d. -f1)" ]]; then
      fail "$CI_WORKFLOW: NODE_VERSION $CI_NODE (expected $NVMRC_VERSION per .nvmrc)"
    else
      ok "$CI_WORKFLOW: NODE_VERSION $CI_NODE"
    fi
  fi
fi

# ── Deploy-host toolchain contract ───────────────────────────────────────────
# toolchain.toml is the single source of truth for binaries the deploy host
# needs; its Go floor must agree with go.mod too.
TOOLCHAIN="toolchain.toml"
if [[ -f "$TOOLCHAIN" ]]; then
  TOOLCHAIN_GO=$(awk '/^\[tools\.required\.go\]/{found=1; next} found && /^min[[:space:]]*=/{gsub(/[" ]/, "", $3); print $3; exit}' "$TOOLCHAIN")
  if [[ -n "$TOOLCHAIN_GO" ]]; then
    if [[ "$TOOLCHAIN_GO" != "$GOMOD_MAJOR_MINOR" ]]; then
      fail "$TOOLCHAIN: go min $TOOLCHAIN_GO (expected $GOMOD_MAJOR_MINOR per go.mod)"
    else
      ok "$TOOLCHAIN: go min $TOOLCHAIN_GO"
    fi
  fi
fi

# ── README version badge must be dynamic ──────────────────────────────────────
# The release bot bumps VERSION/CHANGELOG, not README. A literal
# `badge/version-X.Y.Z` silently drifts (it read 2.89.0 while VERSION was
# 2.92.2). Require the dynamic GitHub release badge instead.
if [[ -f README.md ]] && grep -qE 'badge/version-[0-9]+\.[0-9]+\.[0-9]+' README.md; then
  fail "README.md hardcodes a version badge (use shields github/v/release)"
else
  ok "README.md version badge is dynamic"
fi

# ── Result ──────────────────────────────
echo ""
if [[ "$FAILED" == "true" ]]; then
  echo "ERROR: Version drift detected. Update the files listed above."
  exit 1
else
  echo "All version claims are consistent."
fi
