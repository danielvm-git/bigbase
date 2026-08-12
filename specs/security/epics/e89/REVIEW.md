# Security Review — Epic e89 Native Secret Manager

**Date:** 2026-08-11
**Scope:** Pre-implementation review of e89 baseline attack paths, threat model, capsule, and planned controls.
**Branch/Diff:** `main` at `b535145d1`; no implementation branch exists.
**Threat Model:** `specs/security/epics/e89/THREAT_MODEL.md`
**Verdict:** NOT READY for kickoff until the findings below are fixed and re-reviewed.

## Scope Resolution

Reviewed:

- `components/sites` Site environment storage and HTTP ownership checks.
- `components/deploy` EnvResolver, runtime injection, build/runtime redaction, and legacy fetch paths.
- `components/mcp` authentication, Site environment tools, and error formatting.
- `main.go` composition-root key and adapter wiring.
- e89 ADRs, scope, test plan, task ledgers, and parallel execution plan.

The review is pre-implementation. Findings describe confirmed baseline exploit paths that
e89 is intended to remove; planned mitigations are not treated as implemented controls.

## Findings

### F-001 — HIGH — CWE-311/CWE-312 — Optional Site-secret encryption

**Confidence:** 10/10
**Evidence:** `components/sites/sites.go:161-180` parses the optional legacy key and
continues with a warning when the key is invalid. The current path permits no-key/no-op
storage behavior.

**Exploit scenario:** An operator starts production without a valid encryption key. A
Site environment mutation can persist a plaintext/no-op value, creating database and
backup exposure.

**Required mitigation:** e89s01 must wire canonical `BIGBASE_ROOT_ENCRYPTION_KEY`, reject
missing/invalid production configuration, and allow plaintext only through an explicit
development opt-in. Add startup and mutation regression tests before accepting s01.

**Gate:** SC-e89s01-P0-01; e89s01t1.

### F-002 — HIGH — CWE-639 — MCP Site target IDOR

**Confidence:** 10/10
**Evidence:** `components/mcp/mcp.go` accepts caller-supplied `site_id` for Site env
operations and delegates through `SiteEnvVarManager`; the existing adapter forwards to
Sites without binding the target to the authenticated principal.

**Exploit scenario:** A valid organization or Site credential submits another Site ID to
`get_site_env_vars`, `set_site_env_vars`, or `delete_site_env_var` and accesses or mutates
secrets outside its authorized Site.

**Required mitigation:** Resolve organization/Site ownership before every operation.
Represent organization keys and Site deploy keys in one authenticated principal model.
Keep Site deploy keys Site-bound and test cross-org/cross-Site denial over HTTP and stdio.

**Gate:** SC-e89s01-P0-03, SC-e89s07-P0-03, SC-e89s07-P0-04.

### F-003 — HIGH — CWE-754/CWE-532 — Runtime fail-open after secret failure

**Confidence:** 10/10
**Evidence:** `components/deploy/engine.go:635-640` logs a runtime env fetch failure and
continues starting the application without Site secrets. `components/deploy/envresolver.go`
currently skips individual decryption failures while continuing resolution.

**Exploit scenario:** A protected deployment starts without a required credential after a
key/decryption/database failure. The application may run in an unsafe or misconfigured
state, while the failure path can expose secret-related details through logs.

**Required mitigation:** Make selected decryption/fetch failures fatal for protected
build/runtime paths. Route build and runtime through one resolver. Redact diagnostics,
LLM output, and child-process logs with the shared redaction primitive.

**Gate:** SC-e89s01-P0-04, SC-e89s01-P0-05, SC-e89s06-P0-02, SC-e89s06-P0-03.

### F-004 — HIGH — CWE-209/CWE-532 — MCP internal error disclosure

**Confidence:** 9/10
**Evidence:** Existing MCP Site environment handlers format underlying errors into client
responses, including patterns equivalent to `Failed to ...: %v` in
`components/mcp/mcp.go`.

**Exploit scenario:** A database, SQL, key, or decryption error reaches an MCP client and
reveals internal implementation details or sensitive material. Tool logs also include
caller-controlled target metadata.

**Required mitigation:** Return typed generic MCP errors. Keep diagnostic detail in
structured server logs after redaction. Assert that SQL text, key material, ciphertext,
and plaintext are absent from HTTP and stdio results.

**Gate:** SC-e89s07-P0-05; e89s07t4.

## Planned-control verification blockers

These are not counted as implemented mitigations until code and tests exist:

1. The `read_project_secret_value` MCP action must be distinct from metadata-only
   `get_project_secret`, require `mcp:secrets:read`, and be audited.
2. REST role/action checks must use the documented matrix and bounds: 30 mutations per
   actor/Project/minute, 128-byte keys, 64 KiB values, and 1,000-key batches.
3. SecretManager must enforce AAD scope binding and immutable versions under a transaction
   seam implemented by both SQLite and PostgreSQL paths.
4. `main.go` composition wiring must remain coordinator-owned; story branches must not
   bypass the frozen SecretManager policy seam.
5. The UI must never store value-read responses in list state or echo imported values in
   errors.

## Security gate result

**FAIL / NOT READY.** Four HIGH-confidence baseline findings remain until e89s01 and the
later dependent stories implement and verify their controls. No security exception is
recommended. Do not start `develop-tdd` or mark e89s01 complete until the dedicated
security tests pass and this review is refreshed against the implementation diff.

## Required verification before re-review

```bash
go test -race ./components/sites/... ./components/deploy/... ./components/mcp/... ./components/auth/... ./components/secrets/...
go test ./...
go vet ./...
golangci-lint run ./...
```
