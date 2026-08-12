#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
python3 - <<'PY'
from datetime import datetime, timezone
from pathlib import Path
import json

root = Path.cwd()
items = []
for spec in sorted((root / "specs" / "epics").glob("*/e89s*.md")):
    stem = spec.stem
    story = stem.split("-", 1)[0]
    tagged = []
    for base in (root / "components", root / "kernel", root / "internal", root / "ui", root / "tests"):
        if not base.exists():
            continue
        for path in base.rglob("*"):
            if path.is_file() and path.suffix in {".go", ".ts", ".tsx", ".js", ".css"}:
                try:
                    if f"story: {story}" in path.read_text(errors="replace") and path.stat().st_mtime > spec.stat().st_mtime:
                        tagged.append(path.relative_to(root).as_posix())
                except OSError:
                    pass
    if tagged:
        items.append({"story_id": story, "spec": spec.relative_to(root).as_posix(), "newer_tagged_files": sorted(tagged)})
report = {"generated_at": datetime.now(timezone.utc).isoformat(), "suspect_links": items, "count": len(items)}
(root / "specs" / "drift-report.json").write_text(json.dumps(report, indent=2) + "\n")
print(json.dumps(report))
raise SystemExit(1 if items else 0)
PY
