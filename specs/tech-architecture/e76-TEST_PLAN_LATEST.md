# Test Design: e76 — Token Generation, Refresh, Revocation Journeys

> **Scope:** All token types across the BigBase auth system — JWT access tokens, refresh tokens, org API keys (`bb_*`), site deploy keys (`bb_dep_*`), anonymous tokens, and MCP-provisioned credentials. Covers HTTP API, middleware, and MCP tool surfaces.
>
> **Rationale:** 0 E2E tests exist for any token journey. Existing coverage is entirely Go `httptest` unit tests. Token auth is a P0 security boundary — a broken token flow locks every user out of the entire platform.
>
> **Risk level:** P0

---

## 1. Token Architecture Overview

```
Token Type          Format           Lifetime     Scope              Revocable  Storage
────────────────────────────────────────────────────────────────────────────────────
JWT access token    JWT (HS256)      24h (cfg)    User session       No         JWT claims only
Refresh token       40-char hex      30d (cfg)    User session       Yes        refresh_tokens table
Org API key         bb_<hex>         Permanent    Org-level access   Yes        org_api_keys table
Site deploy key     bb_dep_<hex>     Permanent    Site-level access  Yes        org_api_keys table
Anonymous token     JWT (HS256)      1h           Anonymous access   No         JWT claims only
MCP provisioned     bb_dep_<hex>     Permanent    Per-site deploy    Yes        (via MCP → site keys)
```

---

## 2. Risk Matrix & Scenarios

### P0 — Critical (security boundary, release-blocking)

| Scenario ID | Behavior | Existing Unit Coverage | Missing Test Level | Target File |
|---|---|---|---|---|
| SC-e76-P0-01 | **Full register→login→use→refresh→revoke journey** — register, login, call `/api/auth/me` (200), refresh, old refresh rejected (401), new refresh works, logout-all, all refreshes rejected (401), access still works for `/api/auth/me` | ✅ e50_test.go (httptest) | **E2E** — no running-server test of the orchestrated flow | `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P0-02 | **Org API key created → authenticates → revoked → rejected** — create `bb_*` key, call protected endpoint (200), revoke, call same endpoint (401) | ✅ apikeys_test.go (httptest) | **E2E** | `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P0-03 | **Site deploy key created → resolves → revoked → rejected** — create `bb_dep_*` key, use deploy endpoint (200), revoke, fail (401) | ✅ sitekeys_test.go, sitekey_handlers_test.go (httptest) | **E2E** | `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P0-04 | **Anonymous token created → used → expired → rejected** — POST anonymous, call endpoint (200), wait/force expiry, call again (401) | ❌ No test exists | **Unit + E2E** | `components/auth/anonymous_test.go`, `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P0-05 | **Refresh token replay detection → family invalidation** — rotate twice, replay first token, all family tokens revoked | ✅ refreshtoken_test.go (httptest) | **E2E** | `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P0-06 | **MCP provision CI credentials → deploy → revoke via MCP** — call `bigbase_provision_ci_credentials`, get deploy key, trigger deploy with it, call `bigbase_revoke_site_key`, deploy rejected | ❌ No test exists | **Unit + E2E** | `components/mcp/provisioning_test.go`, `tests/e2e/mcp-token.spec.ts` |

### P1 — High (common user flows, regression-prone)

| Scenario ID | Behavior | Existing Unit Coverage | Missing Test Level | Target File |
|---|---|---|---|---|
| SC-e76-P1-01 | **Registration returns valid token structure** — response has `token`, `refresh_token`, `expires_in`, `expires_at`, `user` | ✅ e50_test.go (httptest) | **E2E** | `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P1-02 | **Login returns valid token structure** — same fields as register | ✅ e50_test.go (httptest) | **E2E** | `tests/e2e/login.spec.ts` (already exists) |
| SC-e76-P1-03 | **Custom token lifetimes applied** — `--jwt-access-expiry=5m` → `expires_in=300` | ✅ e50_test.go (httptest) | **E2E** | `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P1-04 | **Token used in Cookie auth** — login sets cookie, subsequent requests use cookie, logout clears it | ✅ logout_test.go (httptest) | **E2E** | `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P1-05 | **Invalid/expired/malformed JWT returns 401** — tampered token, expired token, wrong format | ❌ Partial (only in JWT lib tests) | **Unit + E2E** | `components/auth/auth_test.go`, `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P1-06 | **Missing Authorization header returns 401** — unauthenticated request to protected endpoint | ❌ No explicit test | **Unit** | `components/auth/auth_test.go` |
| SC-e76-P1-07 | **Org API key with wrong origin rejects** — key created for org A cannot access org B resources | ✅ apikeys_test.go (httptest) | **E2E** | `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P1-08 | **Site deploy key cannot access other sites** — key scoped to site A fails on site B | ❌ No cross-site test | **Unit** | `components/auth/sitekeys_test.go` |
| SC-e76-P1-09 | **MCP server serves `/mcp` with valid auth** — SSE stream connects, tool calls succeed | ❌ No test exists | **E2E** | `tests/e2e/mcp-token.spec.ts` |
| SC-e76-P1-10 | **MCP server rejects invalid bearer token** — `/mcp` returns 401 for bad token | ❌ No test exists | **E2E** | `tests/e2e/mcp-token.spec.ts` |
| SC-e76-P1-11 | **Create deploy key → shown once → not in list response** — POST returns raw token, GET list has no raw token column | ✅ sitekey_handlers_test.go (httptest), sitekeys_test.go | **E2E** | `tests/e2e/token-lifecycle.spec.ts` |

### P2 — Medium (secondary features, edge cases)

| Scenario ID | Behavior | Existing Unit Coverage | Missing Test Level | Target File |
|---|---|---|---|---|
| SC-e76-P2-01 | **Rate limiting on deploy key creation** — 11th POST in 1 hour returns 429 | ❌ No test exists (e74s01 tasks mention rate limit but test not found) | **Unit** | `components/auth/sitekey_handlers_test.go` |
| SC-e76-P2-02 | **Deploy key name validation** — empty name, >100 chars, special chars rejected | ✅ sitekey_handlers_test.go (httptest) | **E2E** | `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P2-03 | **Revoke nonexistent deploy key** — DELETE on missing keyID returns 404 | ✅ sitekeys_test.go (httptest) | **E2E** | `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P2-04 | **Revoke already-revoked deploy key (idempotent)** — revoke twice returns 200 both times | ❌ No test exists | **Unit** | `components/auth/sitekeys_test.go` |
| SC-e76-P2-05 | **Refresh token with wrong family** — token from other user rejected (401) | ✅ refreshtoken_test.go (httptest) | **E2E** | `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P2-06 | **Logout-all on user with zero tokens (idempotent)** — returns 200 even when no refresh tokens exist | ✅ e50_test.go (httptest) | **E2E** | `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P2-07 | **Password reset invalidates refresh tokens** — forgot→reset→old refresh rejected | ❌ No integration between flows tested | **Integration** | `components/auth/passwordreset_test.go`, `tests/e2e/token-lifecycle.spec.ts` |
| SC-e76-P2-08 | **Token never written to server logs** — grep handlers for `token` format strings in log calls | ✅ e74s01 task (log-redaction) | **Unit** | `components/auth/auth.go` (static analysis) |
| SC-e76-P2-09 | **MCP `bigbase_list_site_keys` returns metadata only** — no raw token in response | ❌ No test exists | **Unit** | `components/mcp/mcp_test.go` |
| SC-e76-P2-10 | **MCP `bigbase_revoke_site_key` idempotent** — revoke same key twice returns success both times | ❌ No test exists | **Unit** | `components/mcp/mcp_test.go` |

### P3 — Low (edge cases, states)

| Scenario ID | Behavior | Existing Unit Coverage | Missing Test Level | Target File |
|---|---|---|---|---|
| SC-e76-P3-01 | **Empty JWT secret env var** — `BIGBASE_JWT_SECRET=""` falls back to auto-gen (with warning) | ✅ e50_test.go (httptest) | **Unit** | already covered |
| SC-e76-P3-02 | **Short JWT secret rejected** — `BIGBASE_JWT_SECRET=short` panics with clear message | ✅ e50_test.go (httptest) | **Unit** | already covered |
| SC-e76-P3-03 | **Known default JWT secret rejected** — `test-secret-32-chars!!!` panics with "known default" | ✅ e50_test.go (httptest) | **Unit** | already covered |
| SC-e76-P3-04 | **Access expiry < 1 minute rejected** — `--jwt-access-expiry=30s` fatals at startup | ✅ e50_test.go (httptest) | **Unit** | already covered |
| SC-e76-P3-05 | **Refresh expiry < access expiry rejected** — not yet implemented | ❌ No test exists | **Unit** | `components/auth/auth_test.go` |
| SC-e76-P3-06 | **Bearer token with wrong scheme** — `Authorization: Basic xxx` treated as unauthenticated | ❌ No test exists | **Unit** | `components/auth/auth_test.go` |

---

## 3. Test Level Distribution

| Test Level | Count | Rationale |
|---|---|---|
| **Unit** (Go `httptest`) | 12 new | Core auth logic, token parsing, validators, error paths — fast, focused, no db needed |
| **Integration** (Go + sqlite) | 4 new | Cross-flow interactions (password reset → token invalidation), MCP → site key → deploy |
| **E2E** (Playwright HTTP) | 20 new | Full server lifecycle — register through revoke, running server, real middleware stack |

**Default to lowest possible level.** Logic that can be proven with a table-driven unit test (e.g., JWT claim parsing, duration validation) stays at Unit. Flows that require a running server, real DB, and middleware chaining (e.g., full auth journey) go to E2E.

---

## 4. Fixture Architecture

### E2E Fixtures (`tests/e2e/fixtures/`)

```typescript
// tests/e2e/fixtures/auth.ts — Augment existing fixture
export interface AuthFixture {
  email: string
  password: string
  accessToken: string
  refreshToken: string
  userId: string   // from /api/auth/me
}

export async function createUser(request: APIRequestContext): Promise<AuthFixture>
export async function refreshTokens(request: APIRequestContext, refreshToken: string): Promise<{accessToken: string, refreshToken: string}>
export async function logoutAll(request: APIRequestContext, accessToken: string): Promise<void>
```

```typescript
// tests/e2e/fixtures/site.ts
export interface SiteFixture {
  siteId: string
  deployKey: string       // bb_dep_* raw token
  deployKeyId: string
}

export async function createSiteWithDeployKey(request: APIRequestContext, authToken: string): Promise<SiteFixture>
export async function createOrgAPIKey(request: APIRequestContext, authToken: string, orgId: string): Promise<{apiKey: string, keyId: string}>
```

```typescript
// tests/e2e/fixtures/mcp.ts
export async function provisionDeployKey(name: string): Promise<{deployKey: string, keyId: string}>
```

### Page Objects (for UI tests of deploy keys)

```typescript
// tests/e2e/pages/deploy-keys-tab.page.ts
export class DeployKeysTab {
  async generateKey(name: string): Promise<{keyValue: string}>
  async listKeys(): Promise<Array<{keyId: string, name: string, prefix: string}>>
  async revokeKey(keyId: string): Promise<void>
  async getKeyCount(): Promise<number>
}
```

---

## 5. NFR Verification

| NFR Type | Requirement | Verification Command |
|---|---|---|
| **Perf** | Token auth adds < 5ms p99 latency | `go test -bench=BenchmarkAuth -benchtime=10s ./components/auth/` |
| **Isolation** | Concurrent token operations don't race | `go test -race -run 'TestToken|TestRefresh|TestSiteKey' ./components/auth/ -count=10` |
| **Coverage** | Every token format (JWT, bb_, bb_dep_) tested through create→use→revoke | `rg 'Test.*Key|Test.*Token' components/auth/*_test.go | wc -l` |
| **Security** | Raw token never in logs | `rg 'token.*%s|token.*%v' components/auth/auth.go && echo FAIL || echo PASS` |

---

## 6. Implementation Order

| Layer | Effort | Dependencies |
|---|---|---|
| 1. **P0-P1 Unit tests** (new missing coverage) | 4 BCPs | None |
| 2. **E2E fixture infra** (auth.ts, site.ts, mcp.ts) | 2 BCPs | None |
| 3. **P0-P1 E2E tests** (full lifecycle, MCP) | 6 BCPs | Fixtures |
| 4. **P2 Unit tests** (rate limit, idempotent revoke) | 3 BCPs | None |
| 5. **P2-P3 E2E tests** (edge cases, validation) | 3 BCPs | Fixtures |
| 6. **MCP tool tests** (unit + E2E) | 2 BCPs | MCP server |

**Total:** 20 BCPs across 6 phases.

---

## 7. Out of Scope

- JWT secret rotation / multiple active secrets (no epic)
- Per-session audit trail for token usage (no epic)
- OAuth token exchange (Google token → BigBase token) — covered by e01
- WebSocket token validation at `/realtime` — deferred to e11
- Visual regression testing of deploy-keys UI — covered by e75
- Admin-only token management UI — no admin role in E2E setup
- Performance / load testing of auth endpoints
