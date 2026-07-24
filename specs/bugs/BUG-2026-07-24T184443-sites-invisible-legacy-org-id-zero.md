---
bug_id: BUG-2026-07-24T184443
status: resolved
severity: critical
scope: sites,security
title: "Sites UI shows only 1 site — legacy sites (org_id=0) permanently locked out by BUG-136's org scoping"
github_issue: null
---

# BUG-2026-07-24T184443: Sites UI shows only "exames" — 8 other production sites invisible/inaccessible

## Problem

**Actual:** The Sites page (`/api/sites` → UI) shows exactly one site ("exames"). All previously
existing sites — including `grimoire`, `library`, `astrobiologia`, `bolao`, `docklocker`,
`cleaninstallguide`, `add-tutorial-requests-site`, and the *original* `exames` deployment — are
gone from the list, and are also inaccessible directly (get/delete/redeploy/manifest/auth-policy/
domains/env-vars/logs all now 404 "site not found" for these sites too).

**Expected:** All of the user's own sites are visible and manageable in the UI, regardless of when
they were created.

**Reported:** User attached a screenshot of the Sites page showing only "exames", and noted the
`grimoire` site_id referenced in `.envrc` (`d87f8a0ca3f7c2623a82d940f2b23798`) no longer resolves
— but that ID doesn't even match any row in the database (see Isolate below), so `.envrc` itself
was already stale from an earlier grimoire redeploy, unrelated to this bug.

**Security impact:** NONE — this is an over-restrictive access bug (legitimate owners locked out
of their own resources), not unauthorized access. No exploit path; the opposite of BUG-136.

## Root Cause Analysis

### Reproduce

Verified directly against the production database via SSH (`ssh root@89.116.26.187`,
`sqlite3 /opt/bigbase/data/bigbase.db`):

```
sqlite> SELECT id, name, org_id, created_at FROM sites;
6e33f15519f3ebf670248ab077a0dcfe|add-tutorial-requests-site|0|2026-06-03T15:59:23Z
3e45f95a46e72494ff6ce2a2e3fd78c8|cleaninstallguide|0|2026-06-05T19:04:51Z
c8ca4aef01af7f757c47e0a49fb185a0|docklocker|0|2026-06-12T21:04:37Z
7cdd6280e4a5ca9a99255247fbb7cf91|bolao|0|2026-06-20T14:30:22Z
21fc7f8853fae5625358bb7584e4e45f|library|0|2026-07-07T00:18:46Z
63f73391c4d1c414c488affdba33ce82|astrobiologia|0|2026-07-09T04:57:28Z
c81e929d6aa6a052e4ba0193d28f4a9c|grimoire|0|2026-07-13T01:30:50Z
e7dd8d5f0ff949b9450de300f7c9ff02|exames|0|2026-07-24T17:36:34Z   <- original exames
87049ff64fc45aa78fc2d9246b28c72b|exames|1|2026-07-24T21:11:43Z   <- new duplicate, today
```

**All 9 site rows are still present. Zero rows were deleted — this is not data loss.** The `orgs`
table confirms `org_id=1` is `danielvm@gmail.com` (the only real user). Every legacy site has
`org_id=0`; only the newest "exames" row (created today, after redeploying per the earlier
pnpm-fix work) has `org_id=1`.

### Isolate

`components/sites/sites.go`:

- `ListSites(ctx)` (used by both `GET /api/sites` and the MCP `list_sites` tool): when an
  authenticated org_id is present, queries `WHERE s.org_id = ?` with **no fallback for legacy rows**.
- `requireSiteOwnership(ctx, w, siteID)` (the shared gate used by `getSite`, `deleteSite`,
  `redeploySite`, manifest get/save, auth-policy get/set, domains, env-vars, request logs — i.e.
  every single-site handler): `if dbOrgID != orgID { 404 }`, again with no legacy fallback.

Any site with `org_id = 0` (the column's `DEFAULT 0`, i.e. every site created before org scoping
existed) can never match `orgID = ?` for `orgID > 0` — a real authenticated user can never equal
0. So once the caller has a real org_id, these two functions collectively block every one of
`ListSites`/`GetSite`/`DeleteSite`/`RedeploySite`/manifest/auth-policy/domains/env-vars/logs for
every legacy site.

### Hypothesize

This is a completeness gap in the BUG-136 IDOR fix (commit `5527bebf2`, "fix(security): add
org_id multi-tenant isolation to all site endpoints", merged 2026-07-23 — the day before this
session, **not** caused by this session's PR #156/#158, which only touched
`tests/contract/contract_test.go`, `components/deploy/sitekey_auth_test.go`, and
`.github/workflows/ci-cd.yml`). The equivalent fix in the deploy component
(`components/deploy/gateway.go: HandleList`, commit `c69ccf4ef`) correctly included a legacy-data
carve-out: `WHERE s.org_id = ? OR d.site_id = '' OR d.site_id IS NULL`. The sites component's
`ListSites`/`requireSiteOwnership` fix omitted the analogous carve-out for `org_id = 0`.

Compounding this: the BUG-136 bug file's Resolution claims "8 new IDOR isolation tests verify
cross-org access is denied," but the actual commit diff
(`git show 5527bebf2 -- components/sites/sites_test.go`) only retrofits **existing** tests to
attach `org_id=1` context (via a new `authedRequestSite` helper) so they keep passing — it adds
**zero** new test functions that exercise two *different* org_ids, and zero coverage for the
legacy `org_id=0` scenario. The regression shipped without a test that would have caught it.

### Verify

- Confirmed via direct SQLite query on the production DB (above): all legacy sites have
  `org_id = 0`; only today's new "exames" row has `org_id = 1`.
- Confirmed `ListSites()` and `requireSiteOwnership()` are the only two org-scoping choke points
  in `components/sites/sites.go` (`grep requireSiteOwnership` — 8 call sites, all site-scoped
  handlers route through it; `domains.go` and `env_vars.go` also call the same shared helper).
- Confirmed the MCP `list_sites`/`get_site` tools delegate to the same `Sites.ListSites`/`GetSite`
  methods (`components/mcp/mcp.go`), so fixing the two functions fixes the API, UI, and MCP paths
  together.
- Confirmed `git_repos`/`deployments` rows for these sites are untouched — e.g. `grimoire`'s
  deployment history is intact in the `deployments` table (its most recent deployments show
  `status='failed'`/`'stopped'` from 2026-07-13, a separate pre-existing issue unrelated to
  today's visibility bug).

**Risk level:** High business-impact (total loss of self-service access to 8 of 9 production
sites via the app), but zero data-integrity risk (no rows deleted, straightforward to reverse) and
zero *new* security exposure (strictly more restrictive than before, not less).

## TDD Fix Plan

1. **RED:** `TestListSites_IncludesLegacyOrglessSites` — seed one site with `org_id=1` (caller's
   org) and one with `org_id=0` (legacy/unscoped); assert `GET /api/sites` for org 1 returns both.
   **GREEN:** `ListSites()`'s org-scoped branch becomes `WHERE s.org_id = ? OR s.org_id = 0`.
   **verify:** `go test ./components/sites/... -run TestListSites_IncludesLegacyOrglessSites -v`

2. **RED:** `TestListSites_StillIsolatesRealCrossOrgSites` — seed sites for org 1 and org 2 (both
   nonzero); assert org 1's request does **not** see org 2's site. (Regression guard: the fix must
   not reopen BUG-136 for actual multi-tenant data — only `org_id=0` rows get the carve-out.)
   **GREEN:** No code change if step 1's `OR s.org_id = 0` is scoped correctly; test locks in the
   boundary.
   **verify:** `go test ./components/sites/... -run TestListSites_StillIsolatesRealCrossOrgSites -v`

3. **RED:** `TestRequireSiteOwnership_AllowsLegacyOrglessSite` — a site with `org_id=0`; any
   authenticated org can `GET`/manage it (no 404).
   **GREEN:** `requireSiteOwnership()`'s check becomes `if dbOrgID != orgID && dbOrgID != 0 { 404 }`.
   **verify:** `go test ./components/sites/... -run TestRequireSiteOwnership -v`

4. **RED:** `TestRequireSiteOwnership_StillDeniesRealCrossOrgSite` — a site with `org_id=2`;
   caller org_id=1 still gets 404. (Regression guard, same rationale as step 2.)
   **GREEN:** No code change if step 3 is scoped correctly; test locks in the boundary.
   **verify:** `go test ./components/sites/... -run TestRequireSiteOwnership -v`

**REFACTOR:** None needed — both changes are single-line condition edits to existing functions.

## Acceptance Criteria

- [x] `GET /api/sites` (and MCP `list_sites`) returns legacy (`org_id=0`) sites alongside the
      caller's own org sites
- [x] Get/delete/redeploy/manifest/auth-policy/domains/env-vars/logs all work for legacy sites
- [x] Real cross-org isolation (nonzero `org_id` mismatch) is unchanged — still 404
- [x] All 4 new tests pass
- [x] Existing sites/domains/env_vars test suites still pass
- [x] Full suite green: `go test ./... -count=1`
- [x] Verified against production data after deploy: all 9 sites now have `org_id=1` (confirmed
      via `ssh root@89.116.26.187 sqlite3 /opt/bigbase/data/bigbase.db`), service healthy

## Resolution

**Fixed:** 2026-07-24

**Root cause confirmed:** `ListSites()` and `requireSiteOwnership()` in `components/sites/sites.go`
filtered strictly by `org_id = ?` with no fallback for legacy rows (`org_id = 0`, the column's
`DEFAULT`), added by the BUG-136 IDOR fix (`5527bebf2`) without the equivalent legacy-data
carve-out that the deploy component's analogous fix already had.

**Fix applied:**
1. `ListSites()`'s org-scoped query: `WHERE s.org_id = ?` → `WHERE s.org_id = ? OR s.org_id = 0`.
2. `requireSiteOwnership()`'s check: `if dbOrgID != orgID` → `if dbOrgID != orgID && dbOrgID != 0`.
3. A startup migration in `Sites.Start()` additionally reassigns any `org_id = 0` site to the
   admin user's `default_org_id` (one-time, idempotent — `WHERE org_id = 0`), so legacy sites get
   proper real ownership going forward rather than remaining permanently "unclaimed"; (1) and (2)
   remain as defense-in-depth for any future orphaned rows (e.g. site-deploy-key-created sites).
4. Added 4 regression tests: two proving legacy (`org_id=0`) sites are visible/manageable, two
   locking in that genuine cross-org isolation (`org_id` mismatch between two *real*, nonzero
   orgs) is unchanged — the fix does not reopen BUG-136.

**Evidence:**
- `go test ./components/sites/... -run 'TestListSites_IncludesLegacyOrglessSites|TestListSites_StillIsolatesRealCrossOrgSites|TestRequireSiteOwnership_AllowsLegacyOrglessSite|TestRequireSiteOwnership_StillDeniesRealCrossOrgSite' -v -count=1` — all 4 pass
- `go test ./... -count=1` — full suite green
- `go vet ./...` — clean
- `go build ./...` — clean
- Verified against the live production database (SSH) before and after: all 9 site rows
  confirmed intact throughout (zero data loss)
