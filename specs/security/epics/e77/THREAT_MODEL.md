# E77 API Surface E2E — Threat Model

## Metadata

| Field | Value |
|---|---|
| Epic | e77 — API Surface E2E |
| Story | e77s01 — API Surface E2E Tests |
| Scope | Playwright request-level E2E tests covering all 17 API components |
| Change type | Net-new test code (no production logic changes) |
| Risk level | **Low** — tests verify existing behavior; no new attack surface introduced |

## Surface Area

The tests exercise ~115+ HTTP routes across 17 registered components:

| # | Component | Routes | Auth required | Notes |
|---|-----------|--------|---------------|-------|
| 1 | **Auth (public)** | 17 | No | Register, login, OAuth, OTP, magic link, password reset, anonymous, refresh, logout |
| 2 | **Auth (protected)** | 22 | Yes | Users, orgs, API keys, site keys, invites, logout-all |
| 3 | **API/Data** | 6 | Yes | Collections CRUD, SQL, onboarding, scaffold |
| 4 | **Storage** | 4 | Yes | Upload, download, list, delete |
| 5 | **Git** | 4 | Yes | Repos CRUD |
| 6 | **Forge** | 5 | Yes | Issues, labels, kanban board, wiki |
| 7 | **GitHub** | 6 | Partial (2 public, 4 protected) | Callback, webhook, status, install, connect, repos |
| 8 | **Sites** | 12+ | Yes | Sites CRUD, deploy, logs, manifest, auth policy, domains, env vars |
| 9 | **Deploy** | 12+ | Yes | Deploy CRUD, logs, stream, rollback, cache, diagnosis |
| 10 | **Monitoring** | 13 | Yes | Health, metrics, logs, alerts, incidents, processes, SSE events |
| 11 | **Functions** | 2 | Yes | Functions CRUD |
| 12 | **CICI** | 5 | Yes | CI runs, workflow CRUD, trigger |
| 13 | **Messaging** | 5 | Yes | Email, SMS, push, telegram, list |
| 14 | **Webhooks** | 3 | Yes | Webhooks CRUD |
| 15 | **Backup** | 2 | Not registered in main.go | Backup/restore (orphan) |
| 16 | **Realtime** | 2 | Yes | WebSocket, status |
| 17 | **MCP** | 2 | Yes (separate port) | MCP stream, health |
| — | **Proxy (infra)** | 6 | Partial | Home, docs, health, version, caddy-allow, mcp-discovery |

## Vulnerability Categories by Priority

### P0 — Must test (blocking)
| ID | Category | Affected components | Risk | Test approach |
|----|----------|-------------------|------|---------------|
| V01 | **Org isolation bypass** | All protected components | **Critical** — tenant A reads tenant B's data | Register two orgs, cross-read resources → expect 404/403 |
| V02 | **Auth bypass** | All protected components | **Critical** — unauthenticated access to protected endpoints | Send requests without Authorization header → expect 401 |

### P1 — High priority
| ID | Category | Affected components | Risk | Test approach |
|----|----------|-------------------|------|---------------|
| V03 | **Error info leakage** | All API endpoints | **High** — stack traces or internal paths in error responses | Assert error shape `{ error: string }` on 4xx/5xx responses |
| V04 | **CORS misconfiguration** | Auth, API, Sites, Storage | **High** — cross-origin data access | OPTIONS preflight + cross-origin requests |
| V05 | **Rate limiter bypass** | Auth (public endpoints) | **High** — brute-force attacks | Rapid registration requests → expect 429 |

### P2 — Medium priority
| ID | Category | Affected components | Risk | Test approach |
|----|----------|-------------------|------|---------------|
| V06 | **Path traversal** | Storage | **Medium** — `../` in file paths reads arbitrary files | Send download with `../` patterns → expect 400 |
| V07 | **IDOR (horizontal auth)** | Sites, Deploy, Git, Forge | **Medium** — user A accesses user B's resource by ID | Cross-tenant resource access → expect 404 |
| V08 | **Token replay after revocation** | Auth | **Medium** — revoked API key still accepted | Create key, revoke, use → expect 401 |

### P3 — Lower priority
| ID | Category | Affected components | Risk | Test approach |
|----|----------|-------------------|------|---------------|
| V09 | **JWT claim tampering** | Auth, all protected | **Low** — modified JWT accepted | Tamper JWT payload → expect 401 |
| V10 | **Expired token acceptance** | Auth, all protected | **Low** — token past expiry still works | Wait for expiry, use token → expect 401 |
| V11 | **Mass assignment** | Collections, Sites | **Low** — setting fields that should be read-only | Include extra fields in POST request |
| V12 | **CSRF protection** | State-changing endpoints | **Low** — missing origin validation | Requests without Origin/Referer headers |

## Risk Assessment

```
Risk score: 7/10
┌──────────────────────────────────────────────┐
│  V01 Org isolation bypass      ████████████  │  Critical
│  V02 Auth bypass               ████████████  │  Critical
│  V03 Error leakage             ████████████  │  High
│  V04 CORS misconfig            ████████      │  High
│  V05 Rate limiting             ████████      │  High
│  V06 Path traversal            ██████        │  Medium
│  V07 IDOR                      ██████        │  Medium
│  V08 Token replay              ██████        │  Medium
└──────────────────────────────────────────────┘
```

**Key insight**: The tests are net-new verification code. The threat model describes what the tests *must assert* — the server-side mitigations already exist in production code. Test failure = real security regression.

## Test Coverage Requirements

### Authentication enforcement (all protected endpoints)
- Unauthenticated request → 401
- Invalid token → 401
- Expired token → 401
- Revoked API key → 401

### Org isolation
- Tenant A resources invisible to Tenant B → 404 or 403
- Cross-tenant IDOR on path-based and query-based lookups

### Input validation
- Path traversal in file paths → 400/403
- Malformed IDs → 400
- Missing required fields → 400

### Output safety
- Error responses are `{ error: string }` — no stack traces
- No sensitive data in error messages

### Transport security
- CORS preflight responds correctly
- Rate limiting on auth endpoints

## Mitigation Guidance for Tests

1. **Use unique user/org per test** — `Date.now()` email pattern prevents collision
2. **Isolate test data** — one test's data must not affect another's (use `test.info().workerIndex`)
3. **Test auth FIRST** — register user → get token → use token in subsequent tests
4. **Verify security before functionality** — 401 test before CRUD happy-path
5. **Clean up state** — delete created resources where possible (or use TTL-based cleanup)
6. **Do not test production infrastructure** — rate limiting is tested with low thresholds; do not flood
7. **Error shapes are contracts** — assert `{ error: string }` not just status codes
