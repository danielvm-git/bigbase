#!/usr/bin/env bash
set -euo pipefail

# Check if codeql CLI is available
if ! command -v codeql >/dev/null 2>&1; then
    echo "codeql: SKIP (not installed)"
    exit 0
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

DB_DIR="${REPO_ROOT}/.codeql-db"
SARIF_OUT="${REPO_ROOT}/codeql-results.sarif"

cleanup() {
    rm -rf "$DB_DIR" "$SARIF_OUT"
}
trap cleanup EXIT

echo "codeql: creating database for Go..."
codeql database create "$DB_DIR" \
    --language=go \
    --command="go build ./..." \
    --overwrite \
    --quiet

echo "codeql: running go-code-scanning analysis..."
codeql database analyze "$DB_DIR" \
    codeql/go-queries:codeql-suites/go-code-scanning.qls \
    --format=sarif-latest \
    --output="$SARIF_OUT" \
    --quiet

python3 - <<'EOF'
import json, sys

try:
    with open("codeql-results.sarif") as f:
        sarif = json.load(f)
except Exception as e:
    print(f"❌ Failed to parse SARIF output: {e}", file=sys.stderr)
    sys.exit(1)

# Known audited DB passthrough wrapper alerts (specs/bugs/BUG-2026-07-11T032535-codeql-sql-injection.md)
known_sinks = {
    ("components/db/db.go", "go/sql-injection"),
    ("components/db/postgres.go", "go/sql-injection"),
}

unexpected = []
known_count = 0

for run in sarif.get("runs", []):
    for res in run.get("results", []):
        rule_id = res.get("ruleId")
        locs = res.get("locations", [])
        for loc in locs:
            phys = loc.get("physicalLocation", {})
            path = phys.get("artifactLocation", {}).get("uri", "")
            region = phys.get("region", {})
            line = region.get("startLine", 0)
            msg = res.get("message", {}).get("text", "")
            
            if (path, rule_id) in known_sinks:
                known_count += 1
            else:
                unexpected.append((rule_id, path, line, msg))

if unexpected:
    print(f"❌ CodeQL detected {len(unexpected)} new vulnerability alert(s):", file=sys.stderr)
    for rule, path, line, msg in unexpected:
        print(f"   [{rule}] {path}:{line} - {msg}", file=sys.stderr)
    sys.exit(1)

print(f"codeql: OK ({known_count} known DB driver passthroughs, 0 regressions)")
EOF
