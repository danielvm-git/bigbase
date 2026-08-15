# Phase 1 Re-Audit — e75 / e39 / e40 + Deferred Bugs

**Date:** 2026-08-13
**Trigger:** "Deep re-audit first" before reconciling stale status markers (the roadmap
assumed these were open coding work; the code was already merged to `main`).
**Baseline:** `main` @ `5ac4c51d3` (v2.89.0).
**Verdict:** **PASS across the board.** Every story is implemented and green; the tracking
metadata was stale. No production `.go`/UI code changes were required.

## Verification evidence

| Item | Acceptance criterion | Result |
|------|----------------------|--------|
| **e75s01** FIFO log eviction | Deterministic FIFO (not random map iteration); `TestDeployLogFIFOEviction` asserts oldest-first | PASS — `logs.go` uses an `order []string` slice + mutex; `TestDeployLogFIFOEviction` (logs_test.go:7) green |
| **e75s02** three flaky tests | 0 failures across 10 consecutive runs (epic hard gate) | PASS — `TestDeployStopShutsDownStaticServer\|TestConnectionDrainTimeout\|TestRedeployReplacesPrevious` `-count=10` → `ok` (70.7s), 0 FAIL |
| **e39s03** terminal log viewer | `TerminalLogViewer` wired into `SiteDetailPage`; `BuildLogs` removed from index barrel (file retained) | PASS — exported at `ui/src/components/index.ts:21`; `BuildLogs` absent from index. Accepted structural divergence: toolbar state delegated to `StreamLog` (functionally complete) |
| **e40s02** init CLI + `--manifest` | `runInitCmd` + `InitManifest`; `deploy --manifest` flag parsed and forwarded | PASS — `main.go:722` defines the flag, `main.go:745` forwards it as `manifest_path` |
| **Full Go suite** | `go test ./...` green | PASS — 0 failures, all packages `ok` (the "2 pre-existing failures" in state.yaml do not reproduce) |
| **UI suite** | vitest under jsdom green | PASS — 93 files / 736 tests pass, 0 failures |
| **Build/vet** | clean | PASS — `go build ./...` + `go vet ./...` clean |

## Deferred bugs (Phase 0)

| Bug | Original framing | Finding | New status |
|-----|------------------|---------|-----------|
| BUG-2026-07-10T160006 | `WithProjectID` has zero production callers | Function was **deleted** in `e673ed0de` (v2.76.16) as dead code; `WithSiteID` is the active scope key (`tech-stack.md:170`). Nothing to fix. | `resolved` |
| BUG-2026-07-10T160111 | JWT lacks `project_id` claim | Adding it standalone = dead field (the `WithProjectID` plumbing it needed was removed); `org_id` already carries the tenant boundary. YAGNI. | `wontfix` |

## Actions taken from this audit

- Reconciled `specs/bugs/registry.yaml` (both entries above, with evidence).
- e75 epic + both stories `todo → done`; `e39s03`/`e40s02` tasks `planned → done`.
- No GAPs found → no `develop-tdd` remediation tasks were required.
- Separately (not from a GAP): the `Deploy` workflow was red on `main` because the e89
  release made `BIGBASE_ROOT_ENCRYPTION_KEY` mandatory but it was never provisioned —
  fixed under Phase 1 CI/CD hardening.

## Phase 1 CI/CD hardening (2026-08-13 → 2026-08-15)

- **Production fix:** provisioned `BIGBASE_ROOT_ENCRYPTION_KEY` (repo secret + `deploy.yml`
  VPS-env wiring). `Deploy` green; `https://bigbase.click/api/monitoring/health` → `{"status":"ok"}`.
- **UI vitest in CI:** added a `test-ui` job (jsdom) that gates release — the Admin UI was
  previously only *built*, never tested. Later extended by peers with eslint/typecheck/build.
- **Lockfile-sync guard:** `npm ci` drift check in `verify` (the `@ctxo/plugin-api` failure class).
- **Node pinned to 24** across all jobs via `NODE_VERSION`.

## e81s06 security regression tests — 6/6 complete

Regression tests guarding already-fixed vulns so they cannot silently return. No production
code changed; each asserts an existing guard. Initially 4/6 landed with 2 documented gaps;
both gaps were subsequently closed with real behavioral tests (2026-08-15).

| # | Test | Location | Guards |
|---|------|----------|--------|
| 1 | `TestPathTraversal` | `components/git/path_traversal_regression_test.go` | `deleteRepo` rejects `/`- and `..`-containing repo ids with 400 before any FS `RemoveAll` (seeds rows the public API can't mint) |
| 2 | `TestCommandInjection` | `components/deploy/security_regression_test.go` | `cloneAndCheckout`'s `--` separator treats a `--upload-pack=<cmd>` payload as a nonexistent path — injected command never runs (sentinel absent) |
| 3 | `TestAnonymousWriteBlocked` | `components/auth/anon_write_regression_test.go` | anonymous JWTs get 403 on POST/PUT/PATCH/DELETE; 200 on GET/HEAD/OPTIONS |
| 4 | `TestAppTypeIsValid` | `components/deploy/security_regression_test.go` | `AppType.IsValid()` / `AllAppTypes()` stay in sync; unknown types rejected |
| 5 | `TestErrorSentinels` | `components/deploy/security_regression_test.go` | `ErrRepoNotFound` / `ErrDeploymentNotFound` match via `errors.Is`, not fragile strings |
| 6 | `TestManifestPathTraversal` | `components/deploy/security_regression_test.go` | `LoadManifestPath` rejects absolute and directory-escaping manifest paths |

All six pass in `test-sqlite` / `test-postgres`. Every GitHub Actions workflow green on `main`
(`b5814bdb7`). One unrelated pre-existing flake surfaced during landing —
`TestDrainStatusHistory` under `-race` in full parallel runs (passes in isolation and on rerun);
filed as a separate task, not part of this work.
