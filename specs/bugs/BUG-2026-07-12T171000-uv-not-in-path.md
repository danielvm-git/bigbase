# BUG-2026-07-12T171000: Python deploys fail when uv is not installed on build server

| Field | Value |
|-------|-------|
| **ID** | BUG-2026-07-12T171000 |
| **Date** | 2026-07-12 |
| **Severity** | medium |
| **Priority** | high |
| **Scope** | deploy |
| **Status** | fixed |

## Summary

The deploy engine hardcodes `uv` for Python projects with `pyproject.toml`. If `uv` isn't installed on the build server, deployment fails with:

```
exec: "uv": executable file not found in $PATH
```

This is a BigBase infrastructure gap — same class as prior bug BUG-2026-06-19T173242 which fixed `pip`/PEP 668.

## Root Cause

Two sites hardcode `uv` without a PATH check:

1. **Build phase** — `engine.go:373`: `d.runBuildCommand(..., "uv", "sync", "--frozen")` — runs `uv sync` unconditionally during Python build
2. **Runtime phase** — `python.go:141-169`: `pythonStartCommand()` hardcodes `"uv"` in 3 exec calls (uvicorn, module, app.py)

Neither checks `exec.LookPath("uv")` before using it.

## Fix Plan (2 RED-GREEN cycles)

### Cycle 1: Build fallback
- **RED**: Verify `engine.go:373` calls `uv sync` even when `uv` is not on PATH
- **GREEN**: Add `exec.LookPath("uv")` check; fall back to `pip install --break-system-packages .` when uv is absent

### Cycle 2: Runtime fallback
- **RED**: Verify `pythonStartCommand()` returns a cmd with `"uv"` path even when uv is missing
- **GREEN**: Check `exec.LookPath("uv")` in `pythonStartCommand()`; when absent, fall back to `python3`/`python` via a new `resolvePythonBin()` helper

## Files Changed

- `components/deploy/engine.go` — Build: `LookPath("uv")` check + `pip install` fallback
- `components/deploy/python.go` — Runtime: `LookPath("uv")` check + `resolvePythonBin()` helper
- `components/deploy/python_test.go` — Tests: `TestResolvePythonBin`, `TestPythonStartCommand_PyProject_NoUv`

## Security Impact

NONE — no user-controlled input, no privilege escalation, no data exposure. Pure operational availability fix.

## Regression Guards

- `components/deploy/python_test.go` `TestResolvePythonBin` — validates helper returns python3 or python
- `components/deploy/python_test.go` `TestPythonStartCommand_PyProject_NoUv` — validates fallback command

## Resolution

**Fixed:** 2026-07-12
**Root cause confirmed:** Both `engine.go:373` (build) and `python.go:141-169` (runtime) hardcoded `"uv"` as the command name with no `exec.LookPath` check.
**Fix applied:** Added `exec.LookPath("uv")` checks at all 4 call sites. Build falls back to `pip install --break-system-packages .`. Runtime falls back to `python3`/`python` via a new `resolvePythonBin()` helper.
**Hardening added:** Runtime availability check (`exec.LookPath`) at invocation time — the same pattern that already exists for `python3`/`python` in the legacy legacy path. The `resolvePythonBin()` helper centralizes Python binary resolution.
**Evidence:** 6/6 specific Python tests pass. 198/199 deploy tests pass (pre-existing flaky integration test `TestHealthSummaryInStatusAPI` — confirmed fails identically without these changes). Build and lint both clean.
**Commit:** `fix(deploy): fall back to pip/python3 when uv is not on PATH`
**verify:** `go test ./components/deploy/ -run "TestPythonStartCommand|TestResolvePythonBin" -count=1`
