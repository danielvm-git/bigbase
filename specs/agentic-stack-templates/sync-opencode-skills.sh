#!/usr/bin/env bash
# Sync bigpowers SKILL.md sources into OpenCode-native .opencode/skills/<name>/SKILL.md
# Do NOT use bigpowers/scripts/sync-skills.sh (that targets Cursor .mdc + Gemini only).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BIGPOWERS="${BIGPOWERS_DIR:-$ROOT/node_modules/bigpowers}"
OUT="$ROOT/.opencode/skills"

if [[ ! -d "$BIGPOWERS" ]]; then
  echo "bigpowers not found at $BIGPOWERS — run: npm install -D bigpowers" >&2
  exit 1
fi

mkdir -p "$OUT"
count=0

for skill_md in "$BIGPOWERS"/*/SKILL.md; do
  [[ -f "$skill_md" ]] || continue
  skill_dir="$(dirname "$skill_md")"
  name="$(basename "$skill_dir")"

  # Skip non-skill dirs
  [[ "$name" == "scripts" || "$name" == "specs" || "$name" == ".gemini" ]] && continue
  [[ "$name" == node_modules ]] && continue

  description="$(awk '/^---/{f++} f==1 && /^description:/{print; exit}' "$skill_md" \
    | sed 's/^description:[[:space:]]*//' \
    | sed 's/^"\(.*\)"$/\1/')"

  body="$(awk '/^---/{f++; next} f>=2{print}' "$skill_md")"

  mkdir -p "$OUT/$name"
  {
    echo "---"
    echo "name: $name"
    if [[ -n "$description" ]]; then
      echo "description: \"$description\""
    fi
    echo "compatibility: opencode"
    echo "---"
    echo ""
    echo "$body"
  } > "$OUT/$name/SKILL.md"

  count=$((count + 1))
done

echo "sync-opencode-skills: $count skills → $OUT/"
