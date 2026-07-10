# BUG-2026-07-10T192800: TestDeployCustomSiteName cleanup failure on CI — .git dir not empty

## Problem

`TestDeployCustomSiteName` fails in CI with:

```
testing.go:1464: TempDir RemoveAll cleanup: unlinkat /tmp/TestDeployCustomSiteName3006779772/002/.../.git: directory not empty
```

The test fires `POST /api/deploy` which returns HTTP 201 immediately, but the actual build runs asynchronously via `go d.runDeployment(...)`. When the test function returns, Go's `t.TempDir()` cleanup runs `os.RemoveAll` on the `buildsDir` — but the async goroutine is still cloning the repo into that directory, leaving temporary git files that `os.RemoveAll` cannot remove on Linux.

**Expected:** Test completes without cleanup errors.
**Actual:** `os.RemoveAll` on buildsDir fails because `.git` still has contents from the in-progress clone.

## Root Cause Analysis

- **Trigger code path:** `HandleCreate` → `Trigger` (engine.go:77) → `go d.runDeployment(...)`
- **BuildsDir collision:** `Trigger` creates `buildDir = filepath.Join(d.buildsDir, id)` and the goroutine runs `git clone <bare-repo> .` in that directory
- **Cleanup race:** Test function returns before the clone completes; `t.TempDir()` tries to remove `buildsDir` while git still has open file handles or temporary objects in `.git/objects/pack/`
- **Why CI only:** Linux `os.RemoveAll` is stricter about non-empty directories during traversal than macOS. macOS may silently succeed in removing files that are still open; Linux (ext4/XFS) can fail with `ENOTEMPTY`.

This is **not** a regression from recent code — the pattern has existed since the test was written. The test happened to "pass" on macOS because `os.RemoveAll` is more lenient, but fails consistently on Linux CI runners.

**Risk level:** Low — only affects test cleanup, not production code.

## TDD Fix Plan

One cycle — no new behavior, just test cleanup hardening:

1. **RED/GREEN**: Capture the deployment ID from the POST response and call `waitForDeploymentTerminal(t, handler, depID, 5*time.Second)` before the test returns. This waits for the clone/build to reach `"running"` or `"failed"` state, ensuring all git file handles are released before `t.TempDir()` cleanup.
   **verify:** `go test ./components/deploy/... -run TestDeployCustomSiteName -count=1`

## Acceptance Criteria

- [ ] `go test ./components/deploy/... -run TestDeployCustomSiteName -count=10` passes on Linux
- [ ] Test still validates the URL prefix assertion (`https://my-custom-slug.bigbase.click`)
- [ ] No existing deploy test regressions

## Resolution

<!-- filled in by validate-fix -->
