#!/usr/bin/env bash
# =============================================================================
# verify-toolchain.sh — Declared-toolchain contract verifier (issue #179)
#
# Reads toolchain.toml and checks every declared tool is on PATH and meets its
# minimum version. This is the recurrence-prevention mechanism: a tool missing
# in CI is a tool that would be missing on the VPS at deploy time.
#
# Exit codes:
#   0 — contract satisfied
#   1 — required tool missing or below declared floor
#   2 — contract file itself is invalid (parse/typo)
#
# Usage:
#   bash scripts/verify-toolchain.sh [path/to/toolchain.toml]
#   scripts/verify-toolchain.sh                # uses ./toolchain.toml
# =============================================================================
set -euo pipefail

# Resolve repo root from script location so the command works from any cwd.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONTRACT="${1:-${REPO_ROOT}/toolchain.toml}"

if [ ! -f "${CONTRACT}" ]; then
  echo "ERROR: toolchain contract not found at ${CONTRACT}" >&2
  exit 2
fi

# Prefer a prebuilt binary if present (faster in CI), else `go run` the package.
# Using the same module's Go toolchain guarantees the version-parsing logic
# under test is exactly what runs here.
cd "${REPO_ROOT}"

if [ -x "${REPO_ROOT}/bin/verifytoolchain" ]; then
  exec "${REPO_ROOT}/bin/verifytoolchain" -contract "${CONTRACT}"
fi

# `go run` is the zero-install path (dev machines + first CI run). The CLI
# lives under cmd/; the library (with the unit-tested version logic) is the
# parent package.
exec go run "${REPO_ROOT}/scripts/verifytoolchain/cmd/verifytoolchain" -contract "${CONTRACT}"
