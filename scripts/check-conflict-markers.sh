#!/usr/bin/env bash
# check-conflict-markers.sh — fail if any tracked file contains an unresolved
# git merge-conflict marker. Guards against the .gitignore:44 regression class
# (e90s02): a dangling `<<<<<<< HEAD` from a bad merge that lints clean but
# corrupts the file's meaning.
#
# Matches the unambiguous open/close markers (`<<<<<<<` / `>>>>>>>`, 7 chars at
# line start) across all tracked files. The `=======` separator is intentionally
# NOT matched on its own — 7 equals signs legitimately appear in markdown rules
# and comment dividers, and either an open or close marker is always present in
# a real conflict.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# -I skips binary files; anchored to line start; 7 markers then space-or-EOL.
if git grep -nIE '^(<{7}|>{7})( |$)' -- ':!*.png' ':!*.jpg' ':!*.gif' >/tmp/conflict-markers.txt 2>/dev/null; then
  echo "ERROR: unresolved git conflict markers found:" >&2
  cat /tmp/conflict-markers.txt >&2
  rm -f /tmp/conflict-markers.txt
  exit 1
fi

rm -f /tmp/conflict-markers.txt
echo "conflict-markers: OK (none found)"
