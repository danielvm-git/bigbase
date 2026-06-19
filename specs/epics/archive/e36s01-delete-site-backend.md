# Story e36s01: Backend DELETE /api/sites/:id with safe cascade cleanup

**type:** feat
**context:** domain
**epic:** e36 — Delete Deployed Site
**bcps:** 3
**status:** done
**wsjf:** 6.0 (BV=8 TC=5 RR=5 / JS=3)
**scope:** `components/sites/`

## 1. User need

Admins need a backend API to delete stale Sites from the console so the Sites grid can stop showing accidental or obsolete deployments.

## 2. Outcome

`DELETE /api/sites/:id` deletes a site record and site-owned terminal data, returns `204 No Content`, and never deletes the source Git repository.

## 3. Module purpose

The Sites component owns the Sites admin API: listing sites, creating Git-backed sites, showing site details, redeploying a site, exposing site request logs, and delegating custom-domain subroutes.

## 4. Callers

- `main.go` mounts `sites.Handler()` at authenticated `/api/sites` and `/api/sites/` routes.
- `ui/src/lib/sitesData.ts` calls `/api/sites` and `/api/sites/:id`.
- `ui/src/pages/DeployPage.tsx` renders the Sites grid from `getSites()`.
- `ui/src/pages/SiteDetailPage.tsx` calls `/api/sites/:id`, `/api/sites/:id/deploy`, and `/api/sites/:id/logs`.
- `components/sites/domains.go` handles `/api/sites/:id/domains` subpaths.
- `components/deploy/deploy.go` owns deployment processes/hosts/log writes and the `deployments` table.

## 5. Contracts to preserve

- `GET /api/sites` returns `{data: Site[]}`.
- `POST /api/sites` validates `name` and `git_repo_id`; on success optionally triggers deploy.
- `GET /api/sites/:id` accepts both `sites.id` and legacy `git_repo_id` aliases.
- `/api/sites/:id/deploy`, `/api/sites/:id/logs`, and `/api/sites/:id/domains` keep their current behavior.
- API clients receive generic errors; detailed DB errors stay in logs.
- Deleting a Site must not delete the underlying `git_repos` row.

## 6. Prior art

- `components/deploy/deploy.go` already implements `DELETE /api/deploy/:id` with 404/409/204 behavior.
- `components/git/git.go` and `components/webhooks/webhooks.go` use direct `DELETE FROM ... WHERE id = ?` patterns.
- `components/sites/domains_test.go` has helper setup for Sites route tests.

## 7. Impact assessment

See `specs/IMPACT.md`.

Risk is **High** because this is a destructive shared API and crosses Sites, Deploy, Domains, and Request Logs data.

## 8. Interpretation decision

This story implements **DB-safe site deletion** only. It must reject active deployments (`pending`, `building`, `running`) with `409 Conflict` because the Sites component cannot safely terminate Deploy-owned processes or unregister host routes by direct import. Running deployment cleanup is handled by follow-up story `e36s02`.

## 9. Acceptance Criteria (§17)

### AC1: missing site returns 404
**Given** no site exists for the requested id
**When** `DELETE /api/sites/missing` is called
**Then** the API returns `404` and `{ "error": "site not found" }`.

### AC2: active deployment guard
**Given** a site has a deployment with status `pending`, `building`, or `running`
**When** `DELETE /api/sites/:id` is called
**Then** the API returns `409` and does not delete the site, domains, logs, deployments, or git repo.

### AC3: terminal site deletion by site id
**Given** a site has only terminal deployments such as `failed`
**When** `DELETE /api/sites/:id` is called with the canonical site id
**Then** the API returns `204`, deletes the site row, deletes `site_domains`, deletes `site_request_logs`, and deletes matching terminal `deployments` rows.

### AC4: legacy alias deletion
**Given** a site is reachable by `git_repo_id` alias
**When** `DELETE /api/sites/:git_repo_id` is called
**Then** the same cascade deletion runs for the canonical `sites.id`.

### AC5: source repository preservation
**Given** the site was created from a row in `git_repos`
**When** the site is deleted
**Then** the `git_repos` row remains available for creating/deploying a new site later.

### AC6: subroutes still work
**Given** existing Sites subroutes
**When** tests for domains, logs, get/list/create, and redeploy route parsing run
**Then** behavior is unchanged except for the new DELETE method.

## 10. Out of scope

- Stopping running app processes.
- Closing static file servers.
- Unregistering deployment host routes.
- UI delete buttons.
- E2E browser coverage.
- Deleting `git_repos`.

## 11. New abstractions

- `siteForDelete` value/helper: canonicalizes `sites.id`, `git_repo_id`, and `name` for deletion.
  - **Reason for Depth:** Avoid duplicating alias resolution between active-deployment checks and cascade deletes.
- `deleteSiteRecords(ctx, siteForDelete)` helper.
  - **Reason for Depth:** Keeps route parsing separate from destructive DB operations and makes cascade behavior testable.

## 12. External packages / Slopcheck

No new external packages. `[OK]` standard library only.

## 13. Data safety rules

- Check active deployments before deleting any row.
- Use parameterized SQL only.
- Delete by canonical `site.id` plus `git_repo_id` fallback for deployments.
- Preserve `git_repos`.
- If any DB operation fails, log internally and return `500 {"error":"internal error"}`.

## 14. Implementation Steps

1. Add RED route/status tests for missing site and active deployment guard → verify: `go test ./components/sites/ -run TestDeleteSite -v`
2. Add RED cascade tests for canonical site id, legacy `git_repo_id` alias, and `git_repos` preservation → verify: `go test ./components/sites/ -run TestDeleteSite -v`
3. Route `DELETE /api/sites/:id` in `handleSiteByID` before the generic method guard → verify: `go test ./components/sites/ -run TestDeleteSiteNotFound -v`
4. Implement site alias resolution helper and active deployment guard for `pending`, `building`, and `running` statuses → verify: `go test ./components/sites/ -run TestDeleteSiteActiveDeploymentConflict -v`
5. Implement idempotent cascade deletes for terminal data (`site_domains`, `site_request_logs`, `deployments`, then `sites`) with generic client errors → verify: `go test ./components/sites/ -run TestDeleteSiteCascade -v`
6. Run focused regression tests for Sites and Deploy deletion contracts → verify: `go test ./components/sites/ ./components/deploy/ -run 'TestDeleteSite|TestCustomDomain|TestDeleteDeployment' -v`

## 15. Verification Script

1. Start from a test database or seeded local dev database.
2. Create a git repo row and site row.
3. Add a domain, request-log row, and failed deployment row for that site.
4. Call `DELETE /api/sites/{site_id}`.
5. Observe `204 No Content`.
6. Confirm `sites`, `site_domains`, `site_request_logs`, and matching `deployments` rows are gone.
7. Confirm the source `git_repos` row still exists.
8. Seed a `running` deployment and confirm deletion returns `409` until `e36s02` provides Deploy cleanup.

## 16. Risks

| Risk | Detection | Mitigation |
|------|-----------|------------|
| Orphan running process | `running` deployment deletion test | Reject `running` with 409 in e36s01; cleanup in e36s02 |
| Over-delete by repo alias | legacy alias tests | Resolve canonical site first; preserve `git_repos` |
| Missing deploy table in isolated Sites tests | test setup | Create minimal deployments table in tests where needed |
| Subroute regression | focused tests | Route DELETE only on root site path, not subpaths |

## 17. Countability

BCP size: 3. The story has six independently verifiable tasks and one focused Go test target.

## 18. Dependencies

- `e15` Sites API exists.
- `e28` deployment deletion semantics exist.
- `components/db` implements `kernel.DBer` but does not expose transactions, so this story uses ordered idempotent deletes rather than a transaction.

## 19. Done criteria

- `TestDeleteSite*` exists and passes.
- Existing Sites domain/log tests pass.
- Existing Deploy delete tests pass.
- Spec status can move from `pending` to `done` after implementation and verification.

## 20. Handoff

After plan acceptance, run `kickoff-branch` from `main`, then implement via TDD in `develop-tdd`.
