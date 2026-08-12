# Security Review — e89s01 Implementation

**Date:** 2026-08-12
**Scope:** e89s01 isolated branch `feat/e89s01`.
**Threat Model:** `specs/security/epics/e89/THREAT_MODEL.md`
**Verdict:** PASS for e89s01 scope; e89s07 Site-key principal work remains a dependency gate.

## Findings disposition

| Finding | Status | Evidence |
|---|---|---|
| F-001 CWE-311/CWE-312 optional encryption | Remediated | Canonical base64 root-key parsing; production composition rejects missing/invalid root key; plaintext requires `BIGBASE_ENV=development` and explicit opt-in. |
| F-002 CWE-639 MCP Site target IDOR | Remediated for e89s01 org-auth path | Organization-authenticated Site target authorization runs before existing Site env/key tools with parameterized ownership lookup. Site deploy-key principal support remains explicitly assigned to e89s07. |
| F-003 CWE-754/CWE-532 runtime fail-open | Remediated | Runtime uses EnvResolver; selected resolution failures mark deployment failed; runtime stdout/stderr are redacted. |
| F-004 CWE-209/CWE-532 MCP error disclosure | Remediated | Site env handlers return generic errors while preserving safe not-found behavior; SQL/key/plaintext details are not returned. |

## Controls

- `BIGBASE_ROOT_ENCRYPTION_KEY` is the production input and must decode to exactly 32
  bytes from base64.
- Legacy hex parsing is explicit migration/test input only.
- Plaintext mode requires `BIGBASE_ENV=development` and
  `BIGBASE_ALLOW_PLAINTEXT_SECRETS=true`.
- Site mutation responses omit plaintext and enforce key/value/flag limits.
- EnvResolver returns errors for scan/decryption failures instead of dropping variables.
- MCP target checks use parameterized organization ownership queries and generic denials.
- LLM prompt filtering and Deploy child-process redaction are active.
- Supervisor test fakes are synchronized; the race gate now covers Sites, Deploy, MCP,
  and Auth without findings.

## Verification evidence

Passed:

```text
go test ./...
go vet ./...
go test -race ./components/sites/... ./components/deploy/... ./components/mcp/... ./components/auth/...
go test ./components/internal/envcrypto ./components/sites ./components/deploy ./components/mcp ./components/db
TestMCPEnvToolsEnforceSiteTargetOwnership
TestEnvResolverDecryptionFailureStopsResolution
```

## Residual dependency

Site deploy-key authentication and per-tool secret scopes remain e89s07 work. They are
not silently treated as complete in this story and block final e89 release until s07
passes its own security review.

## Gate result

**PASS for e89s01.** Proceed to story audit/release integration only after the
`audit-code --gate` checklist is completed. Do not release the epic until e89s07 closes
the Site-key principal dependency and the integrated security review is refreshed.
