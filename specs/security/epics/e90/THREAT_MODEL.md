# Threat Model: e90 Dev-Environment Hardening

**type:** threat-model
**epic:** e90
**risk:** LOW
**context:** tooling, CI/CD, and developer environment contracts

## 1. Scope & Surface Area

Epic e90 hardens internal build automation, CI checks, documentation consistency, and developer tooling:
- Deployment environment secrets validation (`BIGBASE_ROOT_ENCRYPTION_KEY`)
- Security scanner configuration (`gosec`, `.gosec.json`, conflict marker guards)
- Race detection (`-race` flag in CI for deploy, functions, realtime)
- Frontend automated type checking and linting in CI
- MCP client config generation and sanitization (no leaking of hardcoded paths or credentials)
- Doc drift and version consistency verification

## 2. Threat Analysis (STRIDE)

| Category | Threat Scenario | Impact | Mitigation |
|---|---|---|---|
| **Information Disclosure** | Hardcoded secrets or tokens in MCP configs / environment files | HIGH | Canonical `.mcp.json` with env var substitution; Gitleaks scan in CI and pre-commit hook. |
| **Tampering** | Uncaught git conflict markers (`<<<<<<<`) in configuration or source | MEDIUM | Conflict marker scanning script and CI verify step. |
| **Denial of Service** | Race conditions causing deadlocks or crashes in concurrent Go components | HIGH | `-race` test suites running in CI on deploy, functions, and realtime packages. |
| **Elevation of Privilege** | Supply-chain tampering of unpinned GitHub Actions | HIGH | Pinned action SHAs across all GitHub workflows. |

## 3. Security Verdict

LOW intrinsic risk. All changes strengthen defensive posture and CI verification gates.
Proceed to implementation without an architectural exception.
