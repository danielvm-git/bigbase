# BUG-2026-06-21T120000: Orphaned process-based apps survive redeploy + list_repos SQL column error

## Problem

### Issue A — Orphaned process apps (Python/Go/Node) survive redeploys

**Reported site:** bolao.bigbase.click (Python Telegram bot, app.py)
**Symptom:** After deploying release 1.12.0, the Telegram bot intermittently replies with
old code. A burst of 10 `getUpdates` calls returned 3× HTTP 409 Conflict, proving that
multiple processes are simultaneously holding the Telegram long-polling connection.
**Timeline:** At least one orphaned process dates back to before June 20 — several
redeploys have accumulated zombie app.py instances on the VPS.

**Expected:** Each redeploy kills the previous process-based deployment so exactly one
app.py holds the Telegram token at any time.

### Issue B — `list_repos` MCP tool fails with SQL error

`list_repos` on mcp.bigbase.click returns:
```
SQL logic error: no such column: updated_at (1)
```

## Root Cause Analysis

### Bug A — Process not killed when absent from in-memory app registry

The deploy component tracks running process-based apps (Python, Go, Node) in an in-memory
map (`d.apps`, keyed by deployment ID). `stopDeployment` only kills a process when its
entry is present in that map:

```
stopDeployment(id):
    app, hasApp = d.apps[id]    ← only kills if in-memory entry exists
    if hasApp: kill process
    update DB status to "replaced"  ← always runs, even when hasApp=false
```

A process is absent from `d.apps` after any of these events:
- BigBase receives SIGKILL or crashes (panic) — `Stop()` is never called, children
  become OS orphans reparented to PID 1.
- BigBase restarts normally but `resumeCandidates` only resumes **static** deployments.
  Process-based apps (Python/Go/Node) are skipped; d.apps starts empty for them.

Consequence: the next `Trigger` call finds the old deployment in the DB (status='running'),
calls `stopDeployment`, but `hasApp=false` so the orphaned OS process is **not killed**.
A new process is launched alongside it. Each subsequent redeploy adds another instance,
accumulating parallel pollers that compete for the Telegram token.

The fix introduced in BUG-2026-06-20T014500 (stop-previous on redeploy) correctly handles
the in-memory case but left this gap for the crash/restart scenario.

**Risk: High** — Any long-running or worker-style process app is affected. Side effects
include port conflicts, duplicate background workers, and inconsistent bot behavior.

### Bug B — MCP `list_repos` query references non-existent column

The MCP tool queries `updated_at` from `git_repos`, but the git component never creates
that column — the schema only has `created_at`. The MCP test creates `git_repos` with
`updated_at`, masking the mismatch. In production, the real schema is used and the query
fails at runtime.

**Risk: Low** — Read-only, cosmetic breakage. Blocks self-service inspection via MCP.

## TDD Fix Plan

### Cycle 1 — Persist OS PID; kill orphaned process on next redeploy

**RED**: Write `TestStopDeploymentKillsOrphanedProcess` in the deploy package:
- Start a real subprocess (e.g. `sleep 60`) via a test deployment.
- Simulate a BigBase restart by clearing `d.apps` without calling `Stop()` (so the process
  is not killed).
- Trigger a new deployment for the same `site_id`.
- Assert that after `stopPreviousDeployments` runs, the original process is no longer
  running (check via `os.FindProcess` + `p.Signal(os.Signal(0))`).
**GREEN**:
1. Add `pid` column to deployments table via `ensurePIDColumn` migration.
2. In `startApp`, after `cmd.Start()` succeeds, persist `cmd.Process.Pid` to the DB.
3. In `stopDeployment`, when `hasApp=false`, read `pid` from DB; if `pid > 0`, call
   `os.FindProcess(pid)` and `p.Kill()`.
**verify**: `go test ./components/deploy/... -run TestStopDeploymentKillsOrphanedProcess`

### Cycle 2 — PID column migration is idempotent

**RED**: Write `TestEnsurePIDColumnIdempotent`:
- Call `Start()` (which runs all migrations including the new `ensurePIDColumn`) twice
  against the same DB.
- Assert no error on the second call.
**GREEN**: `ensurePIDColumn` uses `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` pattern with
duplicate-column error swallowing (same pattern as `ensureSiteIDColumn`).
**verify**: `go test ./components/deploy/... -run TestEnsurePIDColumnIdempotent`

### Cycle 3 — `list_repos` MCP tool returns results with real git_repos schema

**RED**: Update `TestListRepos` in mcp_test.go to create `git_repos` WITHOUT `updated_at`
(matching the real schema: `id, name, description, created_at`). Verify the tool call
succeeds and returns the repo list without error.
**GREEN**: In `mcp.go`, change the `list_repos` query from
`SELECT id, name, description, updated_at ... ORDER BY updated_at DESC`
to
`SELECT id, name, description, created_at ... ORDER BY created_at DESC`.
Scan into `created` (not `updated`). Update the test schema to match real git_repos.
**verify**: `go test ./components/mcp/... -run TestListRepos`

**REFACTOR**: Extract the `ensurePIDColumn` helper next to the other `ensure*Column` helpers
for consistency. No behavior changes.

## Acceptance Criteria

- [ ] After BigBase restarts (d.apps cleared), a subsequent redeploy for the same site
      terminates the previously running process-based app
- [ ] Only one instance of app.py is running per site after any sequence of redeploys
- [ ] Telegram 409 Conflict errors stop after platform-side cleanup
- [ ] `list_repos` MCP tool returns repositories without SQL error
- [ ] All new tests pass
- [ ] Existing tests still pass (`go test ./...`)

## Resolution

**Fixed:** 2026-06-21
**Root cause confirmed:**
- **Bug A**: `stopDeployment` only killed processes found in the in-memory `d.apps` map.
  After a BigBase restart (or crash), `d.apps` is empty, so orphaned OS processes
  survived redeploys and accumulated.
- **Bug B**: MCP `list_repos` query used `updated_at` column that doesn't exist in the
  real `git_repos` schema (`git_repos` only has `created_at`).

**Fix applied:**
1. Added `pid` column to deployments table (`ensurePIDColumn` migration).
2. `startApp` persists `cmd.Process.Pid` to DB after starting a process-based app.
3. `stopDeployment` reads PID from DB and kills the process when `hasApp=false`
   (orphaned after restart).
4. Fixed MCP `list_repos` query from `updated_at` to `created_at`.

**Evidence:** All 24 packages pass; `TestStopDeploymentKillsOrphanedProcess`
verifies orphaned processes are killed on redeploy; `TestListRepos` passes
with real `git_repos` schema.
