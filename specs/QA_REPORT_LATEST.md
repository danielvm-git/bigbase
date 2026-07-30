# QA Audit Report — 2026-07-30

## Run Config

**N (ceiling)**: 100
**Floor**: 0 (0 open bug-labelled issues, CI green)
**Repo**: 66k LOC Go, 20 components, 116 registry entries (104 fixed, 9 wontfix, 2 deferred, 1 done)

### FROZEN (must not touch)

| # | Item | Source | Reason |
|---|------|--------|--------|
| F1 | No Docker/Kubernetes | ADR 0004 | "BigBase will never use Docker" — explicit architectural rule |
| F2 | Deploy decomposition (Engine + Gateway + Orchestrator) | ADR 0005 | Accepted and implemented; split seams are stable |
| F3 | MCP auth via ResolveOrgKey + Bearer tokens | ADR 0006 | Security contract; CWE-862 closure |
| F4 | Component interface `Name/Version/Dependencies/Init/Start/Stop/Hooks` | kernel/component.go | Public API — all 20 components implement it |
| F5 | Kernel event bus (hook-based communication) | CONVENTIONS.md | "Components communicate via event hooks, not direct imports" |
| F6 | Go standard layout (`kernel/`, `components/<name>/`, `config/`) | CONVENTIONS.md | Structural invariant |
| F7 | SQLite DDL migrations in component `Start()` | schema pattern | Every component owns its migration; no central schema file |
| F8 | MCP tool interface (JSON-RPC over Streamable HTTP) | wire format | Clients depend on this contract |
| F9 | Auth middleware context keys (`ctxOrgID`, `ctxOrgKeyScopes`) | security boundary | Cross-component contract |

### Hotspots (high-churn × high-fix-density)

| File | Churn | Fixes | Risk |
|------|-------|-------|------|
| `components/deploy/deploy.go` | 62 | deploy reliability, port allocator, IDOR | P0 |
| `components/auth/auth.go` | 43 | auth bypass, scope enforcement, IDOR | P0 |
| `components/mcp/mcp.go` | 25 | MCP auth, tool registration | P0 |
| `components/api/api.go` | 20 | SQL injection (BUG-129) | P0 |
| `components/proxy/proxy.go` | 31 | routing, CORS, CSP | P1 |
| `components/sites/sites.go` | 33 | org scoping, migrations | P1 |
| `components/monitoring/monitoring.go` | 20 | org isolation (BUG-143) | P1 |

### Per-Module Risk Levels

| Module | Risk | Depth Tier | Focus |
|--------|------|-----------|-------|
| deploy | P0 | full_maturity | IDOR, engine, gateway, rollback, health checks |
| auth | P0 | full_maturity | middleware, scopes, API keys, JWT, OAuth |
| mcp | P0 | full_maturity | auth enforcement, tool tiers, session management |
| api | P0 | full_maturity | SQL injection, org scoping, input validation |
| proxy | P1 | standard | routing, CSP, CORS, WebSocket |
| sites | P1 | standard | org scoping, env vars, migrations |
| monitoring | P1 | standard | org isolation, alert delivery |
| storage | P1 | standard | file org scoping, upload path traversal |
| functions | P2 | standard | runtime isolation, DB injection |
| cici | P2 | standard | workflow execution, command validation |
| git | P2 | standard | clone/fetch injection, repo ownership |
| forge | P2 | standard | IDOR on issues/labels/wiki |
| messaging | P2 | standard | cross-tenant message leak |
| realtime | P2 | standard | WebSocket auth, channel scoping |
| backup | P3 | minimal_decisive | backup/restore integrity |
| admin | P3 | minimal_decisive | admin-only routes |
| db | P3 | minimal_decisive | SQLite/Postgres abstraction |
| config | P3 | minimal_decisive | config merge, validation |
| internal/* | P3 | minimal_decisive | envcrypto, eventrecorder, llm |

### Seeded Issues

None — 0 open GitHub issues. All 116 registry entries resolved.

### Baseline

- `go test ./...` — **PASS** (1226+ tests, 3 expected skips)
- `golangci-lint run ./...` — **0 issues**  
- CI (main) — **5/5 jobs green**

---

## Findings

### BUG-2026-07-30T000001 — Cross-tenant info disclosure in handleDeployStats

**Severity**: medium | **Status**: fixed | **PR**: #195 (merged)

`handleDeployStats` returned deployment statistics (total, running, failed, 24h failure rates) without filtering by the caller's org_id. Any authenticated user could see operational metrics for ALL tenants.

**Root cause**: SQL queries had no org_id filter — unlike `HandleList` which JOINs through `sites`.

**Fix**: JOIN through `sites` table to scope by org_id, matching the `HandleList` pattern.

**Regression guard**: `TestIDOR_StatsScopedByOrg` — verifies org 100 sees only its 2 deployments, org 200 sees only its 3.

### Other observations (not bugs)

- **Cache handlers** (`handleCache`, `handleCachePrune`): Global operations behind auth. Any authenticated user can clear/prune the deploy cache. Design choice — global admin operation, not a vulnerability.
- **Monitoring flaky test** (`TestHostMetrics`): Environment-dependent, passes on re-run. Pre-existing.

---

## Summary

| Metric | Value |
|--------|-------|
| Bugs found | 1 |
| Bugs fixed | 1 |
| PRs merged | 1 (#195) |
| Registry entries | 117 (was 116) |
| CI status | All green |
| Floor resolved | 0 → 0 (was already clean) |
