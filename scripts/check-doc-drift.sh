#!/usr/bin/env bash
# check-doc-drift.sh — fail if declared tool versions disagree (e90s06).
#
# Canonical sources of truth:
#   Go   -> go.mod `go` directive (major.minor)
#   Node -> .nvmrc
# Everything else (CI pins, prose version claims in docs) must agree with them.
# This is the recurrence guard for the "README says Go 1.22 while go.mod is 1.26"
# / ".nvmrc 22 vs CI Node 24" drift class.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

WF=".github/workflows/test-build-release.yml"
DOCS="README.md CONTRIBUTING.md CLAUDE.md GEMINI.md AGENTS.md"
fail=0

GO_CANON=$(awk '/^go /{print $2; exit}' go.mod | cut -d. -f1,2)
NODE_CANON=$(tr -d '[:space:]' < .nvmrc)
[ -n "$GO_CANON" ]   || { echo "FATAL: cannot read go version from go.mod" >&2; exit 2; }
[ -n "$NODE_CANON" ] || { echo "FATAL: cannot read node version from .nvmrc" >&2; exit 2; }
echo "canonical: Go ${GO_CANON}, Node ${NODE_CANON}"

# --- CI env pins must equal canon ---
grep -qE "GO_VERSION: '?${GO_CANON}'?" "$WF" \
  || { echo "DRIFT: $WF GO_VERSION != ${GO_CANON}" >&2; fail=1; }
grep -qE "NODE_VERSION: '?${NODE_CANON}'?" "$WF" \
  || { echo "DRIFT: $WF NODE_VERSION != ${NODE_CANON}" >&2; fail=1; }

# --- Prose "Go 1.NN" claims in docs must equal canon ---
# (grep -o yields file:line:match; keep only mismatches.)
STALE_GO=$(grep -rnoE 'Go 1\.[0-9]+' $DOCS 2>/dev/null | grep -vE ":Go ${GO_CANON}$" || true)
if [ -n "$STALE_GO" ]; then
  echo "DRIFT: stale Go version claim(s) — canonical is ${GO_CANON}:" >&2
  echo "$STALE_GO" >&2
  fail=1
fi

if [ "$fail" -ne 0 ]; then
  echo "" >&2
  echo "Doc-drift detected. Reconcile the above against go.mod / .nvmrc." >&2
  exit 1
fi
echo "doc-drift: OK (versions consistent)"
