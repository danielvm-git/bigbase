---
bug_id: BUG-004
status: fixed
severity: critical
scope: deploy
component: python
title: Python/uvicorn deployments fail with "Failed to spawn" and "app:" ASGI import string
---

# BUG-004: Python/uvicorn Deploy Spawn Failure

**Summary:** Three independent bugs in `pythonStartCommand` prevented Python deployments with uvicorn from starting. Together they caused a cascading failure: first `cmd.Dir` not set (uv can't find project), then `$PORT` literal, then `"app:"` ASGI import string.

**Reported:** 2026-07-12

## Current Behavior

All Python deployments with `pyproject.toml` that declare a uvicorn dependency fail with different errors depending on which bug masked which:

1. **Bug A — `cmd.Dir` not set:** `[runtime] error: Failed to spawn: uvicorn — No such file or directory (os error 2)` — uv cannot find the project's `.venv` because the working directory defaults to BigBase's CWD (`/opt/bigbase`).

2. **Bug B — Literal `$PORT` string:** When Bug A was fixed (by deployment resumption via `deployRunner` which sets `cmd.Dir`), uvicorn would fail because `--port $PORT` is passed as a literal string — `exec.CommandContext` does not shell-expand.

3. **Bug C — Missing `appVar` default:** When both A and B were fixed, uvicorn would fail with `ERROR: Error loading ASGI app. Import string "app:" must be in format "<module>:<attribute>"` — when `pyproject.toml` has no `[project.scripts]` section, `EntryPoint()` returns `("", "")`, the code defaults module to `"app"` but leaves `appVar` empty, producing `"app:"`.

## Root Cause

**Bug A:** `startApp` in `engine.go` sets `cmd.Dir` for Go and Node types but not Python. `pythonStartCommand` also never sets `cmd.Dir`. The `deployRunner.spawnProcess` path sets `cmd.Dir = spec.Dir`, so only new deployments via `startApp` are affected.

**Bug B:** Line 162 and 167 in `python.go` use the string `"$PORT"` as a literal argument. `exec.CommandContext` passes arguments directly to the OS — no shell expansion occurs. `PORT` is set in the environment (line 479 of `engine.go`) but uvicorn's `--port` flag only reads from CLI args.

**Bug C:** Lines 157-160 in `python.go` only check `module == ""` and default it to `"app"`, but never check `appVar`. When no `[project.scripts]` section exists, `EntryPoint()` returns `("", "")`, producing the ASGI string `"app:"`.

## Expected Behavior

1. Python app processes start with `cmd.Dir` pointing to the build directory.
2. `--port` receives the actual port number, not a literal `$PORT`.
3. The ASGI import string defaults to `"app:app"` (the uvicorn convention) when no entry point is declared.

## Fix Applied

Three changes to `pythonStartCommand` in `components/deploy/python.go`:

1. **Bug A:** `cmd.Dir = buildDir` set on every return path inside `pythonStartCommand`.
2. **Bug B:** Function signature changed to `pythonStartCommand(ctx, buildDir, port int)`. `$PORT` replaced with `fmt.Sprintf("%d", port)`.
3. **Bug C:** Added `if appVar == "" { appVar = "app" }` after the module default.

Callers updated:
- `components/deploy/engine.go`: `pythonStartCommand(ctx, buildDir, deploy.Port)`
- `components/deploy/deploy_runner.go`: `pythonStartCommand(ctx, spec.Dir, spec.Port)`

Test: `TestPythonStartCommand_UvicornDefaultsAppApp` in `python_test.go` verifies the ASGI string is `app:app` and port is numeric.

## Verify Steps

```bash
go test ./components/deploy/... -count=1
```

## Resolution

**Fixed:** 2026-07-12

**Root cause confirmed:** Three independent bugs in `pythonStartCommand` — missing `cmd.Dir`, literal `$PORT` string, and missing `appVar` default.

**Fix applied:**
1. `cmd.Dir = buildDir` on all return paths in `pythonStartCommand`
2. `port int` parameter replaces literal `$PORT`
3. `if appVar == "" { appVar = "app" }` added after module default
4. Callers in `engine.go` and `deploy_runner.go` pass port

**Hardening added:** The `cmd.Dir = buildDir` pattern now lives inside `pythonStartCommand` (same place as command construction) so future callers can't forget it. The port parameter is typed (`int`) so `$PORT` string injection is structurally impossible.

**Evidence:** 260/260 deploy tests pass (`go test ./components/deploy/... -count=1`).

**Commit:** `fix(deploy): default ASGI import string to app:app when no entry point declared`
