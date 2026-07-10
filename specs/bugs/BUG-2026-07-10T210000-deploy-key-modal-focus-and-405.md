---
bug_id: BUG-2026-07-10T210000
status: fixed
severity: high
scope: ui, auth
title: Generate Deploy Key modal loses input focus per keystroke; POST returns 405
---

# BUG-2026-07-10T210000: Deploy Key modal broken — focus theft + 405 on generate

## Problem

**Actual:**
1. Typing in the "Key Name" field of the Generate Deploy Key modal loses focus after every keystroke, forcing the user to re-click the input each time.
2. Clicking "Generate Key" returns `method not allowed`.

**Expected:** Typing is uninterrupted; clicking "Generate Key" creates the key.

**Reproduce:**
1. Open a site's Deploy Keys tab → Generate Key
2. Type any character in "Key Name" → focus jumps away from the input
3. Click "Generate Key" → see "method not allowed"

## Root Cause Analysis

### Bug 1 — focus theft (ui/src/components/Modal.tsx)

`Modal`'s focus-management `useEffect` depended on `[open, onClose]`. `onClose` passed into `GenerateModal`'s `<Modal onClose={handleClose}>` is a new function reference on every render, and `GenerateModal` re-renders on every keystroke (`setName`). Each re-render therefore re-ran the effect: the cleanup restored focus to the pre-open trigger, then the effect re-focused `getFocusable(dialog)[0]` — the modal's close button (✕), which appears before the `<input>` in DOM order — stealing focus away from the field being typed into.

**Root cause:** effect dependency on an unstable callback reference caused the focus-trap init/teardown to re-run on every keystroke instead of only on open/close.

### Bug 2 — 405 on POST /api/sites/{id}/deploy-keys (main.go)

`components/auth/middleware.go`'s `ProtectedHandler()` registers the deploy-key routes (`POST/GET /api/sites/{id}/deploy-keys`, `DELETE /api/sites/{id}/deploy-keys/{keyID}`), but `main.go` wires each `ProtectedHandler()` route onto the shared proxy mux **individually** (e.g. `p.Handle("GET /api/orgs", ...)`) rather than mounting the whole sub-mux at a prefix — and no lines were added for the new deploy-key routes when they were introduced (epic e74). Requests to `/api/sites/{id}/deploy-keys` therefore fell through to the `sites` component's catch-all handler at `/api/sites/` ([components/sites/sites.go:173](components/sites/sites.go#L173)), which has no knowledge of `deploy-keys` and falls through to its final `if r.Method != http.MethodGet { method not allowed }` guard.

**Root cause:** new protected routes were added to the auth component's internal mux but never registered on the top-level proxy mux in `main.go`, so they were unreachable — dead code shadowed by the sites component's broader catch-all route.

**Risk level:** High — the entire deploy-keys feature (epic e74) was non-functional in production despite shipping.

## TDD Fix Plan

### Cycle 1 — Stop focus-trap effect from re-running on every keystroke

**RED:** Typing in the Key Name field loses focus after each character (manual repro; no direct DOM test harness in this component's suite).

**GREEN:** [ui/src/components/Modal.tsx](ui/src/components/Modal.tsx) — store `onClose` in a ref (`onCloseRef`), update it every render, and reference `onCloseRef.current()` inside the `Escape` handler. Change the effect's dependency array to `[open]` only, so it inits once per open/close transition instead of per keystroke.

**verify:** `cd ui && npm run build` (typecheck); manual verification in browser.

### Cycle 2 — Register site deploy-key routes on the proxy mux

**RED:** `go build` succeeds but `POST /api/sites/{id}/deploy-keys` returns 405 at runtime (routes registered on `authComp.ProtectedHandler()` mux, never mounted onto `p` in main.go).

**GREEN:** [main.go](main.go) — add three `p.Handle(...)` registrations mirroring the existing org routes pattern:
- `p.Handle("POST /api/sites/{id}/deploy-keys", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)`
- `p.Handle("GET /api/sites/{id}/deploy-keys", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)`
- `p.Handle("DELETE /api/sites/{id}/deploy-keys/{keyID}", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)`

Go's stdlib `http.ServeMux` (1.22+) resolves the most specific pattern regardless of registration order, so these take precedence over the broader `/api/sites/` catch-all.

**verify:** `go build -o /tmp/bigbase-check . && go test ./components/auth/... ./components/sites/...`

**REFACTOR:** None — both fixes are minimal, targeted changes.

## Acceptance Criteria

- [x] `Modal.tsx` focus-trap effect depends only on `[open]`; `onClose` read via ref
- [x] `cd ui && npm run build` passes with no TypeScript errors
- [x] `main.go` registers all three site deploy-key routes on the proxy mux
- [x] `go build` succeeds
- [x] `go test ./components/auth/... ./components/sites/...` passes (249 tests)
- [ ] Manual verification in production: typing in Key Name field keeps focus; Generate Key succeeds

## Resolution

**Fixed:** 2026-07-10
**Root cause confirmed:** (1) unstable `onClose` reference in `Modal`'s effect deps caused focus-trap re-init per keystroke; (2) deploy-key routes registered on the auth component's internal mux were never mounted onto the top-level proxy router in `main.go`.
**Fix applied:** Ref-based `onClose` + `[open]`-only deps in `Modal.tsx`; three missing `p.Handle(...)` route registrations added in `main.go`.
**Evidence:** `npm run build` clean; `go build` clean; `go test ./components/auth/... ./components/sites/...` — 249 passed.
**Next:** Commit, push to `main`, confirm CI/CD deploy succeeds, then manually verify in the browser.
