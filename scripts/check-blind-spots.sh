#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
python3 - <<'PY'
from datetime import datetime, timezone
from pathlib import Path
import json
import re

root = Path.cwd()
findings = []
for ledger in sorted((root / "specs" / "epics").rglob("e??s??-tasks.yaml")):
    text = ledger.read_text(errors="replace")
    story = re.search(r"^story_id:\s*(\S+)", text, re.M)
    story_id = story.group(1) if story else ledger.stem
    task_count = len(re.findall(r"^\s+- id:\s*\S+", text, re.M))
    verify_count = len(re.findall(r"^\s+verify:\s*\S+", text, re.M))
    passing_count = len(re.findall(r"^\s+status:\s+passing\s*$", text, re.M))
    if task_count != verify_count:
        findings.append({"severity": "HIGH", "code": "verify-gap", "story_id": story_id, "detail": f"{task_count} tasks but {verify_count} verify commands"})
    if passing_count != task_count and story_id.startswith("e89"):
        findings.append({"severity": "HIGH", "code": "task-not-passing", "story_id": story_id, "detail": f"{task_count - passing_count} tasks are not passing"})

output = {"generated_at": datetime.now(timezone.utc).isoformat(), "findings": findings, "summary": {"high": sum(f["severity"] == "HIGH" for f in findings), "medium": sum(f["severity"] == "MEDIUM" for f in findings), "low": sum(f["severity"] == "LOW" for f in findings)}}
(root / "specs" / "blind-spots.json").write_text(json.dumps(output, indent=2) + "\n")
print(json.dumps(output))
raise SystemExit(1 if output["summary"]["high"] else 0)
PY
