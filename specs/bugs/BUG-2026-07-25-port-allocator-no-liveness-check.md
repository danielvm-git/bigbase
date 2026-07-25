# BUG-2026-07-25: Port allocator hands out ports still held by orphaned processes — new deployments silently serve a different site's content

## Problem

Discovered live while standing up `bigbase-canary-python` (danielvm-git/bigbase-canary-python) as
part of a canary fleet built specifically to exercise `bigbase-deploy` end to end.

### Symptoms
- `POST /api/sites/{id}/deploy` returns 201, the deployment record's `latest_deployment.status`
  eventually reads `failed`, but `https://<site>.bigbase.click` returns **HTTP 200 with a
  completely different site's content** (`python.bigbase.click` served `bolao.bigbase.click`'s
  built SPA assets; earlier in the same session, `go.bigbase.click` transiently served
  `grimoire.bigbase.click`'s dashboard).
- `bigbase-deploy`'s own health check (GET the site URL, accept 200/301/302) reports
  `✅ Site LIVE` even though it is checking a completely unrelated app — a false positive.
- The failure is **intermittent**: re-running the exact same deploy (same commit, same site)
  a few minutes later succeeds and serves the correct content.
- The metadata banner BigBase itself injects (`window.__BIGBASE_METADATA__`) correctly shows
  the *intended* deployment's commit SHA even while the wrong site's markup is what's actually
  rendered — proving BigBase's own bookkeeping is internally consistent; only the proxy → process
  wiring is wrong.

### This is a recurrence, not a new class of bug

Two prior bug reports already targeted this exact symptom family and were both marked fixed:

- `BUG-2026-06-20T014500-redeploy-stale-static-files.md` — redeploying a site kept routing to
  the *first* deployment's port because `RegisterDeploymentHost` refused to update an existing
  mapping. Fixed by making host registration replace-on-conflict + stopping the previous
  deployment before starting the new one.
- `BUG-2026-06-21T120000-orphaned-process-apps-and-list-repos-sql.md` — after a BigBase restart,
  the in-memory `d.apps` map is empty, so `stopDeployment` couldn't find and kill the OS process
  from the *previous* deployment of the *same* site. Fixed by persisting the PID and killing by
  PID when the in-memory entry is missing.

Both fixes are scoped to **redeploying the same site**. Neither touches how a **brand-new**
deployment (a different site's first-ever deploy, or any deploy following a BigBase restart)
picks its port in the first place — which is the actual root cause below. That's why the bug
"comes back" in a new shape each time: the symptom (wrong content served) is the same, but the
trigger keeps shifting to whatever code path the previous fix didn't cover.

## Root Cause Analysis

`components/deploy/utils.go`:

```go
var pickPortMu sync.Mutex
var pickPortCounter int64

func pickPort(base int) int {
	pickPortMu.Lock()
	defer pickPortMu.Unlock()
	pickPortCounter++
	return base + int(pickPortCounter)
}
```

`pickPortCounter` is an **in-memory, process-lifetime counter** with two independent problems:

1. **No liveness check.** It never asks the OS whether the candidate port is actually free
   (no `net.Listen` probe). It hands out `base+1`, `base+2`, ... purely by arithmetic.
2. **Resets to 0 on every BigBase restart.** Since it's a package-level `int64`, not persisted
   anywhere, a restart means the *very next* deployment gets `base+1` again — a port number
   that was almost certainly already assigned to some earlier deployment before the restart.

Per `BUG-2026-06-21`, orphaned process-based apps (Python/Go/Node) are only cleaned up when
*that same site* is redeployed. An orphan from site A, still bound to `base+3` from before a
restart, is never touched by a brand-new deployment of unrelated site B that also happens to
get handed `base+3` by the reset counter. Site B's new process fails to bind (or the two race),
while site A's orphan is still very much alive and answering — and since the proxy's host→port
mapping for site B now points at `base+3` regardless, requests to site B's domain reach site A's
orphaned process instead.

This explains every observed detail:
- **Wrong site, not stale-own-content** — because the collision is with an unrelated site's
  leftover process, not this site's own previous deployment.
- **Intermittent** — only manifests right after a restart, until the counter has walked past
  every port an orphan is still squatting.
- **`bigbase-deploy`'s health check false-positives** — it only checks for HTTP 200 on the site
  URL; the orphan happily returns 200 for its own routes.
- **Deployment marked `failed` while a response still comes back** — the *new* process's own
  start/health sequence fails (can't bind, or its own health probe never reaches port `base+3`
  because something else answers first depending on bind order), which is scored as `failed`,
  independent of what the *proxy* ends up forwarding to.

## Fix Approach

Make `pickPort` verify OS-level availability before handing out a candidate, skipping ports
that are actually occupied — this is the fix the two prior patches should have applied at the
allocation site instead of (only) at redeploy-cleanup time. Loopback-bind-and-release is
sufficient: anything bound to `0.0.0.0:<port>` or `127.0.0.1:<port>` will make a
`127.0.0.1:<port>` listen attempt fail with `EADDRINUSE`.

```go
func pickPort(base int) (int, error) {
	pickPortMu.Lock()
	defer pickPortMu.Unlock()
	for attempt := 0; attempt < maxPickPortAttempts; attempt++ {
		pickPortCounter++
		candidate := base + int(pickPortCounter)
		if portIsFree(candidate) {
			return candidate, nil
		}
	}
	return 0, fmt.Errorf(...)
}
```

This closes the immediate hole (new deployments never collide with a still-live orphan again,
restart or not) without needing to touch the redeploy-cleanup logic the two prior fixes already
built. It does not eliminate the underlying orphan-accumulation itself (that's the June 20/21
fixes' territory, and per this session those fixes are holding for same-site redeploys) — it
makes port collision with an orphan structurally impossible regardless of *why* the orphan
exists, which is the more durable fix given this bug's history of resurfacing through whatever
gap the last patch left open.

## Acceptance Criteria

- [x] `pickPort` never returns a port an OS-level listener can't bind to
- [x] Regression test proves a pre-occupied candidate port is skipped, not handed out
- [x] `Trigger`'s call site updated to handle `pickPort`'s new error return
- [x] All existing tests pass

## Resolution

**Fixed:** 2026-07-25
**Root cause confirmed:** `pickPort` (components/deploy/utils.go) allocates ports by an
in-memory counter with no OS-level liveness check, and the counter resets on every process
restart — so deployments made shortly after a restart are handed port numbers that orphaned
processes from before the restart may still be bound to.
**Fix applied:** `pickPort` now probes each candidate with a real `net.Listen`/close on
`127.0.0.1:<port>` before returning it, retrying the next candidate on collision (bounded,
returns an error if no free port is found in range). `Trigger` (engine.go) updated to handle
the new `(int, error)` signature.
**Evidence:** `TestPickPort_SkipsOccupiedPort` and `TestPickPort_ReturnsErrorWhenExhausted` in
`components/deploy/utils_test.go`; full `go test ./components/deploy/...` green.
**Verification note:** confirmed against source and unit-tested; full production verification
(watching a real post-restart deploy cycle) needs a session with server access to observe —
flagging this explicitly rather than claiming end-to-end proof I couldn't actually observe.
