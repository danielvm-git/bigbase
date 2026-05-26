#!/usr/bin/env bash
# Safely merge a feature branch into main.
# Usage: scripts/land-branch.sh <branch-name>
set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <branch-name>"
    exit 1
fi

BRANCH="$1"
CURRENT=$(git rev-parse --abbrev-ref HEAD)

if [ "$CURRENT" != "main" ]; then
    echo "❌ Must be on main branch. Currently on: $CURRENT"
    exit 1
fi

echo "🔀 Merging $BRANCH into main..."
git merge --no-ff "$BRANCH" -m "chore: merge $BRANCH into main"
echo "✅ Merged $BRANCH. Run 'git push origin main' to deploy."
