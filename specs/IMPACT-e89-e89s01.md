type: impact-assessment
context: infra
epic: e89
story: e89s01
mode: lightweight

# Impact Assessment: e89s01

## Target

Harden existing Site-secret storage and delivery: canonical encryption-key
configuration, metadata-only Site mutations, shared build/runtime `EnvResolver`,
fail-closed decryption, log redaction, and ownership-aware MCP Site operations.

## Confirmed dependents and affected paths

- `components/deploy/envresolver.go::EnvResolver` is used by `components/deploy/deploy.go::New`, `Deploy.buildApp`, and deployment trigger paths. ctxo reports 10 confirmed/likely impacted symbols across deploy, main, and MCP adapter clusters.
- `components/deploy/engine.go::Deploy.startApp` still calls legacy `FetchSiteEnvVars`, while build uses `EnvResolver.Resolve`; this is the direct runtime parity gap.
- `components/sites/env_vars.go::Sites.handleEnvVarsRoute` is called by `Sites.handleSiteByID`; mutation/list behavior is exercised by Site API tests and MCP adapters.
- `components/mcp/mcp.go::SiteEnvVarManager` is implemented by `adapters.go::mcpEnvVarAdapter` and consumed by MCP env tools.
- `components/mcp/auth.go::Component.enforceToolAuth` fans into tool registration and the HTTP/stdio server lifecycle.
- `main.go` is a composition-root caller for encryption configuration and adapter wiring; it must remain coordinator-owned when later stories add adapters.

## Existing coverage

- `components/sites/env_vars_test.go`: Site env CRUD, masking, encryption, validation.
- `components/deploy/envresolver_test.go`: precedence and redaction behavior.
- `components/deploy/env_vars_test.go`: legacy fetch/injection behavior.
- `components/mcp/mcp_test.go` and `components/mcp/provisioning_test.go`: MCP Site env tools and tool registration.
- e89s01 task ledger defines five runnable verifies, including affected-package regression.

## Gaps to close

- Production startup currently warns and continues when the Site encryption key is
  malformed; e89s01 requires a valid canonical root key or explicit development-only
  plaintext mode.
- Runtime `startApp` uses `FetchSiteEnvVars` and logs a warning while starting without
  secrets; it must use the shared resolver and fail protected delivery.
- The existing resolver skips decryption failures; protected resolution must return an
  error rather than silently dropping a selected secret.
- Composition-root and MCP Site-target ownership need regression coverage without
  leaking Site existence or plaintext.

## Risk

High/critical security risk. This is a shared API, deployment, authentication, and
crypto boundary with multiple callers. The nominal impact score must be produced by
`assess-impact --lightweight`; if it exceeds 7, `grill-me` is mandatory before code.

## Recommended action

Proceed only as the single-owner foundation story. Freeze the key, masking, resolver,
and target-binding contracts before opening s02. Do not parallelize s01 source edits;
parallelism begins after s03 publishes the SecretManager seam.
