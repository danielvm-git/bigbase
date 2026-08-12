#!/usr/bin/env bash
set -euo pipefail

EPIC_DIR=${1:-}
if [[ -z "$EPIC_DIR" ]]; then
  echo "usage: $0 specs/epics/eNN-slug" >&2
  exit 2
fi

python3 - "$EPIC_DIR" <<'PY'
from pathlib import Path
import re
import sys

root = Path(sys.argv[1])
manifest = root / "epic.yaml"
errors = []
if not manifest.is_file():
    errors.append(f"missing {manifest}")
    print("\n".join(errors), file=sys.stderr)
    raise SystemExit(1)

manifest_text = manifest.read_text()
story_rows = re.findall(
    r"- id: (?P<id>e\d+s\d+)\n(?P<body>.*?)(?=\n  - id: e\d+s\d+|\Z)",
    manifest_text,
    re.S,
)
if not story_rows:
    errors.append("epic.yaml has no stories")

seen = set()
for story_id, body in story_rows:
    if story_id in seen:
        errors.append(f"duplicate story {story_id}")
    seen.add(story_id)
    spec_match = re.search(r"spec:\s*(\S+)", body)
    tasks_match = re.search(r"tasks:\s*(\S+)", body)
    delta_match = re.search(r"delta:\s*(ADDED|MODIFIED|REMOVED|RENAMED)", body)
    if not spec_match or not tasks_match:
        errors.append(f"{story_id}: missing spec/tasks manifest entries")
        continue
    spec = root / spec_match.group(1)
    tasks = root / tasks_match.group(1)
    if not spec.is_file():
        errors.append(f"{story_id}: missing {spec}")
        continue
    if not tasks.is_file():
        errors.append(f"{story_id}: missing {tasks}")
        continue
    spec_text = spec.read_text()
    task_text = tasks.read_text()
    for section in ("## Requirements", "## 17. Acceptance Criteria", "## 18. Implementation Steps", "## 19. Verification Script", "## 20. Out of scope"):
        if section not in spec_text:
            errors.append(f"{story_id}: missing {section}")
    if delta_match:
        delta = delta_match.group(1)
        if f"#### {delta}" not in spec_text:
            errors.append(f"{story_id}: manifest delta {delta} missing from spec requirements")
        if delta in {"MODIFIED", "REMOVED", "RENAMED"}:
            if "**Before:**" not in spec_text or "**After:**" not in spec_text:
                errors.append(f"{story_id}: {delta} requires Before/After requirements")
    task_ids = re.findall(r"^\s*- id:\s*(\S+)", task_text, re.M)
    if not task_ids:
        errors.append(f"{story_id}: no tasks")
    for task_id in task_ids:
        if f"{task_id}" not in task_text:
            errors.append(f"{story_id}: malformed task {task_id}")
    statuses = re.findall(r"^\s*status:\s*(\S+)", task_text, re.M)
    if any(status != "failing" for status in statuses):
        errors.append(f"{story_id}: task status must start as failing")
    if len(re.findall(r"^\s*verify:\s*\S+", task_text, re.M)) != len(task_ids):
        errors.append(f"{story_id}: every task needs a verify command")
    if len(re.findall(r"^\s*risk:\s*P[0-3]", task_text, re.M)) != len(task_ids):
        errors.append(f"{story_id}: every task needs a risk")
    if len(re.findall(r"^\s*security:\s*(none|low|medium|high)", task_text, re.M)) != len(task_ids):
        errors.append(f"{story_id}: every task needs security metadata")
    if "security: high" in task_text and "no new security findings in affected paths" not in task_text:
        errors.append(f"{story_id}: high-security tasks need security gate text")

if errors:
    print("\n".join(errors), file=sys.stderr)
    raise SystemExit(1)
print(f"plan-consistency-check: OK ({len(story_rows)} stories)")
PY
