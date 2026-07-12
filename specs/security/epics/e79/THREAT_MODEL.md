# Threat Model — e79: CI/CD Template Consolidation

**Date:** 2026-07-12
**Scope:** `.github/workflows/ci-cd.yml` (new), `.github/workflows/codeql.yml` (updated), `.github/workflows/ci.yml` (removed)
**Risk Level:** Low

## Surface Area

| Component | Change | Risk |
|-----------|--------|------|
| `ci-cd.yml` | NEW — Go CI template with lint/verify | Low |
| `codeql.yml` | MODIFIED — updated template format | Low |
| `ci.yml` | REMOVED — superseded by ci-cd.yml | Low |
| `release-deploy.yml` | UNCHANGED | None |
| `pr-review.yaml` | UNCHANGED | None |

## Vulnerability Assessment

### 1. Supply Chain — golangci-lint install (ci-cd.yml)
- **Category:** Supply chain (CWE-1104)
- **Risk:** `curl | sh` from golangci-lint's install script
- **Mitigation:** This is the official golangci-lint install method, same as current practice. Pinning golangci-lint version is recommended but not in scope.
- **Severity:** Informational

### 2. Secrets Exposure — DEPLOY_TOKEN env var
- **Category:** Credential exposure (CWE-532)
- **Risk:** `BIGBASE_DEPLOY_TOKEN` is exported to `$GITHUB_ENV`
- **Mitigation:** Already masked by GitHub Actions. The ci-cd.yml doesn't use this for deploy (SSH deploy uses separate workflow). Token is exported but never consumed in the new CI pipeline.
- **Severity:** None (token unused in this workflow)

### 3. Permission Escalation
- **Category:** Authorization (CWE-250)
- **Risk:** Template sets `contents: read` at top level, but Allure report needs `contents: write`
- **Mitigation:** Allure job already has `contents: write` in current ci.yml. Preserve job-level permission override.
- **Severity:** None (handled by existing pattern)

### 4. Concurrency Group Collision
- **Category:** Availability
- **Risk:** New concurrency group could cancel in-progress runs incorrectly
- **Mitigation:** Template uses `cancel-in-progress: true` which is standard practice. Release-deploy uses `false`.
- **Severity:** Low

## Verdict

**SAFE TO PROCEED.** No application code changes. All workflows are YAML-only config changes. The removed `ci.yml` is fully superseded by the new `ci-cd.yml` with preserved PostgreSQL matrix + Allure report functionality.
