#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" != "--okf" ]]; then
  echo "usage: $0 --okf" >&2
  exit 2
fi

root=$(git rev-parse --show-toplevel)
source_root="$root/skills"
if [[ ! -d "$source_root" ]]; then
  source_root="${BIGPOWERS_SKILLS_ROOT:-$HOME/.claude/skills}"
fi
if [[ ! -d "$source_root" ]]; then
  echo "skills source not found: $source_root" >&2
  exit 1
fi

python3 - "$root" "$source_root" <<'PY'
from pathlib import Path
import re
import sys

root = Path(sys.argv[1])
source_root = Path(sys.argv[2])
out = root / "specs" / "skills-wiki" / "skills"
out.mkdir(parents=True, exist_ok=True)
index = ["# Skills Wiki", "", f"Generated from `{source_root}`.", ""]
count = 0
for skill in sorted(p for p in source_root.iterdir() if p.is_dir()):
    source = skill / "SKILL.md"
    if not source.is_file():
        continue
    text = source.read_text(errors="replace")
    front = re.match(r"---\n(.*?)\n---", text, re.S)
    name = skill.name
    description = ""
    if front:
        m = re.search(r"^description:\s*(.+)$", front.group(1), re.M)
        if m:
            description = m.group(1).strip().strip('"')
    page = out / f"{name}.md"
    page.write_text(f"# {name}\n\nsource: {source}\nreferences: [{source}]\nenforced_by: [survey-context, plan-work, verify-work]\n\n{description}\n")
    index.append(f"- [{name}]({page.name})")
    count += 1
(root / "specs" / "skills-wiki" / "index.md").write_text("\n".join(index) + "\n")
print(f"generated {count} skill pages")
PY
