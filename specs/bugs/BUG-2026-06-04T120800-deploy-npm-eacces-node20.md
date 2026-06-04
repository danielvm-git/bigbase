# BUG-2026-06-04T120800: Node deploy fails after npm install — no HOME + Node 18

## Problem

- **Actual:** `big-dock-locker-site` redeploy still ends **FAILED** after installing `nodejs`/`npm` on the VPS. Latest deployment `5a697ec52d8447644b3dbafd7b61ec9e` (commit `fe813d8`).
- **Expected:** Node/Vite site builds (`npm install` + `npm run build`) and serves from `dist/` at the public subdomain.
- **Reproduce:** Redeploy the site on production after `apt install nodejs npm`; status remains `failed`.

**Production log (2026-06-04 17:07 UTC):**

```text
"build app","type":"node","error":"npm install: exit status 243"
```

**Manual repro on VPS (same build dir, as `bigbase`):**

```text
npm ERR! EACCES: permission denied, mkdir '/home/bigbase'
```

With `HOME=/opt/bigbase` and `NPM_CONFIG_CACHE=/opt/bigbase/.npm`, `npm install` succeeds; `npm run build` then fails:

```text
Vite requires Node.js version 20.19+ or 22.12+. Current: v18.19.1
```

## Root Cause Analysis

**Primary (verified):** The `bigbase` systemd user is created with `--no-create-home`. `npm install` writes cache and logs under `$HOME/.npm`; with no home directory npm exits with **EACCES** (logged as exit status 243). The deploy runner does not set `HOME` or `NPM_CONFIG_CACHE` on build subprocesses.

**Secondary (verified):** Ubuntu `nodejs` package is **18.19.1**. This repo uses **Vite 8**, which requires **Node ≥ 20.19**. Even after fixing HOME, `npm run build` fails until Node 20+ is on the PATH for `bigbase`.

**Contributing (unchanged from BUG-2026-06-04T120500):** Create-site wizard shows **"Your site is live"** while badge is **failed**; build errors are not returned via API/logs UI.

**Related:** [BUG-2026-06-04T120500](BUG-2026-06-04T120500-deploy-node-npm-missing.md) — first failure was missing npm binary; installing `nodejs`/`npm` alone is insufficient. Partial fix, not closed.

**Risk level:** Medium — blocks all Node/Vite deploys on production until HOME + Node 20 are configured.

## TDD Fix Plan

1. **RED:** Deploy test — `buildApp` subprocess env includes writable `HOME` when set on Deploy options (mock dir); or contract test that `setup-vps.sh` defines `HOME` for the service user.
   **GREEN:** Add `Environment=HOME=/opt/bigbase` and `NPM_CONFIG_CACHE=/opt/bigbase/.npm` to systemd unit in `setup-vps.sh`; create `/opt/bigbase/.npm` owned by `bigbase`; set same vars in `buildApp` via `cmd.Env`.
   **verify:** `grep -q HOME= /opt/bigbase` in script test; `bash -n scripts/setup-vps.sh`

2. **RED:** Document or test minimum Node version for Vite builds (Node 20+).
   **GREEN:** Install Node 20 LTS in `setup-vps.sh` (NodeSource or `nodejs` 20.x package); document in `specs/plans/DEPLOY.md`.
   **verify:** On VPS: `sudo -u bigbase node -v` → v20.x; `npm run build` in sample Vite app succeeds

3. **RED:** `CreateSitePage` test — failed deploy shows failure headline, not "Your site is live".
   **GREEN:** Keep `deploying` true until terminal status; branch title on `doneStatus`.
   **verify:** `cd ui && npm test -- --run CreateSitePage`

4. **RED:** Failed deployment exposes `error_message` from build (npm EACCES or vite node version).
   **GREEN:** Capture stderr in `buildApp`; persist on `deployments`; show in site detail / create wizard.
   **verify:** `go test ./components/deploy/... -run TestDeployBuildError -count=1`

**REFACTOR:** Preflight check before node build: verify `npm`, writable HOME, and `node -v` meets minimum.

## Acceptance Criteria

- [ ] `npm install` succeeds for `big-dock-locker-site` on VPS without EACCES
- [ ] `npm run build` completes; deployment status `running`; `dist/` served
- [ ] Fresh VPS setup via `setup-vps.sh` includes home/cache dirs and Node 20+
- [ ] UI shows accurate failure/success copy and surfaces build error when failed
- [ ] All tests pass

## Resolution

Fixed in `fix/deploy-node-build-env`: Node 20 via NodeSource in `setup-vps.sh`, `HOME`/`NPM_CONFIG_CACHE` on systemd + `BuildEnv` for npm subprocesses, `error_message` on failed deployments, create-site wizard shows accurate status/errors.
