---
bug_id: BUG-2026-06-12T120000-site-name-ignored-for-subdomain
status: fixed
severity: medium
scope: deploy, sites
title: Site name ignored when generating deployment subdomain
---

# BUG-2026-06-12T120000: Site name ignored when generating deployment subdomain

## Problem

When creating a site through the wizard, the user can enter a custom site name (e.g. "docklocker"). However, the actual subdomain assigned to the deployment is derived from the **git repository name** (e.g. `big-dock-locker-site` → `big-dock-locker-site.bigbase.click`) rather than from the user-supplied site name.

- **Actual behavior**: subdomain uses the git repo name
- **Expected behavior**: subdomain uses the site name the user typed
- **Steps to reproduce**: Create a site from GitHub, enter a custom site name on the Configure step, deploy — observe the deployment URL uses the repo name, not the site name

## Root Cause Analysis

The `DeployTrigger` interface (`func(ctx, repoID, branch) *Deployment`) accepted by the Sites component does not carry a site name. When the Deploy component's `Trigger` function is invoked, it fetches the repo's name directly from the `git_repos` table and passes it to all subdomain-generating helpers (`deploymentURL`, `deploymentHost`, `finalizeDeploymentURL`, `serveStatic`, `startApp`).

The user-supplied `name` field is stored in the `sites` table but is never forwarded to the deployment layer — so the subdomain is always derived from the repository name regardless of what the user typed.

Risk level: **Medium** — functional regression, affects every site created from GitHub. Existing sites already deployed keep their old (repo-name-based) URL.

## TDD Fix Plan

1. **RED**: Write an integration test that creates a site with a custom name different from the repo name, triggers a deployment, and asserts the deployment URL contains the site name slug, not the repo name slug.
   **GREEN**: Change the `DeployTrigger` type signature to `func(ctx, repoID, branch, siteName string) (*Deployment, error)`. In `Deploy.Trigger`, accept `siteName`; if non-empty use it as the subdomain label instead of `repoName`. Update `Sites.createSite` to pass `req.Name` as the fourth argument.
   **verify**: `go test ./components/deploy/... ./components/sites/...`

2. **RED**: Write a unit test for `Deploy.Trigger` that passes an empty `siteName` and asserts the URL still falls back to the repo name (backwards compatibility).
   **GREEN**: Add the fallback: `if siteName == "" { siteName = repoName }` inside `Trigger` before calling `deploymentURL`.
   **verify**: `go test ./components/deploy/...`

3. **RED**: Write a UI test (or extend existing format tests) asserting that the URL preview on the Configure step reflects changes to the site name input in real time.
   **GREEN**: Confirm `siteDisplayUrl(name)` is already called with the reactive `name` state — no UI change needed here, only verify the test passes.
   **verify**: `cd ui && npm test -- --run`

**REFACTOR**: Update the `DeployTrigger` type alias in `components/sites/sites.go` to match the new signature and ensure all callers (tests, integration wiring in `main.go`) are updated.

## Acceptance Criteria

- [x] Deployment URL uses the site name slug, not the repo name slug
- [x] Empty site name falls back gracefully to the repo name (no regression)
- [x] URL preview on Configure step reflects the typed site name in real time
- [x] All new tests pass
- [x] Existing tests still pass (`go test ./...` green)

## Resolution

Fixed by updating the `DeployTrigger` signature to include `siteName`. 
- Updated `components/sites/sites.go` to pass the site name during creation and redeployment.
- Updated `components/deploy/deploy.go` to use the provided `siteName` for subdomain generation, falling back to `repoName` if empty.
- Verified with new integration tests in `sites_test.go` and unit tests in `deploy_test.go`.
- Verified UI preview behavior with existing `format.test.ts`.
- Updated wiring in `main.go` and other call sites.

