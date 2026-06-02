# BUG-2026-06-02T163718: Dashboard root URL shows “waiting for agent” instead of YAML cockpit

## Problem

**Actual:** After `bash visual-dashboard/scripts/start-server.sh --project-dir "$(pwd)"`, opening the printed `url` (e.g. `http://localhost:63226/`) shows “Bigpowers Dashboard — Waiting for the agent to push a screen…” with an empty white page. Browser console may show `404` for `/favicon.ico` (noise only).

**Expected:** With `--project-dir` pointing at a repo that has `specs/state.yaml` and `specs/release-plan.yaml`, the default URL should show release/epic status (YAML cockpit) without requiring an agent to push `content/*.html` first.

**Reproduce:**

```bash
cd /path/to/big-appwrite
bash .opencode/skills/visual-dashboard/scripts/start-server.sh --project-dir "$(pwd)"
# Open the "url" from JSON output (root / only)
```

**Works today (not advertised in startup output):**

```bash
curl -s "http://127.0.0.1:PORT/api/status?projectDir=$(pwd)" | jq .release.version
# Browser: http://127.0.0.1:PORT/cockpit.html?projectDir=$(pwd)
```

## Prior art

No prior bug in `specs/bugs/registry.yaml` for visual-dashboard routing. Related: specs migrated to YAML (2026-06-02); skill documents cockpit at `/cockpit.html` but `start-server.sh` still prints only root `url`. **Novel** UX gap after YAML migration, not a recurrence.

## Root Cause Analysis

### Phase 1 — Reproduce

Consistent when `content/` has no `.html` files (normal after `.bigpowers` cleanup). Server is healthy; `/api/status` returns JSON.

### Phase 2 — Isolate

Failure is at the **HTTP routing layer** of the visual-dashboard server:

- `GET /` → if no newest file in session `content/`, respond with static `WAITING_PAGE` (“Waiting for the agent…”).
- `GET /cockpit.html?projectDir=<abs>` → YAML PM view (requires query param).
- `start-server.sh --project-dir` sets session directory under `.bigpowers/dashboard/` but does **not** pass project path into the Node server or the printed URL.

The user follows the only URL in the startup JSON; they never receive `projectDir` or `/cockpit.html`.

### Phase 3 — Hypothesize

| # | Hypothesis | Probability |
|---|------------|-------------|
| H1 | Root `/` is legacy “agent screen” mode; cockpit is a separate route not linked from startup | **High (verified)** |
| H2 | Server failed to start | Low (API works) |
| H3 | Missing specs | Low (`/api/status` OK) |

### Phase 4 — Verify

Inspected `server.cjs`: lines 171–175 serve `WAITING_PAGE` when `getNewestScreen()` is null. SKILL.md table: `GET /` = “Agent-pushed HTML screens (legacy, unchanged)”. **H1 confirmed.**

### Secondary defect (cockpit data wrong even when opened)

`read-specs-status.cjs` reads epic status from `execution-status.yaml` key `development_status`, but the repo file uses top-level `epics:` / `stories:` (from `sync-status-from-epics.sh`). Result: cockpit shows all epics as `pending` (e.g. e17 should be `done`). Planning block parser also expects nested `status:` under workflow keys; `planning-status.yaml` uses flat `scope: done` lines.

**Risk level:** Low (UX / status display; no data loss). Medium if users trust wrong epic status in cockpit.

## TDD Fix Plan

### Cycle 1 — Startup contract (primary)

1. **RED:** Shell test (or Node integration test): run `start-server.sh --project-dir <fixture>`; parse JSON stdout; assert field `cockpit_url` contains `/cockpit.html?projectDir=` and `projectDir` matches fixture.
   **GREEN:** Export `BIGPOWERS_PROJECT_DIR` in `start-server.sh`; extend server-started JSON with `project_dir` and `cockpit_url`.
   **verify:** `bash .opencode/skills/visual-dashboard/scripts/start-server.sh --project-dir "$(pwd)" | jq -e '.cockpit_url'`

### Cycle 2 — Default route redirect

2. **RED:** HTTP test: `GET /` with env `BIGPOWERS_PROJECT_DIR` set, empty `content/` → expect `302` to `/cockpit.html?projectDir=...` (or `200` with cockpit body).
   **GREEN:** In `handleRequest`, when no agent screen and `process.env.BIGPOWERS_PROJECT_DIR` is set, redirect or serve cockpit as default.
   **verify:** `curl -sI http://127.0.0.1:PORT/ | grep -i location`

### Cycle 3 — Execution status parsing

3. **RED:** Node test: `readSpecsStatus(fixture)` → `epics.find(e => e.id === 'e17').status === 'done'` when fixture `execution-status.yaml` has `epics: e17: done`.
   **GREEN:** Parse `epics:` and `stories:` maps from `execution-status.yaml` (merge into status lookup); keep backward compat for `development_status` if present.
   **verify:** `node .opencode/skills/visual-dashboard/scripts/read-specs-status.cjs "$(pwd)" | jq '.epics[] | select(.id=="e17") | .status'`

### Cycle 4 — Planning status parsing

4. **RED:** Node test: `planning_status.scope.status === 'done'` for flat `discover:\n  scope: done` YAML.
   **GREEN:** Adjust planning parser for `planning-status.yaml` flat key/value under `discover:`.
   **verify:** same read-specs-status jq one-liner on `planning_status`

### Cycle 5 — Favicon noise (optional)

5. **RED:** `GET /favicon.ico` returns 404 today.
   **GREEN:** Return `204` or minimal ICO from server.
   **verify:** `curl -sI http://127.0.0.1:PORT/favicon.ico | head -1`

**REFACTOR:** Update `visual-dashboard/SKILL.md` and `specs/README.md` so “Starting a Session” tells users the printed `cockpit_url` is the default entry point when using `--project-dir`.

## Acceptance Criteria

- [x] Opening printed URL after `start-server.sh --project-dir $(pwd)` shows YAML release/epic status without agent-pushed HTML
- [x] Startup JSON includes `cockpit_url` with absolute `projectDir`
- [x] Cockpit shows e17 as `done` when `execution-status.yaml` says so
- [x] `/favicon.ico` does not 404 (returns 204)
- [x] Existing agent `content/*.html` flow still works when files exist (newest screen wins on `/` or documented precedence)
- [x] All new tests pass; `npm run preflight` unchanged for Go/UI

## Resolution

**Fixed:** 2026-06-02

**Root cause confirmed:** Root `/` served legacy `WAITING_PAGE` when `content/` was empty; `start-server.sh` did not pass `BIGPOWERS_PROJECT_DIR` or advertise `cockpit_url`. `read-specs-status.cjs` parsed wrong YAML keys (`development_status` vs `epics:`; planning as objects).

**Fix applied:**
- `start-server.sh` exports `BIGPOWERS_PROJECT_DIR`; startup JSON includes `cockpit_url` and `project_dir`
- `server.cjs` redirects `/` to cockpit when project dir set and no agent HTML; favicon 204; HEAD support
- `read-specs-status.cjs` parses `epics`/`stories` status maps and flat `planning-status` items
- `cockpit.html` renders string planning statuses correctly
- `read-specs-status.test.cjs` (3 tests)

**Hardening added:** Node unit tests for status parsing (schema-like assertions on epic/planning shape).

**Evidence:**
```bash
node --test .opencode/skills/visual-dashboard/scripts/read-specs-status.test.cjs
bash .opencode/skills/visual-dashboard/scripts/start-server.sh --project-dir "$(pwd)" | jq -e '.cockpit_url'
# curl -sI http://127.0.0.1:PORT/ → 302 Location: /cockpit.html?projectDir=...
node .opencode/skills/visual-dashboard/scripts/read-specs-status.cjs "$(pwd)" | jq '.epics[]|select(.id=="e17")|.status,.planning_status.scope'
npm run preflight
```

**Commit:** `fix(visual-dashboard): default to YAML cockpit and parse execution-status`
