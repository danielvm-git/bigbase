#!/usr/bin/env bash
# Validate bigpowers YAML specs layout.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

python3 <<'PY'
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("FAIL: PyYAML required (pip install pyyaml)")
    sys.exit(1)

root = Path("specs")
errors: list[str] = []

required = [
    root / "state.yaml",
    root / "release-plan.yaml",
    root / "execution-status.yaml",
    root / "planning-status.yaml",
    root / "dependencies.yaml",
    root / "requirements" / "SCOPE_LATEST.yaml",
    root / "plans" / "TECH_STACK_LATEST.md",
]

META_YAML = [
    root / "state.yaml",
    root / "release-plan.yaml",
    root / "execution-status.yaml",
    root / "planning-status.yaml",
    root / "dependencies.yaml",
    root / "requirements" / "SCOPE_LATEST.yaml",
    root / "requirements" / "VISION_LATEST.yaml",
    root / "requirements" / "GLOSSARY_LATEST.yaml",
    root / "bugs" / "registry.yaml",
]

for p in required:
    if not p.is_file():
        errors.append(f"missing required file: {p}")

for p in META_YAML:
    if not p.is_file():
        continue
    doc = yaml.safe_load(p.read_text()) or {}
    if not doc.get("type"):
        errors.append(f"{p}: missing type:")
    if not doc.get("context"):
        errors.append(f"{p}: missing context:")

rp_path = root / "release-plan.yaml"
if rp_path.is_file():
    rp = yaml.safe_load(rp_path.read_text()) or {}
    epics = rp.get("epics") or []
    if not epics:
        errors.append("release-plan.yaml: no epics defined")
    for ep in epics:
        eid = ep.get("id")
        fpath = ep.get("file")
        if not eid or not fpath:
            errors.append(f"release-plan epic missing id or file: {ep}")
            continue
        shard = root / fpath
        if not shard.is_file():
            errors.append(f"epic shard not found: {shard} (id={eid})")

        data = yaml.safe_load(shard.read_text()) or {}
        if not data.get("type"):
            errors.append(f"{shard}: missing type:")
        if not data.get("context"):
            errors.append(f"{shard}: missing context:")
        stories = data.get("stories") or []
        for story in stories:
            tasks = story.get("tasks") or []
            if isinstance(tasks, str):
                continue
            for task in tasks:
                if not task.get("verify"):
                    errors.append(
                        f"{eid}/{story.get('id')}: task missing verify: {task.get('desc', '')[:60]}"
                    )

        story_dir = shard.parent / "stories"
        if story_dir.is_dir():
            for smd in sorted(story_dir.glob("*.md")):
                text = smd.read_text()
                if not text.startswith("---"):
                    errors.append(f"{smd}: missing YAML frontmatter")
                    continue
                end = text.find("---", 3)
                if end < 0:
                    errors.append(f"{smd}: unclosed frontmatter")
                    continue
                fm = yaml.safe_load(text[3:end]) or {}
                sid = fm.get("id", smd.stem)
                for task in fm.get("tasks") or []:
                    if not task.get("verify"):
                        errors.append(f"{sid}: task missing verify")

ex_path = root / "execution-status.yaml"
if ex_path.is_file():
    ex = yaml.safe_load(ex_path.read_text()) or {}
    if "epics" not in ex and "stories" not in ex:
        errors.append("execution-status.yaml: expected epics or stories key")

if errors:
    print("validate-specs-yaml: FAIL")
    for e in errors:
        print(f"  - {e}")
    sys.exit(1)

print("validate-specs-yaml: OK")
PY
