# Threat Model: e76 — Token Lifecycle API E2E

**Date**: 2026-07-11
**Epic**: e76 — Token Lifecycle API E2E
**Nature**: Test-only epic — no production code changes
**Risk Level**: Medium (tests exercise auth boundaries with real secrets)

## 1. Attack Surface Being Tested

### Story 1 — Full Session Lifecycle (Scenario: P0)

| Step | Surface | Token Type | Auth Gate | New/Existing |
|------|---------|------------|-----------|-------------|
| Register | `POST /api/auth/register` | None → issues JWT + refresh | None (public) | Existing |
| Login | `POST /api/auth/login` | None → issues JWT + refresh | None (public) | Existing |
| Refresh | `POST /api/auth/refresh` | Refresh token (opaque 64-char hex) | Request body | Existing |
| Old token reuse | `POST /api/auth/refresh` | Consumed refresh token | Family invalidation | Existing |
| Logout-all | `POST /api/auth/logout-all` | JWT (cookie/header) | Auth middleware | Existing |
| Post-logout-all access | Any protected endpoint | JWT + refresh | Auth middleware | Existing |

### Story 1 — Org API Key Lifecycle (Scenario: P0)

| Step | Surface | Token Type | Auth Gate | New/Existing |
|------|---------|------------|-----------|-------------|
| Generate key | `POST /api/org/{id}/keys` | JWT + `bb_*` key returned | Auth middleware + RequireAdmin | Existing |
| Authenticate with key | Any protected endpoint | `bb_*` raw key in `Authorization: Bearer` | Auth middleware (prefix detect) | Existing |
| Revoke key | `DELETE /api/org/{id}/keys/{keyID}` | JWT | Auth middleware + RequireAdmin | Existing |
| Post-revoke access | Any protected endpoint | Revoked `bb_*` key | Auth middleware (hash lookup + revoked check) | Existing |

### Infrastructure Surface (test harness)

| Component | Role in E2E |
|-----------|-------------|
| Playwright test runner | Executes `token-lifecycle.spec.ts` against running BigBase server |
| BigBase server instance | Target binary, started with `go run . serve` using temp DB |
| Temporary database | SQLite file at `/tmp/bigbase-e2e-token-lifecycle.db` |
| Test credentials | Ephemeral email (`e2e-token-{ts}@test.com`), temporary password |

## 2. Trust Boundaries

```
[Playwright Test Runner] ──HTTP──> [BigBase Server:9999]
                                        │
                                    [Auth Middleware]
                                        │
                              ┌─────────┼─────────┐
                              │         │         │
                         [JWT Verify] [Refresh] [API Key Resolve]
                              │         │         │
                              └─────────┼─────────┘
                                        │
                                  [Handler Layer]
                                        │
                                  [SQLite DB]

Key boundaries:
- Test runner ↔ Server: untrusted network (localhost, no TLS in test)
- Server ↔ DB: same-process SQLite (in-process boundary)
```

## 3. Vulnerability Categories the Tests Should Probe

### 3.1 — Session Lifecycle Vulnerabilities

| # | Category | Finding | Existing Protection | E2E Test Must Verify |
|---|----------|---------|-------------------|---------------------|
| 1 | **Token Rotation Failure** | Old refresh token is accepted after rotation | Atomic UPDATE + family invalidation on replay | Present consumed refresh token → expect 401 |
| 2 | **Family Invalidation Gap** | Logout-all does not invalidate all user refresh token families | `invalidateAllUserTokens` marks all families used | After logout-all, present any refresh token → expect 401 |
| 3 | **Replay Attack Detection** | Concurrent refresh race creates valid duplicate tokens | Atomic UPDATE `WHERE used=0` triggers family invalidation on 0 rows affected | Reuse a consumed token → verify entire family is invalidated (no sibling token works) |
| 4 | **JWT Lifetime Enforcement** | JWT accepted after `exp` claim passes | `verifyJWT` validates `exp` via `jwt-go` library | Wait for JWT expiry (24h default — use config override in test) or verify claim structure |
| 5 | **JWT Reuse After Refresh** | Old JWT still valid after refresh | None — JWT is stateless; no server-side blacklist | Verify old JWT still works until expiry (document as accepted behavior, not a bug) |

### 3.2 — Org API Key Vulnerabilities

| # | Category | Finding | Existing Protection | E2E Test Must Verify |
|---|----------|---------|-------------------|---------------------|
| 6 | **Revocation Enforcement** | Revoked key grants access | Soft-delete via `revoked` column; `ResolveOrgKey` checks `revoked=0` | Use revoked key → expect 401 |
| 7 | **Key Enumeration** | Attacker can enumerate valid `bb_*` keys | HMAC-SHA256 hash + constant-time comparison (subtle) | Verify error messages are identical for "not found" vs "invalid" (no oracle) — **gap if not** |
| 8 | **Privilege Escalation via Key** | `bb_dep_` site key used as org key | `ResolveOrgKey` explicitly rejects `bb_dep_` prefix and `site_id IS NOT NULL` | Attempt org API auth with `bb_dep_` key → expect 401 |
| 9 | **Key Exposure in Error Responses** | Raw key or hash leaked in error body | Handlers return metadata only (key_id, prefix) | Verify error responses do not contain raw key value — **gap if not** |
| 10 | **Key Creation Under Wrong Org** | Key created in org A, used to access org B | `CreateAPIKey` scopes to `orgID`; `ResolveOrgKey` returns `orgID`; middleware enforces org in handler | Create key in org A, authenticate to org B endpoint → expect 403 |

### 3.3 — Cross-Cutting Vulnerabilities

| # | Category | Finding | Existing Protection | E2E Test Must Verify |
|---|----------|---------|-------------------|---------------------|
| 11 | **Token Leak in Test Output** | Raw JWT, refresh token, or `bb_*` key printed to stdout/stderr during test run | None specific to test output | Audit test code for `console.log`, Playwright trace/screenshot config — **gap if not** |
| 12 | **Anonymous Token Write Abuse** | Anonymous JWT used for write operations | Middleware rejects non-GET/HEAD/OPTIONS when `role=anonymous` | Create anon token → attempt POST → expect 403 |
| 13 | **Mixed Auth Context** | JWT context leaked across parallel test workers | Each test generates unique credentials | Verify tests do not share auth state between `describe` blocks — **gap if not** |

## 4. Risk Level Assessment

### Overall Risk: **MEDIUM**

| Dimension | Rating | Rationale |
|-----------|--------|-----------|
| **Production Risk** | **LOW** | Test-only epic — no production code modified. No new endpoints, no new handlers, no new middleware |
| **Test Infrastructure Risk** | **MEDIUM** | Tests manipulate real tokens and API keys. Secrets may appear in CI logs, Playwright traces, or screenshots. Parallel test isolation is critical |
| **False Confidence Risk** | **MEDIUM** | If tests pass due to incorrect setup (e.g., test is not actually authenticated), regression coverage is lost. Tests must actively verify both "success" and "rejection" paths |
| **Secret Exposure Risk** | **HIGH** | Tokens and API keys are the *subject* of the test — they will be present in test code, request bodies, and possibly Playwright output. Mitigation requires deliberate secret handling |
| **Likelihood of Real Exploit** | **LOW** | Existing production protections are already implemented. Tests validate the protections work end-to-end |

## 5. Mitigation Guidance

### 5.1 — Critical (Test Design)

1. **Test both success and failure paths explicitly**: Each lifecycle step must assert both that the expected action succeeds (200) AND the blocked action fails (401/403). A passing test that only checks success may mask auth bypass.
2. **Unique credentials per test run**: Use `e2e-token-{Date.now()}-{randomSuffix}@test.com` email pattern. Never reuse tokens across test suites.
3. **Verify token value is present in response**: After register/login/key-generation, assert `response.token` or `response.key` is a non-empty string matching the expected format (JWT: 3 base64url segments, `bb_*`: matching prefix, refresh: 64 hex chars).

### 5.2 — Critical (Secret Handling in Test Output)

4. **Playwright trace/screenshot config**: Set `trace: 'on-first-retry'` (not `'on'`) and `screenshots: 'only-on-failure'` for the token-lifecycle spec. Traces capture HTTP request/response bodies including raw tokens. See [Playwright security best practices](https://playwright.dev/docs/best-practices#security).
5. **Never log raw tokens in test assertions**: If using `console.log` for debugging, redact token values or remove before commit. CI pipeline should reject PRs containing `.only`, `.skip`, or `console.log` in test files.
6. **CI artifact retention**: Set artifact retention to minimum (7 days) for runs containing auth E2E tests. Mark the CI job as containing sensitive data.
7. **Token redaction in `expect` failure output**: Playwright's `expect` may pretty-print response bodies on failure. Verify no token values appear in test failure messages.

### 5.3 — Important (Test Isolation)

8. **Parallel worker isolation**: Playwright workers run in separate processes. However, if tests share a single BigBase server instance, parallel workers MUST use distinct orgs/users. Use unique org name per worker (e.g., `e2e-org-{workerIndex}`).
9. **Database cleanup on teardown**: After test completion, revoke all created API keys and delete test user accounts. Use `afterAll` hooks. For CI, delete the temp DB file entirely (`rm /tmp/bigbase-e2e-token-lifecycle.db`).
10. **JWT secret isolation**: The test BigBase instance MUST use a **unique, ephemeral JWT secret** (auto-generated by BigBase on first start). Never reuse a production JWT secret.

### 5.4 — Important (Test Rigor)

11. **Verify old JWT acceptance explicitly**: Document in test comments that old JWTs remain valid until expiry (stateless design). Do not assert old JWT returns 401 after refresh — this is expected behavior, not a bug.
12. **Set custom JWT expiry for refresh test**: To avoid waiting 24h for natural expiry, configure the test BigBase instance with `--jwt-access-expiry=30s` and `--refresh-expiry=60s`. Verify the token is rejected after the configured TTL elapses.
13. **Test atomic refresh collision**: The existing unit test (`refreshtoken_test.go`) covers race-condition family invalidation. The E2E test should document this as "covered by unit layer" and focus on the nominal rotation flow.
14. **Test token format validation**: Assert JWT has 3 base64url-encoded segments separated by dots. Assert `bb_*` key starts with `bb_` and has exactly 67 characters (`bb_` + 64 hex). Assert refresh token is exactly 64 hex characters.

### 5.5 — Accepted Risks (No Action Required)

- **JWT reused after refresh stays valid**: Stateless JWT design. Users get a new access token on refresh but old one works until expiry. This is standard JWT behavior. Document in test comments.
- **No TLS in test**: Test server runs on localhost. Acceptable for CI/CD environment on trusted runners.
- **Concurrent race detection E2E**: Already covered by unit tests (`refreshtoken_test.go`). E2E test focuses on sequential lifecycle flows.
- **Rate limiting**: Not covered by e76 E2E tests (covered by integration tests in e74). Token creation rate limits exist at the handler level.

## 6. Test-Specific Security Considerations

### 6.1 — Secret Handling Matrix

| Secret Type | Where It Appears | Risk | Mitigation |
|-------------|------------------|------|------------|
| JWT access token | Request/response body, Authorization header | LOW (short-lived, tied to test user) | No special handling beyond standard output redaction |
| Refresh token | Request body (`refresh_token` field) | MEDIUM (30d TTL by default, enables session hijack) | Set `--refresh-expiry=60s` in test config |
| Raw `bb_*` API key | Response body on creation, subsequent Authorization header | MEDIUM (persistent until revoked) | Revoke in `afterAll` hook; redact from traces |
| Test password | Registration request body | LOW (ephemeral test user, unique per run) | No special handling |
| JWT secret | Server startup config | LOW (auto-generated, ephemeral per test instance) | Ensure auto-generated, never hardcoded |

### 6.2 — Playwright Configuration

```typescript
// Recommended test-specific overrides for token-lifecycle.spec.ts
use: {
  // Traces capture HTTP bodies — limit to failure-only for auth tests
  trace: 'on-first-retry',
  screenshots: 'only-on-failure',
  // Never reuse auth state across tests (no storageState for token lifecycle)
  storageState: undefined,
}
```

### 6.3 — CI Integration

- Add token-lifecycle spec to the CI pipeline as a separate job (not bundled with other E2E tests)
- Run BEFORE browser UI tests (failure blocks UI test execution)
- Set CI job timeout to 120s (registration → login → refresh → old-token check → logout-all → key-gen → auth → revoke → post-revoke)
- Job must run with `--jwt-access-expiry=30s` and `--refresh-expiry=60s` to keep test duration manageable
- CI artifact retention: 7 days maximum for this job's output

## 7. CWE Mapping

| Finding (# from §3) | Category | CWE |
|---------------------|----------|-----|
| 1 | Token Rotation Failure | CWE-613 (Insufficient Session Expiration) |
| 2 | Family Invalidation Gap | CWE-613 (Insufficient Session Expiration) |
| 3 | Replay Attack Detection | CWE-294 (Authentication Bypass by Capture-replay) |
| 4 | JWT Lifetime Enforcement | CWE-613 (Insufficient Session Expiration) |
| 5 | JWT Reuse After Refresh | CWE-613 (Insufficient Session Expiration) |
| 6 | Revocation Enforcement | CWE-613 (Insufficient Session Expiration) |
| 7 | Key Enumeration | CWE-203 (Observable Discrepancy) |
| 8 | Privilege Escalation via Key | CWE-269 (Improper Privilege Management) |
| 9 | Key Exposure in Error Responses | CWE-200 (Information Exposure) |
| 10 | Key Creation Under Wrong Org | CWE-639 (Authorization Bypass) |
| 11 | Token Leak in Test Output | CWE-532 (Insertion of Sensitive Info into Log) |
| 12 | Anonymous Token Write Abuse | CWE-862 (Missing Authorization) |
| 13 | Mixed Auth Context | CWE-270 (Privilege Context Switching Error) |

## 8. Confidence Rubric

All findings in §3 are above confidence threshold 7 and actionable. Findings requiring code-level verification (marked as **gap if not**) should be confirmed during test implementation:

- **Finding 7 (Key Enumeration)**: Verify error messages are identical between "key not found" and "key revoked" cases. If they differ, add to test assertions.
- **Finding 9 (Key Exposure in Error Responses)**: Inspect error response bodies for any raw key content. Add assertion to test if gap confirmed.
- **Finding 11 (Token Leak in Test Output)**: grep for `console.log`, inspect Playwright trace configuration before PR submission.
- **Finding 13 (Mixed Auth Context)**: Verify test file structure uses separate `describe` blocks with independent setup for session lifecycle vs. API key lifecycle.
