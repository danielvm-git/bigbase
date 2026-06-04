# BUG-2026-06-04T120500: Node site deploy fails — npm not on VPS; UI says "live"

## Problem

- **Actual:** Creating site `big-dock-locker-site` ends with badge **FAILED** while the wizard heading reads **"Your site is live"**. Public URL is assigned (`https://danielvm-git-big-dock-locker-site.bigbase.click`) but the app never serves.
- **Expected:** Node (or static) repos build successfully on production; step 3 shows accurate status and a clear error when build fails.
- **Reproduce:** On `https://bigbase.click/admin/`, create site from a GitHub repo with `package.json` (Node stack) → Deploy → wait → status `failed`.

**Production evidence (2026-06-04):**

```text
journalctl -u bigbase:
  "build app","type":"node","error":"npm install: exec: \"npm\": executable file not found in $PATH"

deployments row:
  id=1b5e10a2373bc32a9f5a5bd4fdd1ab27 status=failed app_type=node
```

VPS: `which npm` → not found (only `git` on PATH for `bigbase` user).

## Root Cause Analysis

**Primary (verified):** The deploy component detected a **Node** app (`package.json` present), ran `npm install`, and failed because **Node.js/npm are not installed** on the Contabo VPS. `scripts/setup-vps.sh` installs curl, caddy, ufw, rsync — not `nodejs` or `npm`. The `bigbase` systemd service runs builds as that user without a build toolchain.

**Contributing UI bug (verified):** `CreateSitePage` sets `deploying=false` immediately after the create API returns, while status is still `pending`/`building`. The step-3 title uses `deploying ? 'Building…' : 'Your site is live'`, so users see **"Your site is live"** during build and after **failed** — contradicting the badge.

**Contributing observability gap:** Build failures are only logged server-side (`slog`); `/api/deploy/:id/logs` returns `log_available: false`. The UI never surfaces `npm: executable file not found`.

**Related (not recurrence):** [BUG-2026-06-04T114000](BUG-2026-06-04T114000-prod-site-localhost-url.md) fixed public URLs; this deploy failed before serving regardless of URL.

**Risk level:** Medium — affects all Node (and likely Go/Python) deploys on production until VPS tooling is installed; UI misleads operators.

## TDD Fix Plan

1. **RED:** Integration or deploy test that simulates `npm` missing from `PATH` and expects deployment status `failed` with a persisted or API-visible error reason (or document contract test on `setup-vps.sh` requiring `nodejs`).
   **GREEN:** Add `nodejs` + `npm` (and optionally `python3`, `go` if supported) to `scripts/setup-vps.sh`; document in `specs/plans/DEPLOY.md`; one-time `apt install` on VPS.
   **verify:** `grep -q nodejs scripts/setup-vps.sh && bash -n scripts/setup-vps.sh`

2. **RED:** `CreateSitePage.test.tsx` — when polled status is `failed`, heading must not be "Your site is live"; show failure copy and error hint.
   **GREEN:** Keep `deploying=true` until status is `running` or `failed`; title/message keyed off `doneStatus` (`failed` → "Deploy failed", `running` → "Your site is live", else "Building…").
   **verify:** `cd ui && npm test -- --run CreateSitePage`

3. **RED:** Deploy test or API test — after failed `npm install`, GET deployment includes `error` or logs endpoint returns last build error line.
   **GREEN:** Store `error_message` on deployments (migration) or implement minimal log buffer; set from `buildApp` failure; UI shows in step 3.
   **verify:** `go test ./components/deploy/... -run TestDeployBuildError -count=1`

**REFACTOR:** Align `SiteDetailPage` / deploy table to show same error field; add preflight in deploy start that checks `npm`/`go`/`python` when app type requires them.

## Acceptance Criteria

- [ ] Node repo deploy reaches `running` on VPS after `nodejs` installed (or clear docs if static-only)
- [ ] Create-site wizard never shows "Your site is live" when status is `failed` or still building
- [ ] User-visible message explains missing toolchain or build command failure
- [ ] Regression tests pass; `go test ./...` and UI tests green

## Resolution

Closed with [BUG-2026-06-04T120800](BUG-2026-06-04T120800-deploy-npm-eacces-node20.md): `setup-vps.sh` installs Node 20/npm, create-site wizard no longer shows "Your site is live" on failed builds, API returns `error_message`.
