---
bug_id: BUG-2026-07-24-static-directory-listing
status: fixed
severity: critical
priority: critical
scope: deploy
title: Static hosts serve git checkout directory listings instead of built apps
github_issues: [155]
consumer_issues:
  - danielvm-git/big-exames#3
  - danielvm-git/big-library#4
  - danielvm-git/grimoire#5
related:
  - BUG-2026-07-24-deploy-404-site-not-found
security_impact: MEDIUM
---

# BUG-2026-07-24-static-directory-listing: Static host serves checkout directory listing

## Problem

Three production hosts return Go `http.FileServer` directory listings of the git checkout (HTTP 200 + `<pre><a href=".git/">`) instead of built app HTML:

| Host | Symptom |
|------|---------|
| https://library.bigbase.click | Astro source tree listing + `__BIGBASE_METADATA__` |
| https://exames.bigbase.click | Monorepo root listing (`web/`, `.big-release.yml`, …) |
| https://add-tutorial-requests-site.bigbase.click | Firebase repo root (`firebase.json`, `public/`, …) |

Also confirmed: https://grimoire.bigbase.click (Python app) listing the same way.

Working controls: bolao, docklocker, danielvm-git-clean-install-guide → real HTML.

**Security impact: MEDIUM** — directory listings expose `.git/`, source trees, and repo layout on public HTTPS hosts (info disclosure). No RCE path identified.

## Reproduce

1. Prod VPS `89.116.26.187`, binary `/opt/bigbase/bin/bigbase` → `version 2.79.11` (mtime 2026-07-25 01:58 CEST).
2. `curl -sL https://{library,exames,add-tutorial-requests-site,grimoire}.bigbase.click | head` → `<pre>` listings.
3. SQLite latest deploys (all `status=running`, `app_type=static` for broken static hosts):
   - exames `5f1c7f1c…` port 10003, created `2026-07-24T23:47:48Z`
   - library `1eb582f1…` port 10005, created `2026-07-24T23:48:06Z`
   - add-tutorial `24da70d8…` port 10352, created `2026-06-24T16:26:16Z`
   - grimoire `67a516de…` port 10004, created `2026-07-24T23:48:06Z`
4. Build logs for exames/library (176 bytes): clone → `Detected app type: static` → `Serving static files` — **no** `App root:`, **no** Node install/build.
5. On disk: no `index.html` under exames/library build trees; add-tutorial has `public/index.html` but serveDir is checkout root.

## Isolate

- Proxy + TLS healthy (200). Failure is `serveStatic` → `http.FileServer(http.Dir(serveDir))` with wrong `serveDir`.
- Fleet deploys at 23:47–23:48Z ran **~11 min before** 2.79.11 binary install (23:58Z). Those deploys never ran the #166 framework-static Node path; resume after restart keeps the bad checkout.
- exames `root_path=web` is set, but the running deploy log has no `→ App root: web` (pre-#166 binary path / no honor). Earlier node attempt `0945823a…` built at **monorepo root** → `vite: not found` when `pnpm -C web run build` ran without installing `web/` deps in-app-root.
- library: Astro (`astro.config.mjs`, `npm run build`) — needs install+build → `dist/`; never built.
- add-tutorial: Firebase layout — `public/index.html` present; platform never prefers `public/` for static FileServer (PHP already does).
- Soft fallback in `engine.go`: if outDir missing after build, `serveDir` stays `appRoot` and still marks `running`.

## Hypothesize (ranked) → Verify

1. **Platform — soft serveDir + no entrypoint gate** — deploy marks `running` without `index.html` under serveDir → FileServer listing. **Falsify:** code path before `serveStatic` rejects missing entry. **Result: confirmed** (engine serves any dir; live ports return listings).
2. **Platform — no `public/` static outDir** — Firebase/static sites with only `public/index.html` serve repo root. **Falsify:** add-tutorial disk has `public/index.html`, host lists root. **Result: confirmed**.
3. **Ops — stale fleet after #166** — hosts never redeployed on 2.79.11 with build. **Falsify:** deploy timestamps &lt; binary mtime; logs lack Node build. **Result: confirmed** (necessary but not sufficient alone for add-tutorial).
4. **Owner-only missing build** — only if platform correctly fails closed / builds and owner outDir still empty. **Result: deferred** until after platform fix + redeploy.

## Classification

| Site | Class | Notes |
|------|-------|-------|
| add-tutorial-requests-site | **platform** | Prefer `public/` when `public/index.html`; fail closed |
| library | **platform + ops** | Framework-static must build; fail closed; redeploy |
| exames | **platform + ops** | Honor `root_path=web` → adapter-node → AppNode; redeploy (not static listing) |
| grimoire | **ops (+ platform fail-closed)** | Python app mis-served as static listing; redeploy as python; fail closed prevents static lie |

## TDD Fix Plan

1. **RED**: framework-static / package.json without outDir `index.html` → deploy **failed** with `code=static_output_missing`, not listing  
   **GREEN**: before `serveStatic` / marking static `running`, require entrypoint; `failDeployment` + hint of tried paths  
   **verify**: `rtk go test ./components/deploy/ -run 'StaticOutputMissing|FailClosed' -count=1`

2. **RED**: repo with only `public/index.html` (no root index, no package.json) → serveDir `public`, HTML 200  
   **GREEN**: pure-static serveDir selection prefers `public/` when `public/index.html` exists  
   **verify**: `rtk go test ./components/deploy/ -run 'PublicDir|FirebaseStatic' -count=1`

3. **RED**: `root_path=web` + SvelteKit/Astro under `web/` → detection/build under `web`, not monorepo-root listing  
   **GREEN**: keep/extend `framework_mode_test.go` + engine path that logs `App root` and builds in appRoot  
   **verify**: `rtk go test ./components/deploy/ -run 'RootPath|FrameworkMode' -count=1`

## Acceptance

- [x] Missing `index.html` under chosen serveDir → failed deploy (`static_output_missing`), never FileServer listing
- [x] `public/index.html` pure-static → served from `public/`
- [x] `root_path` regressions green
- [x] Fleet redeploy: exames, library, add-tutorial, grimoire → curl real HTML (not `<a href=".git/">`)
- [x] Consumer issues updated/closed from VPS evidence; bigbase#155 updated

## Resolution

Shipped in **v2.79.12–v2.79.14** via PRs #167 / #168 / #169:

1. Fail-closed static serve (`static_output_missing`); prefer `public/`; skip bad static resume (#167)
2. Do not promote AppNode→AppStatic unless outDir has `index.html` (#168) — unblocks adapter-node
3. `GetStartCommand` → `node build/index.js` when present without `scripts.start` (#169)

**Fleet curl 2026-07-25:** library, exames, add-tutorial, grimoire, bolao → HTTP 200 real HTML.

Consumer: closed big-exames#3, big-library#4, grimoire#5; documented add-tutorial#2; closed bigbase#155.
