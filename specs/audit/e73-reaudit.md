# e73 Python Runtime — Phase-2 Re-Audit

**Date:** 2026-08-15
**Baseline:** `main` @ `9f0d4e505`.
**Verdict:** **Already shipped.** All 4 stories were implemented and green on `main`; the
`planned`/`pending` status was stale (same pattern as e75/e39/e40). One real test gap closed.

## Story-by-story evidence

| Story | Acceptance criteria | State |
|-------|---------------------|-------|
| **e73s01** pyproject detection + uv | `DetectAppType` finds `pyproject.toml`; `uv sync --frozen` build; `[project.scripts]` parsing; `InitManifest` uv-aware | DONE — `python.go` (`HasPyProjectTOML`, `ParsePyProjectTOML`, `ParseScripts`), `engine.go:478-498`; tests `TestDetectAppType_PyProject*`, `TestParsePyProjectTOML_EntryPoint`, `TestInitManifest` |
| **e73s02** uvicorn/ASGI runtime | detect uvicorn dep; `uv run uvicorn` start cmd; `python app.py` fallback | DONE — `pythonStartCommand`, `ASGIImport`; tests `TestIsUvicornDep`, `TestPyProjectTOML_HasUvicorn`, `TestPythonStartCommand_Uvicorn*`, `TestPythonStartCommand_Fallback` |
| **e73s03** writable disk + health polling | per-deploy writable dir; poll `/health`; restart after 3 failures; emit `deploy.health_failed` | DONE — `engine.go:635` writable dir + `WRITABLE_DIR`; `supervisor.go` `healthLoop` (interval poll, `healthFailureThreshold=3`, `deploy.health_failed` + `inst.Stop`). **Gap closed:** the health-poll loop had no direct unit test — added `TestSupervisorHealthLoop_RestartsAfterThreshold` + `_HealthyNeverRestarts` (supervisor_health_test.go). Required making `healthPollInterval` a test-overridable `var`. |
| **e73s04** system deps + background procs | `SystemDeps` manifest field; `apt-get install` before build; package allowlist; `BackgroundProcesses` + multi-process supervision | DONE — `python.go:141 SystemDeps()` + `allowedSystemDeps` allowlist; `engine.go:481-487` apt; `deploy_runner.go:62` background spawn; tests `TestPyProjectTOML_SystemDeps*`, `TestAllowedSystemDep`, `TestSpec_BackgroundProcesses` |

Battle-tested in production via the Grimoire reference app (FastAPI/uv/SQLite); the pre-e73
Python bugs (uvicorn spawn, uv-not-in-PATH, pip PEP-668) are all closed in the registry.

## Actions

- e73 epic + all 4 stories `planned`/`pending` → `done`.
- Added the missing s03 health-poll regression tests (2 tests, pass under `-race`).
- No production behavior changed (only `healthPollInterval` const → var as a test seam).
