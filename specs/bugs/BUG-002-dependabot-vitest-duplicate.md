---
bug_id: BUG-002
status: closed
severity: medium
scope: ci
title: Dependabot vitest update in /packages/auth fails — PR already exists
fixed_at: '2026-07-11'
fix: 'Merged PR #81 — updated vitest from ^2.0 to ^4.1 in /packages/auth'
---

## Symptom

Dependabot security update run [#29137331160] for `vitest` in `/packages/auth` fails with:
```
Error: pull_request_exists_for_latest_version
Dependency: vitest, Latest: 4.1.10
```

## Root Cause

Dependabot triggered a security update for vitest in `/packages/auth` but PR #81 (`dependabot/npm_and_yarn/packages/auth/multi-36eeedcbf7`) already exists bumping vitest from `^2.0` to `^4.1.10`. Dependabot errors instead of silently skipping.

## Fix

Merge PR #81 (already open, already targets the latest vitest 4.1.10) after verifying tests pass.

## Verify

```bash
gh pr checkout 81
npm ci --prefix packages/auth
npx vitest run --project auth 2>&1 | tail -5
# all green → merge
gh pr merge 81 --squash
```
