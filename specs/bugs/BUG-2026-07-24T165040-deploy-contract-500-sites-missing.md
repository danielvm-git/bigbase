---
bug_id: BUG-2026-07-24T165040
status: resolved
severity: high
scope: deploy
title: CI red on main — TestDeployContract 500s, TestSiteKeyMatchingSite CI-only race, and Allure dashboard job broken
---

# BUG-2026-07-24T165040: CI red on main — three distinct causes, all fixed

Investigating `gh run view 30114291380` (CI/CD on `main`, triggered by PR #152) surfaced three
separate, real bugs in our own code/CI config. All three are fixed in this cycle.

## Bug 1: TestDeployContract fails 500 — deploy list org-scope query assumes `sites` table exists

### Problem

**Actual:** `TestDeployContract` (tests/contract/contract_test.go) fails on both sqlite and postgres CI matrix jobs:
```
contract_test.go:212: status code: got 500, want 200
contract_test.go:216: response missing field "data"
```
CI/CD on `main` has been red since commit `60fb5066e` (fix: cross-tenant deployment hijack via missing org_id check), confirmed via `gh run view 30114291380 --log-failed` (workflow CI/CD, triggered by merge of PR #152).

**Expected:** `GET /api/deploy` returns 200 with `{"data": [...]}`.

**Reproduce:** `go test ./tests/contract/... -run TestDeployContract -v -count=1`

**Security impact:** NONE — production topology (main.go) always registers the `sites` component before serving requests, so this never manifests outside of test isolation.

### Root Cause Analysis

**Reproduce:** Confirmed locally with `go test ./tests/contract/...`. Same failure as CI.

**Isolate:** Added temporary debug logging to `components/deploy/gateway.go: HandleList`. Output:
```
DEBUG: took org branch, orgID= 1 ok= true
DEBUG list deployments error: SQL logic error: no such table: sites (1)
```
`tests/contract/contract_test.go`'s `doRequest` helper injects `auth.WithOrgID(ctx, 1)` into every request (added for BUG-136 site multi-tenant isolation tests). This routes `HandleList` into its org-scoped branch, which runs:
```sql
SELECT ... FROM deployments d LEFT JOIN sites s ON d.site_id = s.id WHERE s.org_id = ? OR ...
```
`TestDeployContract` registers only `db`, `git`, `deploy` (matching `deploy.Dependencies() = ["db", "git"]`) — it never registers the `sites` component, so the `sites` table doesn't exist and the JOIN fails.

**Hypothesize:** The org-scoped deploy IDOR fix (`c69ccf4ef`, BUG-2026-07-23-idor-deploy-lifecycle) added a hard dependency on the `sites` table without declaring it, and without a corresponding test fixture update in `tests/contract/contract_test.go` (unlike `components/deploy/idor_test.go`, which already creates a `sites` table via `setupIDORTest` for exactly this reason).

**Verify:**
- `main.go` registers the `sites` component (`s`) and `deploy` component (`depComp`) together; `kernel.Start()` runs every registered component's `Start()` to completion, sequentially, before the HTTP server accepts requests — so `sites` is always migrated before any request reaches `HandleList` in production. Confirmed no production risk.
- `components/deploy/idor_test.go` already establishes the convention: any deploy test exercising the org-scoped path must provide a `sites` table/component.
- `tests/contract/contract_test.go`'s `TestSitesContract` shows the correct pattern: construct the real `sites.New(...)` component and register it with the kernel.

**Risk level:** Low (test-only gap; no production code path affected)

### Fix

Register `sites.New(sites.Options{DB: dd, Logger: testLogger{}})` alongside `deploy` in `TestDeployContract`, matching production wiring (`main.go`) and the existing `TestSitesContract`/`idor_test.go` conventions.

**Verify:** `go test ./tests/contract/... -run TestDeployContract -v -count=1` — 1 passed

## Bug 2: TestSiteKeyMatchingSite — CI-only flake from an unsynchronized background goroutine racing TempDir cleanup

### Problem

**Actual:** `TestSiteKeyMatchingSite` (components/deploy/sitekey_auth_test.go) intermittently fails in CI (not locally):
```
--- FAIL: TestSiteKeyMatchingSite (0.04s)
    testing.go:1464: TempDir RemoveAll cleanup: unlinkat /tmp/TestSiteKeyMatchingSite.../.git: directory not empty
```
Seen in multiple unrelated CI runs (`30056577647`, `30058206507`, `30092357441`, and again on this fix's own PR run `30122049219`) — a real, pre-existing flake, not something either fix introduced.

**Security impact:** NONE — test-only.

### Root Cause Analysis

`dep.HandleCreate(w, req)` returns 201 synchronously, but the actual deployment work runs in a
detached goroutine: `components/deploy/engine.go: Trigger()` does `go d.runDeployment(deploy, buildDir, siteName)`
and returns immediately. `TestSiteKeyMatchingSite` asserted on the 201 and returned without
waiting for that goroutine to finish. When the test function returns, `t.TempDir()`'s registered
cleanup (`os.RemoveAll` on `gitDir`/`buildsDir`) races the still-running goroutine, which is
concurrently writing into the same `.git` directory (via `createTestRepo`'s `gitDir`) — `RemoveAll`
can observe a directory as empty via `readdir`, then have a new entry appear before the `rmdir`,
producing "directory not empty". This is a timing race, so it reproduces far more often on CI's
slower/contended disks than on a local dev machine (which is why `go test ./components/deploy/...`
always passed locally here, even run standalone and repeatedly).

Every other deploy test that calls a create-deployment endpoint already follows up with the
existing `waitForDeploymentTerminal(t, handler, depID, timeout)` helper (used ~20 times across
`deploy_test.go`, `drain_test.go`, `rollback_test.go`, etc.) specifically to avoid this race.
`TestSiteKeyMatchingSite` was the one outlier missing it.

**Risk level:** Low (test-only race; no production code path affected — production deployments
aren't torn down mid-flight by a test harness)

### Fix

`components/deploy/sitekey_auth_test.go: TestSiteKeyMatchingSite` now decodes the created
deployment's `id` from the response body and calls `waitForDeploymentTerminal(t, handler, depID,
10*time.Second)` before returning, matching the established convention.

**Verify:** `go test ./components/deploy/... -run TestSiteKey -v -count=1` — both pass, ~1.9s
(previously 0.04s, confirming the goroutine is now actually awaited)

## Bug 3: "Generate Progress Dashboard" CI job — third-party Allure Docker action broken

### Problem

**Actual:** The `report` job's "Generate Allure Report" step (`simple-elf/allure-report-action@e463a472d...`)
fails on every run, including the original reported run `30114291380`:
```
generating report from allure-results to allure-report ...
xargs is not available
copy allure-report to allure-history/89
cp: cannot stat './allure-report/.': No such file or directory
```

### Root Cause Analysis

Fetched the action's source (`opensrc path simple-elf/allure-report-action`) and read its
`entrypoint.sh`/`Dockerfile`. The action builds a fresh Docker image on every run
(`yum -y update && yum -y install tar wget gzip findutils` inside an Amazon Corretto 8 base
image) and shells out to the Allure Java commandline tool. "xargs is not available" is printed
by the `allure generate` step itself; the report directory is never created afterward, so the
action's own subsequent `cp` calls fail and the container exits non-zero. Because the Dockerfile
is rebuilt from the floating `yum` repos on every CI run (only the action's own script content is
pinned by SHA, not the OS package versions inside its image), this is an inherently unstable
dependency, not a config mistake on our side we can fix by pinning a different SHA.

**Risk level:** Low (reporting/dashboard job only — doesn't affect Go binary correctness), but it
does keep the `report` job permanently red, which is noise `main` doesn't need.

### Fix

Replaced the Docker-based `simple-elf/allure-report-action` step in `.github/workflows/ci-cd.yml`
with an explicit `run:` step that ports the same `entrypoint.sh` logic (history rotation,
`executor.json`, redirect `index.html`, history copy) directly onto the `ubuntu-latest` runner,
and generates the report via `npx allure-commandline@2.32.0` (pinned npm package, not a floating
yum-built Docker image) on top of explicit `actions/setup-java@v4` (temurin 17) and
`actions/setup-node@v4` (20) steps — matching how Go/Python are already pinned explicitly
elsewhere in this same workflow.

**Verify:**
- `bash -n` syntax-checked the extracted script
- Ran the full script locally with fake `GITHUB_*` env vars against a sample `allure-results/`
  fixture — history rotation, `index.html`, and `executor.json` all produced correctly
- Ran the actual `npx allure-commandline@2.32.0 generate` step inside a `node:20-bookworm` Docker
  container with a JRE installed, confirming it produces `allure-report/` (including
  `allure-report/history/`) with the exact structure the rest of the script's `cp` commands expect
- Full end-to-end dry run (generate → copy into `allure-history/<run>/` and `.../last-history/`)
  verified locally
- Final confirmation: PR CI run for this branch

## Acceptance Criteria

- [x] `TestDeployContract` passes locally and mirrors production component topology
- [x] `TestSiteKeyMatchingSite` no longer races TempDir cleanup
- [x] `.github/workflows/ci-cd.yml`'s Allure report step no longer depends on the broken
      `simple-elf/allure-report-action` Docker image
- [x] Full suite green: `go test ./... -count=1`
- [x] `go vet ./...` clean
- [x] `go build ./...` clean
- [x] CI green end-to-end on PR #156 (Test sqlite/postgres, lint, verify, Generate Progress
      Dashboard, CodeQL, Snyk — all pass)

## Resolution

**Fixed:** 2026-07-24

**Evidence:**
- `go test ./tests/contract/... -run TestDeployContract -v -count=1` — 1 passed
- `go test ./components/deploy/... -run TestSiteKey -v -count=1` — 2 passed
- `go test ./... -count=1` — 1159 passed, 0 failed
- `go vet ./...` — clean
- `go build ./...` — clean
- `python3 -c "import yaml; yaml.safe_load(...)"` on `ci-cd.yml` — valid YAML
- Merged: PR [#156](https://github.com/danielvm-git/bigbase/pull/156), squash-merged to `main` 2026-07-24T20:24:03Z, all CI checks green
