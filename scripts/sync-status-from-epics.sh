#!/usr/bin/env bash
# Regenerate specs/execution-status.yaml from epic shards.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

python3 <<'PY'
from pathlib import Path
import yaml

root = Path("specs")
rp = yaml.safe_load((root / "release-plan.yaml").read_text()) or {}
epics_index = rp.get("epics") or []

epic_status: dict[str, str] = {}
story_status: dict[str, str] = {}

def absorb_shard(path: Path) -> None:
    if not path.is_file():
        return
    data = yaml.safe_load(path.read_text()) or {}
    eid = data.get("id")
    if eid:
        epic_status[eid] = data.get("status", "pending")
    for story in data.get("stories") or []:
        sid = story.get("id")
        if sid:
            story_status[sid] = story.get("status", "pending")

    story_dir = path.parent / "stories"
    if story_dir.is_dir():
        for smd in sorted(story_dir.glob("*.md")):
            text = smd.read_text()
            if not text.startswith("---"):
                continue
            end = text.find("---", 3)
            if end < 0:
                continue
            fm = yaml.safe_load(text[3:end]) or {}
            sid = fm.get("id")
            if sid:
                story_status[sid] = fm.get("status", "pending")

for ep in epics_index:
    fpath = ep.get("file")
    if not fpath:
        continue
    absorb_shard(root / fpath)

out = {
    "generated_by": "scripts/sync-status-from-epics.sh",
    "epics": epic_status,
    "stories": story_status,
}

dest = root / "execution-status.yaml"
dest.write_text(yaml.dump(out, default_flow_style=False, sort_keys=False))
print(f"sync-status-from-epics: wrote {dest} ({len(epic_status)} epics, {len(story_status)} stories)")
PY
