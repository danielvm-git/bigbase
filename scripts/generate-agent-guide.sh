#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
python3 - "$root" <<'PY'
from pathlib import Path
import re
import sys

root = Path(sys.argv[1])
source = root / "AGENTS.md"
out = root / "specs" / "agent-guide"
out.mkdir(parents=True, exist_ok=True)
text = source.read_text(errors="replace")
headings = list(re.finditer(r"^(#{1,4})\s+(.+?)\s*$", text, re.M))
index = ["# Agent Guide", "", "Generated from `AGENTS.md`.", "", "source: AGENTS.md", "references: [AGENTS.md]", ""]
for idx, match in enumerate(headings):
    title = match.group(2).strip()
    slug = re.sub(r"[^a-z0-9]+", "-", title.lower()).strip("-") or f"section-{idx+1}"
    end = headings[idx + 1].start() if idx + 1 < len(headings) else len(text)
    body = text[match.start():end].strip()
    page = out / f"{slug}.md"
    page.write_text(f"# {title}\n\nsource: AGENTS.md\nreferences: [AGENTS.md]\n\n{body}\n")
    index.append(f"- [{title}]({page.name})")
(out / "index.md").write_text("\n".join(index) + "\n")
print(f"generated {len(headings)} agent-guide pages")
PY
