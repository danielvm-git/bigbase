# Impact Assessment — e36s01 Delete Site Backend

## Target

- `components/sites/sites.go`
  - `handleSiteByID`
  - new `deleteSite` flow for `DELETE /api/sites/:id`
  - auxiliary deletion from `sites`, `site_domains`, `site_request_logs`, and `deployments`
- HTTP contract: `/api/sites/:id`

## Dependents (sampled counts)

- `/api/sites` references: 68 across backend, UI, tests, specs, and prototype archives.
- `handleSiteByID` references: 3.
- `site_domains` references: 7.
- `site_request_logs` references: 21.
- `deployments` references in target dirs: 64.

Key callers/dependents:

- `main.go` — mounts authenticated Sites handler at `/api/sites` and `/api/sites/`.
- `ui/src/lib/sitesData.ts` — fetches `/api/sites`, `/api/sites/:id`, and derives deployments through `/api/deploy`.
- `ui/src/pages/DeployPage.tsx` — renders Sites grid from `getSites()`.
- `ui/src/pages/SiteDetailPage.tsx` — loads `/api/sites/:id`, redeploys via `/api/sites/:id/deploy`, and loads request/build logs.
- `components/sites/domains.go` — owns `/api/sites/:id/domains` subroutes and `site_domains` rows.
- `components/deploy/deploy.go` — owns `deployments`, host/process lifecycle, and `site_request_logs` writes.
- `tests/contract/contract_test.go` — contract coverage for GET/POST `/api/sites`.
- `tests/e2e/sites.spec.ts` — API-level E2E coverage for `/api/sites` and `/api/deploy`.

## Affected Stories

- `e15s01` — Sites deploy from GitHub journey; DELETE must not break list/create/get/redeploy behavior.
- `e20` custom domains scope — `site_domains` rows must be deleted with the site.
- `e26` Site Build Logs UI — deployment/log associations must stay coherent after deletion.
- `e27` Site Request Logs — request log rows for a deleted site must be removed.
- `e28s01` — deployment deletion semantics; e36 must not duplicate or regress `DELETE /api/deploy/:id`.
- `e36s02` — follow-up deploy cleanup hook; e36s01 should leave a clear seam and avoid orphaning running deployments before that hook exists.

## Test Coverage

Existing coverage:

- `components/sites/sites_test.go`
  - Create site triggers deploy.
  - Create with custom site name.
  - List request logs.
  - List empty sites.
- `components/sites/domains_test.go`
  - Register/list/verify custom domains.
  - Duplicate-domain 409.
  - Missing-domain 400.
- `components/deploy/deploy_test.go`
  - `TestDeleteDeployment` covers 404, 409 pending/building, and 204 terminal/running deployment deletion.
- `components/deploy/request_logs_test.go`
  - Request log insert/prune/query behavior.
- `tests/contract/contract_test.go`
  - GET/POST `/api/sites` contract.
- `tests/e2e/sites.spec.ts`
  - GET `/api/sites`, POST validation, GET `/api/deploy`.

Gaps to fill before implementation:

- No test for `DELETE /api/sites/:id` route handling.
- No test for cascade deletion of `site_domains`, `site_request_logs`, and terminal `deployments` rows.
- No test that deleting a site preserves the source `git_repos` row.
- No test for legacy lookup by `git_repo_id` alias.
- No test ensuring active deployments are guarded before deploy cleanup exists.

## Risk: High

This touches a shared authenticated API and destructive data path used by Sites, Deploy, Domains, Logs, Dashboard, and future UI deletion flows; missing guards can orphan running processes or delete user data unexpectedly.

## Recommended action

Proceed with TDD. Add focused backend tests first, keep this story limited to safe DB cascade behavior, and explicitly guard active deployments (`pending`, `building`, `running`) until `e36s02` introduces Deploy-owned process/host cleanup.
