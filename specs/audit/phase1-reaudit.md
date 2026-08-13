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
