#!/usr/bin/env bash
set -euo pipefail

mode=${1:-}
if [[ -z "$mode" ]]; then
  echo "usage: $0 security-review|audit-code" >&2
  exit 2
fi
case "$mode" in
  security-review|audit-code) ;;
  *) echo "unsupported review: $mode" >&2; exit 2 ;;
esac

root=$(git rev-parse --show-toplevel)
name="review-${mode}"
path="$root/.bigpowers/worktrees/$name"
mkdir -p "$(dirname "$path")"

if git worktree list --porcelain | grep -Fq "worktree $path"; then
  printf '%s\n' "$path"
  exit 0
fi

if [[ -e "$path" ]]; then
  echo "review worktree path exists but is not registered: $path" >&2
  exit 1
fi

git worktree add --detach "$path" HEAD >/dev/null
printf '%s\n' "$path"
