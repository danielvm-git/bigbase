# Impact Assessment — e72s01: MCP HTTP Bearer Middleware

**Mode:** lightweight  
**Date:** 2026-07-08  
**Risk score:** 6 / 10 (Medium — proceed; no grill-me gate)

## Target

- `components/mcp/mcp.go` — `Handler()`, new `bearerAuthMiddleware`, `KeyAuthenticator` interface
- `main.go` — inject auth adapter into `mcp.Options`
- `components/mcp/mcp_test.go` — new auth middleware tests

## Dependents (fan-out)

| Caller | Usage |
|--------|-------|
| `main.go:420-431` | Constructs MCP component with injected deps |
| `components/proxy/hosts_test.go` | Public `/mcp` route expectations (may need auth-aware tests) |
| `mcp_adapters.go` (untracked) | SiteKeyCreator adapter pattern — mirror for KeyAuthenticator |

## Affected Stories

- **e72s01** (this story): Bearer middleware + org context
- **e72s02**: Scope gate builds on middleware context
- **e72s03**: Public allow-list bypass in same middleware chain
- **e71** (later): Host-level route auth — orthogonal; no conflict

## Test Coverage

| File | Covers |
|------|--------|
| `components/mcp/mcp_test.go` | HTTP transport, ping, SSE, non-localhost host header |
| `components/mcp/provisioning_test.go` | Write tools without auth (will need updating for 401) |
| `components/auth/apikeys_test.go` | `ResolveAPIKey` — reuse, no change expected |

**Gap:** No Bearer middleware tests yet — e72s01 tasks define `TestBearerAuthMiddleware`, `TestBearerAuthContext`.

## Risk Classification: Medium

- Touches shared HTTP handler used by public proxy route
- 54 existing MCP tests must stay green; provisioning tests will need auth fixtures
- Interface injection pattern already established (`SiteKeyCreator`)
- No breaking change to stdio transport

## Recommended Action

**Proceed** with TDD per e72s01-tasks.yaml. Update provisioning tests to present valid Bearer tokens where write tools are exercised.
