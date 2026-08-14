#!/usr/bin/env bash
# setup-dev.sh — one-command developer onboarding.
# Thin alias for scripts/setup.sh (the single source of setup logic) so both
# entrypoints resolve; kept minimal to avoid the duplication this epic targets.
set -euo pipefail
exec "$(cd "$(dirname "$0")" && pwd)/setup.sh" "$@"
