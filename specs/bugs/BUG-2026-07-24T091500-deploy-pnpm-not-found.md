---
bug_id: BUG-2026-07-24T091500
status: open
severity: high
scope: deploy
title: Node deploy ignores pnpm — build fails with pnpm not found
---

# BUG-2026-07-24T091500: Node deploy ignores pnpm (pnpm: not found)

## Problem

**Actual:** Site deploy runs `npm install` then `npm run build`. The build script invokes `pnpm`, which is not on PATH → `sh: 1: pnpm: not found`, exit 127.

**Expected:** Deploy detects the project's package manager from lockfile/`packageManager` and runs matching install/build commands; pnpm is available on the build host via Corepack.

**Reproduce:** Deploy a Node repo with `pnpm-lock.yaml` or a build script that calls `pnpm` (e.g. `big-exames.bigbase.click` on main).

**Security impact:** NONE

## Root Cause Analysis

### Reproduce
User deploy logs show npm install/build succeeding until build script shells out to pnpm.

### Isolate
`nodeInstall` and `buildApp` AppNode branch in deploy engine hardcode `npm install` / `npm run build`. VPS provisioning installs Node+npm only.

### Hypothesize
Platform assumes all Node projects use npm; alternate PMs never detected or installed.

### Verify
Code inspection confirms zero pnpm handling in `components/deploy/`; `setup-vps.sh` has no Corepack.

**Risk level:** Low (build tooling only)

## TDD Fix Plan

1. **RED:** `TestDetectNodePackageManager` — lockfiles + packageManager field  
   **GREEN:** `DetectNodePackageManager` in `node_pm.go`  
   **verify:** `go test ./components/deploy/ -run TestDetectNodePackageManager -count=1`

2. **RED:** `TestNodeInstallCommand_UsesPnpmWhenLockfilePresent`  
   **GREEN:** `NodeInstallCommand` + wire `nodeInstall`/`buildApp`  
   **verify:** `go test ./components/deploy/ -run 'TestNodeInstall|TestDetectNode' -count=1`

3. **RED:** `TestCacheKey_PnpmLockfile`  
   **GREEN:** extend `findLockfileHash`  
   **verify:** `go test ./components/deploy/ -run TestCacheKey -count=1`

4. **RED:** `TestEnsureNodePackageManager_MissingBun`  
   **GREEN:** `ensureNodePackageManager` with Corepack fallback  
   **verify:** `go test ./components/deploy/ -run TestEnsureNodePackageManager -count=1`

## Acceptance Criteria

- [ ] pnpm-lock repos use `pnpm install` / `pnpm run build` in deploy logs
- [ ] Cache keys include `pnpm-lock.yaml`
- [ ] Corepack enabled on VPS; pnpm on PATH for `bigbase` user
- [ ] All deploy tests pass

## Resolution

**Fixed:** 2026-07-24

**Root cause:** Deploy engine hardcoded `npm install` / `npm run build`; VPS had no pnpm on PATH.

**Fix applied:**
- `DetectNodePackageManager` + `ensureNodePackageManager` (Corepack fallback) in `node_pm.go`
- Wired install/build/start in `engine.go`, `deploy_runner.go`, `manifest.go`
- Extended cache lockfile priority for pnpm/bun
- `corepack enable` in `setup-vps.sh`

**VPS:** SSH from agent environment failed (publickey). Run on production host:
```bash
ssh root@89.116.26.187 'corepack enable && corepack prepare pnpm@latest --activate'
ssh root@89.116.26.187 'sudo -u bigbase -H bash -lc "which pnpm && pnpm -v"'
```
Redeploy `big-exames` after binary lands.

**Evidence:** `go test ./components/deploy/ -run 'TestDetectNode|TestNodeInstall|TestEnsureNode|TestCacheKey|TestGenerateManifest' -count=1` — 34 pass
