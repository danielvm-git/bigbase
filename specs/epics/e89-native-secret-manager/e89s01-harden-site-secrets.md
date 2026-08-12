# e89s01 — Harden Existing Site-Secret Storage and Delivery

**type:** fix
**risk:** P0
**context:** infra
**BCPs:** 5

## 1. Type

Security fix and compatibility preservation.

## 2. Context

BigBase already exposes site-scoped environment variables, but production key
wiring is absent, encryption is optional by default, runtime resolution bypasses
EnvResolver, MCP target ownership is incomplete, and runtime logs are not
redacted.

## 3. Summary

Make the existing site-secret path trustworthy before adding Project secrets.

## 4. Problem

The current implementation can store plaintext, omit secrets after decryption
errors, expose plaintext in mutation responses, and allow MCP callers to target
an arbitrary Site ID.

## 5. Users

BigBase operators, Site owners, deployment automation, and MCP clients.

## 6. Solution

Wire canonical encryption configuration, fail closed in production, unify build
and runtime resolution, redact all deployment output, enforce target ownership,
and return metadata-only mutation responses.

## 7. Alternatives

- Keep optional encryption: rejected because it silently violates the security contract.
- Fix only the key wiring: rejected because runtime and MCP paths remain unsafe.

## 8. Dependencies

ADR 0009; existing `components/sites`, `components/deploy`, `components/mcp`,
`components/auth`, and `components/internal/envcrypto`.

## 9. Assumptions

The current site REST contract remains available. Development tests may opt into
plaintext explicitly; production may not.

## 10. Risks

Existing deployments may contain plaintext or legacy ciphertext. Migration mode
must be explicit and must not guess the format.

## 11. Migration Plan

Add canonical root-key wiring while retaining the legacy key as a read/migration
input. Do not rewrite existing rows in this story.

## 12. Data Model

No new domain tables. Add only metadata needed to identify legacy encryption mode
if required by the migration command.

## 13. API

Existing Site env endpoints remain. POST and PUT return masked metadata, never
`value`. MCP operations require organization and Site binding.

## 14. Affected Code

`main.go`, `components/sites/env_vars.go`, `components/sites/sites.go`,
`components/deploy/envresolver.go`, `components/deploy/env_vars.go`,
`components/deploy/engine.go`, `components/mcp/mcp.go`, `adapters.go`, tests.

## 15. Testing Strategy

Unit tests for key configuration, HTTP isolation, MCP isolation, resolver parity,
redaction, decryption failure, payload limits, and both database drivers.

## 16. Rollback Plan

Disable native key enforcement only in an explicit development profile; revert
route behavior without deleting rows. Preserve the old schema for migration.

## 17. Acceptance Criteria

```gherkin
Scenario: [SC-e89s01-P0-01] Production requires encryption configuration
  Given BigBase starts in production mode without a valid root key
  When the server initializes
  Then startup fails with a generic configuration error

Scenario: [SC-e89s01-P0-02] Site mutation does not return plaintext
  Given an authenticated owner creates or updates a site environment variable
  When the request succeeds
  Then the response contains metadata and a masked preview but no value field

Scenario: [SC-e89s01-P0-03] MCP cannot cross organization boundaries
  Given an MCP org key for organization A
  When it targets a Site owned by organization B
  Then the operation is denied without revealing Site existence

Scenario: [SC-e89s01-P0-04] Runtime uses the same resolver as build
  Given build and runtime flags select the same Site secret
  When both paths resolve the environment
  Then precedence and redaction behavior are identical

Scenario: [SC-e89s01-P0-05] Decryption failure is not fail-open
  Given a selected Site secret cannot be decrypted
  When a protected deployment resolves its environment
  Then resolution fails and the app does not start without the secret
Scenario: [SC-e89s01-P0-06] Deployment and diagnostic output is redacted
  Given a tool echoes a resolved secret value
  When build, runtime, diagnostic, or LLM logs are collected
  Then the plaintext value is absent from every log surface

Scenario: [SC-e89s01-P1-07] Secret payload limits are enforced
  Given a Site secret payload is oversized or has an invalid key
  When the mutation request is submitted
  Then the request is rejected without persistence or plaintext in the error
```

## Requirements
+
#### MODIFIED: Site secret storage and delivery is fail-closed
**Before:** Encryption can silently become plaintext pass-through; runtime and MCP paths bypass the shared resolver or ownership check; mutation responses may contain plaintext.
**After:** Production requires valid encryption configuration, selected decryption failures stop protected delivery, all build/runtime paths use EnvResolver, MCP targets are organization/site-bound, and mutations return metadata only.

## 18. Implementation Steps

1. Wire canonical key configuration and explicit development plaintext mode → verify: `go test ./components/internal/envcrypto ./components/sites ./components/deploy -run 'Test.*Key|Test.*Encryption' -count=1`
2. Remove plaintext mutation responses and enforce payload/flag validation → verify: `go test ./components/sites -run 'TestEnvVarsAPI|TestEnvVarsValidation' -count=1`
3. Route build and runtime through EnvResolver and make decryption errors fatal → verify: `go test ./components/deploy -run 'TestEnvResolver|TestEnvVarInjection' -count=1`
4. Redact runtime/build/diagnostic log paths and bind MCP targets to authenticated organization/site scope → verify: `go test ./components/mcp ./components/deploy -run 'TestEnvVar|Test.*Redact|Test.*Org' -count=1 && echo 'no new security findings in affected paths'`
5. Verify both database drivers and full affected packages → verify: `go test ./components/sites ./components/deploy ./components/mcp ./components/db -count=1 && go test ./components/db -run 'Test.*(SQLite|Postgres)' -count=1 && echo 'no new security findings in affected paths'`

## 19. Verification Script

1. Start BigBase with a valid key and two organizations.
2. Create a Site secret as organization A.
3. Attempt the same REST and MCP operations as organization B.
4. Run a build and runtime path that prints a known secret.
5. Inspect deployment logs and confirm the value is absent.
6. Remove the key in production mode and confirm startup fails.

## 20. Out of scope

Project tables, SecretVersion storage, Admin UI redesign, external KMS, and
legacy data re-encryption.
## 21. Zoom-Out Check

- **Purpose:** `envcrypto` owns key parsing and AES-GCM primitives; Sites owns Site env storage/API; EnvResolver owns build/runtime environment and redaction; MCP adapters enforce machine-facing access.
- **Callers:** `main.go` composition, Site HTTP handlers, Deploy `buildApp`/`startApp`/resume paths, MCP HTTP/stdio tools, and LLM/diagnostic log paths.
- **Contracts:** production encryption fails closed, development plaintext is explicit, Site writes are metadata-only, authenticated Site/org ownership is enforced, selected decryption failures stop protected delivery, and all secret-derived output is redacted.
