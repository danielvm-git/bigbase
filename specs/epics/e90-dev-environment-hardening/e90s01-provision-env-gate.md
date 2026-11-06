# e90s01 — Provision BIGBASE_ROOT_ENCRYPTION_KEY and Add Required-Env Gate to CI

**type:** chore
**risk:** P0
**context:** infra
**BCPs:** 3

## 1. Type
Operational fix, documentation, and CI gate addition.

## 2. Context
The native secret manager requires `BIGBASE_ROOT_ENCRYPTION_KEY` at production startup. A structural gate in CI prevents shipping required environment key additions without documentation in `.env.example`.

## 3. Problem
Previously, a new required environment variable could be added to deploy code paths without documentation or validation before binary swap, causing deploy rollbacks.

## 4. Solution
Document `BIGBASE_ROOT_ENCRYPTION_KEY` in `.env.example`, assert `REQUIRED_ENV_KEYS` in deploy workflow, and enforce `scripts/check-required-env-documented.sh` in pre-commit and CI.
