#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"
python3 - <<'PY'
from pathlib import Path
import re

root = Path.cwd()
blockers = []
for story in sorted((root / "specs" / "epics" / "e89-native-secret-manager").glob("e89s*.md")):
    text = story.read_text(errors="replace")
    if "TODO: implement" in text or "TBD" in text or "placeholder" in text.lower():
        blockers.append(f"{story}: placeholder text")
    if "## 18. Implementation Steps" not in text:
        blockers.append(f"{story}: missing implementation steps")
    if not re.search(r"\d+\. .*→ verify: `[^`]+`", text):
        blockers.append(f"{story}: missing runnable verify step")
for ledger in sorted((root / "specs" / "epics" / "e89-native-secret-manager").glob("e89s*-tasks.yaml")):
    text = ledger.read_text(errors="replace")
    tasks = len(re.findall(r"^\s+- id:\s*\S+", text, re.M))
    verifies = len(re.findall(r"^\s+verify:\s*\S+", text, re.M))
    if tasks != verifies:
        blockers.append(f"{ledger}: task/verify count mismatch")
if blockers:
    print("BLOCKER")
    print("\n".join(f"- {item}" for item in blockers))
    raise SystemExit(1)
print("FILLED: e89 story specs and task ledgers have countable verify coverage")
PY
