#!/usr/bin/env bash
# check-required-env-documented.sh
#
# Fail if a required deployment env key — declared as REQUIRED_ENV_KEYS in
# .github/workflows/deploy.yml — is not documented in .env.example.
#
# Why: the deploy-time config-drift class (e89: BIGBASE_ROOT_ENCRYPTION_KEY
# shipped without an .env.example entry, taking the release down at boot).
# deploy.yml already fails the deploy when a required key is missing on the
# VPS; this gate fails earlier, at commit/CI time, when the key is added but
# nobody documented it for local/ops users.
#
# Usage: bash scripts/check-required-env-documented.sh
# Exit 0 = all required keys documented (or files missing), 1 = drift found.
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
cd "$root"

DEPLOY_YML=".github/workflows/deploy.yml"
ENV_EXAMPLE=".env.example"

if [ ! -f "$DEPLOY_YML" ] || [ ! -f "$ENV_EXAMPLE" ]; then
  echo "SKIP: $DEPLOY_YML or $ENV_EXAMPLE not present"
  exit 0
fi

# Extract REQUIRED_ENV_KEYS=(A B C) — the deploy-side config contract.
mapfile -t required < <(
  sed -n 's/^[[:space:]]*REQUIRED_ENV_KEYS=(\(.*\))[[:space:]]*$/\1/p' "$DEPLOY_YML" \
    | tr ' ' '\n' \
    | grep -E '^[A-Z][A-Z0-9_]+$'
)

if [ "${#required[@]}" -eq 0 ]; then
  echo "SKIP: no REQUIRED_ENV_KEYS found in $DEPLOY_YML"
  exit 0
fi

missing=()
for key in "${required[@]}"; do
  # Documented = a KEY= line (possibly a commented placeholder) or a mention
  # in a comment. The .env.example convention is `KEY=` with an explanation.
  if ! grep -qE "^[[:space:]]*#?[[:space:]]*${key}=" "$ENV_EXAMPLE" \
     && ! grep -qE "^[[:space:]]*#[^#]*\b${key}\b" "$ENV_EXAMPLE"; then
    missing+=("$key")
  fi
done

if [ "${#missing[@]}" -gt 0 ]; then
  echo "❌ Required env keys not documented in $ENV_EXAMPLE:"
  printf '   - %s\n' "${missing[@]}"
  echo "   Add a KEY= entry for each (see BIGBASE_ROOT_ENCRYPTION_KEY for the format)."
  exit 1
fi

echo "✅ All ${#required[@]} required env keys documented in $ENV_EXAMPLE"
