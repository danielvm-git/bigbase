---
bug_id: BUG-2026-07-06T213500
status: fixed
severity: high
scope: sites,ui
title: Sites list slow to load and missing sites
---

# BUG-2026-07-06T213500: Sites list slow to load and missing sites

## Problem

- **Actual:** The Admin UI Sites page (`/deploy`) takes a long time to load. Users with many sites see only a subset of their sites (e.g. 2 of N).
- **Expected:** All sites load quickly in one request; each card shows its latest deployment status.
- **Reproduce:** Log into production admin at `bigbase.click`, open Sites. With many sites/deployments, page is slow and incomplete.
- **Security impact:** NONE — read-only listing; no auth bypass or data leak.

## Root Cause Analysis

### Reproduce

1. `GET /api/sites` loads every row from `sites`, then for **each** site runs a separate `latestDeployment` query (`SELECT … FROM deployments WHERE repo_id = ? ORDER BY created_at DESC LIMIT 1`).
2. Handler uses a **10-second** context timeout for the entire operation.
3. With N sites, this is **N+1 queries**. As N and deployment history grow, total latency exceeds 10s.
4. When the context deadline expires, `rows.Next()` on the main sites cursor stops early — the handler returns a **truncated** list (verified: Go `database/sql` cancels row iteration on context cancel).

### Isolate

- Frontend (`DeployPage`) renders whatever `/api/sites` returns — no client-side pagination or limit.
- Backend `listSites` in the sites component is the bottleneck; no proxy-side pagination.

### Hypothesize

| # | Hypothesis | Falsification |
|---|------------|---------------|
| 1 | N+1 deployment queries + 10s timeout truncates results | Seed many sites; list returns fewer than seeded count under timeout pressure |
| 2 | Frontend filter hides sites | User on "All" tab; filter only applies branch name |
| 3 | Only 2 rows in DB | Production has more deployed repos than visible cards |

Hypothesis 1 confirmed via code path analysis and Go context cancellation semantics on `rows.Next()`.

### Verify

- **Root cause:** Per-site `latestDeployment` queries in `listSites` / `listSitesFromRepos` cause O(N) DB round-trips; 10s timeout aborts iteration before all sites are collected.
- **Risk level:** High — worsens with every new site/deployment; production admin becomes unusable.

## TDD Fix Plan

1. **RED:** Test that listing sites with 30+ seeded sites returns every site and attaches the latest deployment per repo.
   **GREEN:** Replace per-site `latestDeployment` calls with a single batched query (`latestDeploymentsByRepo`).
   **verify:** `go test ./components/sites/ -run TestSitesListReturnsAllSitesWithDeployments -count=1`

2. **RED:** Test batched lookup picks the newest deployment when a repo has multiple rows.
   **GREEN:** Use `GROUP BY repo_id` + `MAX(created_at)` join in batch query.
   **verify:** `go test ./components/sites/ -run TestLatestDeploymentsByRepo -count=1`

3. **REFACTOR:** Add index on `deployments(repo_id, created_at)` in deploy migrations for subquery performance.

## Acceptance Criteria

- [x] `GET /api/sites` returns all sites regardless of count (within reasonable timeout)
- [x] Latest deployment still attached per site
- [x] New tests pass
- [x] Existing sites tests pass

## Resolution

**Fix applied:** Replaced per-site `latestDeployment` calls in `listSites` and `listSitesFromRepos` with a single batched `latestDeploymentsByRepo` query. Added `idx_deployments_repo_created` index on the deployments table.

**Evidence:**
- `go test ./components/sites/ -run TestSitesListReturnsAllSitesWithDeployments -count=1` — PASS (35 sites, correct latest deployment each)
- `go test ./components/sites/ -count=1` — PASS (51 tests)

**Deploy to production:** Redeploy BigBase binary to `bigbase.click` so `/api/sites` uses the batched query.
