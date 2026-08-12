#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" != "--self-test" ]]; then
  echo "usage: $0 --self-test" >&2
  exit 2
fi

root=$(git rev-parse --show-toplevel)
cd "$root"
python3 - <<'PY'
from pathlib import Path
import json

for name in ("specs/traceability-matrix.json", "specs/blind-spots.json"):
    path = Path(name)
    if not path.is_file():
        continue
    data = json.loads(path.read_text())
    if not isinstance(data, dict):
        raise SystemExit(f"{name}: expected object")
print("gate-trace self-test: OK")
PY
