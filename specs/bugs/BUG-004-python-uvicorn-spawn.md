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

---

## Follow-up Fix (2026-07-12): CLI Entry Points Used as ASGI Import

### Additional Symptom
When `[project.scripts]` contains a CLI entry point (e.g., `grimoire.__main__:main`), the code reused it as the uvicorn ASGI import string. This caused deployed apps to run in a degraded state — responding to requests but serving unexpected content (e.g., JSON instead of HTML dashboards).

### Root Cause
`[project.scripts]` entries per PEP 621 define **console_scripts** (CLI commands), not ASGI applications. The code treated ALL script entries as ASGI import strings equally, when CLI entries like `__main__:main` should never be passed to uvicorn.

### Fix Applied
Added `isCLIScriptEntry(module, appVar) bool` heuristic that detects CLI patterns:
- Module ends with `__main__` (e.g., `grimoire.__main__`)
- App variable is `main`, `cli`, `entrypoint`, or `run`

When a CLI entry is detected, the ASGI import falls back to the universal FastAPI convention of `app:app` instead of using the CLI entry point.

### Known Limitation
~~Some apps (like Grimoire with `src/` layout) use a non-standard ASGI import path (e.g., `grimoire.app:create_app --factory`). These will fail cleanly with "Could not import module app" instead of running in a degraded state.~~ **Resolved below.**

### Evidence
267/267 deploy tests pass (7 new tests added for CLI detection).

---

## Follow-up Fix 2 (2026-07-12): Manifest `asgi_import` Override

### Symptom
Apps with non-standard ASGI import paths (e.g., `grimoire.app:create_app --factory`) could not be configured, forcing `app:app` which doesn't work for `src/`-layout packages.

### Root Cause
No mechanism existed to override the auto-detected ASGI import string. The only signal was `[project.scripts]` which conflates CLI commands with ASGI apps.

### Fix Applied
Added `asgi_import` field to `bigbase.yaml` under `start:`:

```yaml
start:
  command: ""                          # empty = auto-detect
  port: 8000
  asgi_import: "grimoire.app:create_app --factory"
```

Changes:
- `ManifestStart.ASGIImport` field in `manifest.go` — accepts `"module:app"` or `"module:app --factory --reload"` format
- `validate()` relaxed: `start.command` may be empty when `framework: python` and `asgi_import` is set
- `pythonStartCommand(ctx, buildDir, port, manifest)` now accepts `*Manifest`, checks `manifest.Start.ASGIImport` to override the ASGI import
- `parseASGIImport(raw)` splits the string into `(importStr, extraArgs)` for uvicorn flags
- `orchestrator.go` resume path now loads manifest from build dir (instead of passing nil)
- Callers in `engine.go` and `deploy_runner.go` updated

### Hardening Added
- **Type guard:** `port int` parameter type prevents `$PORT` string injection (structurally impossible)
- **Schema validation:** `ValidateManifest()` ensures `start.command` is required unless `framework: python` + `asgi_import` is set → validated by `TestValidateManifest_ASGIImportRelaxesStartCommand` (4 sub-cases)
- **Defensive defaults:** 4 layers of defaults ensure both `module` and `appVar` always resolve to `"app"` before passing to uvicorn
- **CLI heuristic escape hatch:** Manifest `asgi_import` field overrides any auto-detection, including false positives from `isCLIScriptEntry`
- **Path resolution:** `cmd.Dir = buildDir` on every return path prevents working directory confusion

### Evidence
270/270 deploy tests pass (3 new manifest tests: `TestParseASGIImport`, `TestPythonStartCommand_ManifestASGIImport`, `TestPythonStartCommand_ManifestASGIImportSimple`). 4 hardening tests: `TestValidateManifest_ASGIImportRelaxesStartCommand`.
