# Test Design: e77 — Remaining API Surface E2E Coverage

> **Scope:** All API endpoints across 17 components that have zero E2E coverage. 598 Go unit tests exist but only 3 Playwright files cover 8 endpoints out of ~148 registered routes.
>
> **Risk level:** P0 (authentication and authorization boundaries tested only in-process, never through full HTTP stack)
>
> **Rationale:** Every component has unit tests via `httptest`, but those run in-process with no real server startup, no middleware chain, no CORS, no serialization boundary, and no cross-component interaction. A broken middleware, miswired route, or config issue that works in `httptest` will fail silently in production.

---

## 1. Coverage Overview by Component

| Component | Go unit test functions | E2E tests | Routes | E2E gap |
|---|---|---|---|---|
| auth | 111 | 0 | 29 | 27 routes |
| api (data/collections/env) | 40 | 0 | 9 | 9 |
| deploy | 177 | 0 | 10 | 9 |
| sites | 20 | 0 | 15 | 12 |
| monitoring | 32 | 0 | 14 | 14 |
| mcp | 36 | 0 | 1 (19 tools) | 19 tools |
| functions | 28 | 0 | 5 | 5 |
| storage | 21 | 0 | 4 | 4 |
| messaging | 20 | 0 | 5 | 5 |
| proxy | 32 | 0 | 5 | 5 |
| realtime | 9 | 0 | 2 | 2 |
| cici | 11 | 0 | 5 | 5 |
| github | 8 | 0 | 6 | 6 |
| forge | 8 | 0 | 7 | 7 |
| git | 7 | 0 | 3 | 3 |
| backup | 3 | 0 | 2 | 2 |
| webhooks | 1 | 0 | 3 | 3 |
| admin | 8 | 0 | 1 (static) | 1 |
| **Total** | **598** | **3 files** | **~148** | **~140** |

---

## 2. Risk Matrix & Scenarios

### P0 — Critical (security boundary, release-blocking)

| ID | Scenario | Component | Existing Unit | Needs |
|---|---|---|---|---|
| SC-e77-P0-01 | **Org CRUD** — create org → list orgs → get org → update org → delete org; verify auth guards at each step | auth/org | ✅ membership_test.go | E2E |
| SC-e77-P0-02 | **Org member invites** — create invite → accept invite → list members; invite from wrong org rejected | auth/org | ✅ membership_test.go | E2E |
| SC-e77-P0-03 | **Org isolation** — user A's token cannot access org B's sites, keys, or membership | auth + sites | ❌ no cross-org E2E | E2E |
| SC-e77-P0-04 | **Collections CRUD** — create collection → insert item → query → update → delete; only auth'd users can access | api | ✅ api_test.go | E2E |
| SC-e77-P0-05 | **SQL endpoint auth** — POST /api/sql with valid token succeeds; without token returns 401 | api | ✅ api_test.go | E2E |
| SC-e77-P0-06 | **Full deploy pipeline** — create repo → create site → trigger deploy → poll status → get logs; verify each state transition | deploy | ✅ deploy_test.go | E2E |
| SC-e77-P0-07 | **Deploy rollback** — deploy twice → rollback to first → verify second is rolled back | deploy | ✅ rollback_test.go | E2E |
| SC-e77-P0-08 | **GitHub install status** — GET /api/github/status returns current installation state | github | ✅ github_test.go | E2E |
| SC-e77-P0-09 | **Storage upload → download → delete** — upload file → GET file by ID → DELETE; file accessible only with auth | storage | ✅ storage_test.go | E2E |
| SC-e77-P0-10 | **Real-time WebSocket connection** — /realtime WebSocket connects, receives events, disconnects cleanly | realtime | ✅ realtime_test.go | E2E |
| SC-e77-P0-11 | **MCP `bigbase_ping` → `bigbase_list_sites` → `bigbase_deploy_site`** — MCP tool invocation via /mcp SSE | mcp | ✅ mcp_test.go | E2E |
| SC-e77-P0-12 | **Forge issue creation & board** — create issue → add label → view board; verify permissions | forge | ✅ forge_test.go | E2E |

### P1 — High (common user flows, regression-prone)

| ID | Scenario | Component | Existing Unit | Needs |
|---|---|---|---|---|
| SC-e77-P1-01 | **Site env vars** — create → list → update → delete env var on a site | sites | ✅ env_vars_test.go | E2E |
| SC-e77-P1-02 | **Site custom domains** — add domain → verify DNS → remove domain | sites | ✅ domains_test.go | E2E |
| SC-e77-P1-03 | **Site manifest** — GET manifest → POST update manifest → GET verifies updated | sites | ✅ sites_test.go | E2E |
| SC-e77-P1-04 | **Site auth policy** — GET policy → POST updated policy → GET verifies | sites | ✅ sites_test.go | E2E |
| SC-e77-P1-05 | **Samples listing & deploy** — GET /api/samples lists → POST deploy sample app → verify status | deploy | ✅ samples_test.go | E2E |
| SC-e77-P1-06 | **Deploy cache management** — GET cache → DELETE cache → POST prune | deploy | ✅ cache_api_test.go | E2E |
| SC-e77-P1-07 | **Monitoring health check** — GET /api/monitoring/health returns component status | monitoring | ✅ monitoring_test.go | E2E |
| SC-e77-P1-08 | **Monitoring metrics** — GET /api/monitoring/metrics returns structured metrics | monitoring | ✅ monitoring_test.go | E2E |
| SC-e77-P1-09 | **Monitoring alert CRUD** — create alert rule → list → update → delete | monitoring | ✅ alert_rules_test.go | E2E |
| SC-e77-P1-10 | **Serverless function CRUD + run** — create function → list → get → run → get logs | functions | ✅ functions_test.go | E2E |
| SC-e77-P1-11 | **Messaging email/SMS/push send** — POST each channel → verify message stored in history | messaging | ✅ messaging_test.go | E2E |
| SC-e77-P1-12 | **CI/CD workflow CRUD + run** — save workflow → list → trigger run → get logs | cici | ✅ cici_test.go | E2E |
| SC-e77-P1-13 | **Git repo CRUD** — create repo → list → get → delete | git | ✅ git_test.go | E2E |
| SC-e77-P1-14 | **Webhook CRUD** — create webhook → list → delete | webhooks | ✅ webhooks_test.go | E2E |
| SC-e77-P1-15 | **Backup & restore** — POST /api/backup creates snapshot → POST /api/restore restores it | backup | ✅ backup_test.go | E2E |
| SC-e77-P1-16 | **Global env vars** — create env var → list → delete | api/env | ✅ envvars_test.go | E2E |
| SC-e77-P1-17 | **Monitoring log ingestion + search** — POST log → GET /api/monitoring/logs finds it → GET by ID | monitoring | ✅ monitoring_test.go | E2E |
| SC-e77-P1-18 | **Scaffold db/repo/function** — POST each scaffold endpoint returns generated content | api | ✅ api_test.go | E2E |
| SC-e77-P1-19 | **Realtime status endpoint** — GET /api/realtime/status returns hub stats | realtime | ✅ realtime_test.go | E2E |
| SC-e77-P1-20 | **Monitoring Prometheus endpoint** — GET /api/monitoring/metrics/prometheus returns prometheus format | monitoring | ✅ monitoring_test.go | E2E |

### P2 — Medium (secondary features, edge cases)

| ID | Scenario | Component | Existing Unit | Needs |
|---|---|---|---|---|
| SC-e77-P2-01 | **Collection pagination, filtering, sorting** — collections with query params for filter/sort/offset/limit | api | ✅ (see e30s03) | E2E |
| SC-e77-P2-02 | **Deploy with sample pre-built app** — deploys selected sample to user's site | deploy | ✅ samples_test.go | E2E |
| SC-e77-P2-03 | **Deploy build cache** — first deploy cold cache → second deploy with cache hit → verify timing | deploy | ✅ cache_test.go | E2E |
| SC-e77-P2-04 | **Deploy with custom build command** — site with build command runs correctly | deploy | ✅ manifest_test.go | E2E |
| SC-e77-P2-05 | **Monitoring SSE events stream** — connect to /api/monitoring/events, receive events | monitoring | ✅ events_test.go | E2E |
| SC-e77-P2-06 | **Monitoring incidents** — GET incidents → GET incident investigation details | monitoring | ✅ monitoring_test.go | E2E |
| SC-e77-P2-07 | **Storage file thumbnail** — upload image → GET thumbnail with w/h params → valid image returned | storage | ✅ storage_test.go | E2E |
| SC-e77-P2-08 | **Storage path traversal protection** — requesting `../` in file path returns 404, not root FS access | storage | ✅ storage_test.go | E2E |
| SC-e77-P2-09 | **Forge wiki CRUD** — create wiki page → GET → PUT update → GET verifies content | forge | ✅ forge_test.go | E2E |
| SC-e77-P2-10 | **Forge issue comments** — create issue → add comment → GET issue with comments | forge | ✅ forge_test.go | E2E |
| SC-e77-P2-11 | **GitHub repo connect + list** — connect GitHub repo → list connected repos → verify metadata | github | ✅ github_test.go | E2E |
| SC-e77-P2-12 | **CORS enforcement** — preflight OPTIONS with allowed origin succeeds; disallowed origin rejected | auth | ✅ cors_test.go | E2E |
| SC-e77-P2-13 | **Cookie secure flag** — login over HTTPS sets Secure flag on cookie | auth | ✅ cookie_secure_test.go | E2E |
| SC-e77-P2-14 | **Monitoring processes endpoint** — GET /api/monitoring/processes returns process list | monitoring | ✅ processes_test.go | E2E |
| SC-e77-P2-15 | **Monitoring org usage** — GET /api/orgs/{id}/usage returns usage data | monitoring | ✅ org_usage_test.go | E2E |
| SC-e77-P2-16 | **Deploy drain policy** — drain a site → deploys are rejected → undrain → deploys accepted | deploy | ✅ drain_test.go | E2E |
| SC-e77-P2-17 | **Deploy supervisor health** — supervisor health check passes when all conditions met | deploy | ✅ supervisor_health_test.go | E2E |
| SC-e77-P2-18 | **Messaging message history** — send email and SMS → GET /api/messaging/messages lists both | messaging | ✅ messaging_test.go | E2E |
| SC-e77-P2-19 | **Telegram messaging** — POST /api/messaging/telegram with valid body | messaging | ✅ messaging_test.go | E2E |
| SC-e77-P2-20 | **Proxy version endpoint** — GET /api/version returns version info | proxy | ✅ proxy_test.go | E2E |

### P3 — Low (edge cases, error states, hardening)

| ID | Scenario | Component | Existing Unit | Needs |
|---|---|---|---|---|
| SC-e77-P3-01 | **Collection invalid collection name rejected** — SQL injection chars, empty name, special chars | api | ✅ api_test.go | E2E |
| SC-e77-P3-02 | **Collection pagination boundary** — offset beyond total count returns empty, limit=0 returns default | api | ❌ partial | Unit + E2E |
| SC-e77-P3-03 | **Deploy with nonexistent repo** — POST /api/deploy with bad repo_id returns 400 | deploy | ✅ deploy_test.go | E2E |
| SC-e77-P3-04 | **Deploy with failing build** — bad build command → deployment marked as failed, logs contain error | deploy | ✅ deploy_test.go | E2E |
| SC-e77-P3-05 | **Delete deployment at each state** — try DELETE on pending, building, running, failed — only some succeed | deploy | ✅ deploy_test.go | E2E |
| SC-e77-P3-06 | **Function with runtime error** — function panics → run returns error, logs contain stack trace | functions | ✅ functions_test.go | E2E |
| SC-e77-P3-07 | **Function with timeout** — function hangs → run returns timeout error | functions | ✅ functions_test.go | E2E |
| SC-e77-P3-08 | **Monitoring log with invalid body** — POST malformed JSON returns 400 | monitoring | ✅ monitoring_test.go | E2E |
| SC-e77-P3-09 | **Messaging with invalid payload** — missing required fields returns 400 | messaging | ✅ messaging_test.go | E2E |
| SC-e77-P3-10 | **Storage file too large** — upload exceeds size limit returns 413 | storage | ❌ no size limit test | Unit |
| SC-e77-P3-11 | **GitHub webhook handler** — POST /api/github/webhook processes push event | github | ✅ github_test.go | E2E |
| SC-e77-P3-12 | **Webhook delivery retry** — webhook target returns 500 → retries with backoff | webhooks | ✅ webhooks_test.go | E2E |
| SC-e77-P3-13 | **CI/CD with nonexistent workflow** — trigger run on missing workflow returns 404 | cici | ✅ cici_test.go | E2E |
| SC-e77-P3-14 | **MCP invalid tool call** — call nonexistent tool returns error, not panic | mcp | ❌ no invalid-tool test | Unit |
| SC-e77-P3-15 | **Realtime WebSocket disconnect/reconnect** — client disconnects, hub removes client, client reconnects | realtime | ✅ realtime_test.go | E2E |
| SC-e77-P3-16 | **Proxy Caddy allow endpoint** — registered domain gets allowed, unregistered gets denied | proxy | ✅ hosts_test.go | E2E |

---

## 3. Test Level Distribution

| Test Level | Count | Rationale |
|---|---|---|
| **E2E** (Playwright HTTP) | 48 | Most gaps are at the HTTP layer — verifying routes are wired, middleware fires, serialization is correct |
| **Unit** (Go `httptest`) | 3 | Truly new logic not in existing unit tests: storage file size limit, MCP invalid tool, collection pagination boundary |
| **Integration** (Go + real DB) | 0 | Every component already has integration-level `httptest` tests |

**Principle:** Do not rewrite existing unit tests as E2E. The E2E suite validates the *integration of the full HTTP stack* — route wiring, middleware chaining, CORS, serialization, cross-component interaction. One E2E smoke test per component + one per critical flow catches 90% of integration bugs.

---

## 4. Test Isolation Strategy

| Aspect | Approach |
|---|---|
| **Server** | Playwright config starts one `go run . serve` per session (reuse existing). Fresh `--db :memory:` per run. |
| **Data isolation** | Each test uses unique email (`e2e-${uuid}@test.com`). Tests clean up created resources in `afterAll`. |
| **Parallelism** | Tests within one file run serially (shared token). Separate spec files can run in parallel via `--workers=4`. |
| **Auth setup** | Common `createUser()` fixture returns email/password/accessToken/refreshToken/orgId. |
| **Resource setup** | `createSite()`, `createRepo()`, `createDeploy()` fixtures chain off auth fixture. |

### Shared E2E fixtures to create

```typescript
// tests/e2e/fixtures/org.ts
export interface OrgFixture { orgId: string; slug: string; name: string }
export async function createOrg(request: APIRequestContext, token: string, name: string): Promise<OrgFixture>

// tests/e2e/fixtures/deploy.ts
export interface DeployFixture { deployId: string; site: SiteFixture; repo: RepoFixture }
export async function createFullDeployment(request: APIRequestContext, token: string): Promise<DeployFixture>
export async function waitForDeployment(request: APIRequestContext, deployId: string, timeout: number): Promise<string>

// tests/e2e/fixtures/data.ts
export async function createCollection(request: APIRequestContext, token: string, name: string): Promise<void>
export async function insertItem(request: APIRequestContext, token: string, collection: string, data: object): Promise<number>
```

---

## 5. NFR Verification

| NFR | Requirement | Verification |
|---|---|---|
| **Coverage** | Every registered API route hit by at least one E2E scenario | `scripts/verify-e2e-route-coverage.sh` (extracts routes from `mux.HandleFunc`, cross-references E2E requests) |
| **Speed** | E2E suite completes in < 10 min | `time npx playwright test tests/e2e/` |
| **Isolation** | No test depends on state from another test | `npx playwright test --workers=4 2>&1` (must not fail with parallelism) |
| **Flake** | < 5% flake rate across 3 CI runs | `npx playwright test --repeat-each=3 --workers=1 tests/e2e/critical.spec.ts` |

---

## 6. Implementation Plan (by story)

| Story | Scenarios | Component Focus | Effort | Depends On |
|---|---|---|---|---|
| **e77s01** — Auth & Org E2E | SC-P0-01–03, SC-P1-12–13 | auth/org, CORS, cookie | 4 BCPs | auth fixture |
| **e77s02** — API Data Layer E2E | SC-P0-04–05, SC-P1-16/18, SC-P3-01–02 | collections, SQL, env vars, scaffold | 3 BCPs | auth fixture |
| **e77s03** — Deploy Pipeline E2E | SC-P0-06–07, SC-P1-05–06, SC-P2-02–04, SC-P3-03–05 | deploy, samples, cache, drain | 5 BCPs | repo + site fixtures |
| **e77s04** — Sites E2E | SC-P1-01–04 | env vars, domains, manifest, auth policy | 3 BCPs | site fixture |
| **e77s05** — Monitoring E2E | SC-P1-07–09/17/20, SC-P2-05–06/14–15, SC-P3-08 | health, metrics, alerts, logs, events, incidents, processes, prometheus | 4 BCPs | auth fixture |
| **e77s06** — Functions E2E | SC-P1-10, SC-P3-06–07 | function CRUD, run, logs, runtime errors, timeout | 3 BCPs | auth fixture |
| **e77s07** — Storage E2E | SC-P0-09, SC-P2-07–08, SC-P3-10 | upload, download, delete, thumbnails, path traversal, size limit | 2 BCPs | auth fixture |
| **e77s08** — Messaging E2E | SC-P1-11, SC-P2-18–19, SC-P3-09 | email, sms, push, telegram, history, validation | 2 BCPs | auth fixture |
| **e77s09** — Forge + GitHub E2E | SC-P0-12, SC-P2-09–11, SC-P3-11 | issues, board, wiki, comments, install, repos | 3 BCPs | auth fixture |
| **e77s10** — CI/CD + Git E2E | SC-P1-12–13, SC-P3-13 | workflows, runs, logs, repo CRUD | 2 BCPs | auth fixture |
| **e77s11** — Real-time + Webhooks + Backup E2E | SC-P0-10, SC-P1-14–15, SC-P3-12/15 | websocket, webhook CRUD, backup/restore | 2 BCPs | auth fixture |
| **e77s12** — MCP + Proxy E2E | SC-P0-11, SC-P2-20, SC-P3-14/16 | MCP tools, version, caddy-allow | 2 BCPs | auth + site fixtures |
| **e77s13** — Cross-component orchestration E2E | SC-P0-03 | org isolation, cross-org key access denied | 1 BCP | org fixture |

**Total:** 36 BCPs across 13 stories.

---

## 7. File Map

New E2E spec files to create under `tests/e2e/`:

| File | Stories | Est. Tests |
|---|---|---|
| `tests/e2e/auth-org.spec.ts` | e77s01 | 8 |
| `tests/e2e/api-data.spec.ts` | e77s02 | 7 |
| `tests/e2e/deploy-pipeline.spec.ts` | e77s03 | 10 |
| `tests/e2e/sites-management.spec.ts` | e77s04 | 6 |
| `tests/e2e/monitoring.spec.ts` | e77s05 | 10 |
| `tests/e2e/functions.spec.ts` | e77s06 | 6 |
| `tests/e2e/storage.spec.ts` | e77s07 | 5 |
| `tests/e2e/messaging.spec.ts` | e77s08 | 5 |
| `tests/e2e/forge-github.spec.ts` | e77s09 | 6 |
| `tests/e2e/cici-git.spec.ts` | e77s10 | 4 |
| `tests/e2e/realtime-webhooks.spec.ts` | e77s11 | 4 |
| `tests/e2e/mcp-tools.spec.ts` | e77s12 | 4 |
| `tests/e2e/cross-org-isolation.spec.ts` | e77s13 | 2 |
| `tests/e2e/fixtures/org.ts` | all | N/A |
| `tests/e2e/fixtures/deploy.ts` | e77s03 | N/A |
| `tests/e2e/fixtures/data.ts` | e77s02 | N/A |

Existing `tests/e2e/` files to augment:
- `tests/e2e/login.spec.ts` — no changes (covers basic auth)
- `tests/e2e/sites.spec.ts` — no changes (covers basic sites CRUD)
- `tests/e2e/health.spec.ts` — no changes (covers /health)

---

## 8. Out of Scope

- **Admin-only pages** — no admin role exists in E2E setup
- **WebSocket message content testing** — deferred (requires event emission fixture)
- **Performance / load testing** — covered by Go benchmarks, not E2E
- **Visual regression / screenshot diffing** — deferred to separate epic
- **Mobile / responsive viewports** — deferred
- **Third-party integrations** (GitHub API, Telegram API) — mocked in unit, skip in E2E
- **Multi-instance / HA testing** — out of scope for E2E suite
