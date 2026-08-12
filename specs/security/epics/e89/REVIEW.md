# Security Review — e89s01 Implementation

**Date:** 2026-08-12
**Scope:** e89s01 isolated branch `feat/e89s01` at the current integration checkpoint.
**Threat Model:** `specs/security/epics/e89/THREAT_MODEL.md`
**Verdict:** CONCERNS — e89s01 targeted gates pass; release remains blocked by the existing race gate and deferred Site-key principal work in e89s07.

## Findings disposition

| Finding | Status | Evidence |
|---|---|---|
| F-001 CWE-311/CWE-312 optional encryption | Remediated in s01 | Canonical base64 root-key parsing; production composition rejects missing/invalid root key; plaintext requires `BIGBASE_ENV=development` plus explicit opt-in. |
| F-002 CWE-639 MCP Site target IDOR | Partially remediated | Organization-authenticated Site target authorization now runs before existing Site env/key tools. Site deploy-key principal support remains an e89s07 contract. |
| F-003 CWE-754/CWE-532 runtime fail-open | Remediated in s01 | Runtime uses EnvResolver; selected resolution failures mark deployment failed; runtime stdout/stderr are redacted. |
| F-004 CWE-209/CWE-532 MCP error disclosure | Remediated in s01 | Existing Site env handlers return generic errors while preserving safe not-found behavior; SQL/key/plaintext details are not returned. |

## Security controls reviewed

- `BIGBASE_ROOT_ENCRYPTION_KEY` is the production composition input and must decode to
  exactly 32 bytes from base64.
- Legacy hex parsing remains an explicit test/migration input, not a production fallback.
- Site mutation responses omit plaintext values and enforce 128-byte keys, 64 KiB values,
  and build/runtime flag validity.
- EnvResolver returns errors for scan/decryption failures instead of silently dropping
  selected variables.
- MCP Site target checks use parameterized ownership queries and generic denial results.
- Existing LLM `StripSecretLines` removes secret-like prompt lines; Deploy redacts child
  runtime output through `RedactLogText`.

## Verification evidence

Passed:

```text
go test ./...                         # 28 packages pass, 2 no-test packages
go vet ./...
go test ./components/internal/envcrypto ./components/sites ./components/deploy ./components/mcp ./components/db
go test ./components/mcp -run 'TestMCPEnvToolsEnforceSiteTargetOwnership|TestMCP.*EnvVar|Test.*Site.*Binding' -count=1
```

Failed release-level check:

```text
go test -race ./components/sites/... ./components/deploy/... ./components/mcp/... ./components/auth/...
```

The failure is in existing `components/deploy` supervisor test fixtures (`TestSupervisorRespawnsAfterBackoff`,
`TestSupervisorNoRespawnAfterStop`, and `TestTriggerRunsThroughSupervisor`) due unsynchronized
mock registry access. It is outside e89s01's secret-path changes but blocks the hard race gate
until fixed or explicitly documented through the bug flow.

## Gate result

**CONCERNS.** Do not release e89s01 as complete yet. Required follow-up:

1. Resolve or disposition the existing Deploy race failures through the active bug flow.
2. Implement Site deploy-key principal support in e89s07.
3. Preserve RED/GREEN commit evidence before story release.
4. Re-run security review against the final integrated diff.
