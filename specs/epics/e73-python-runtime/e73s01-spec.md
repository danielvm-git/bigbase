# e73s01 — pyproject.toml Detection + uv Package Manager

## Summary

Extend `DetectAppType()` to recognize `pyproject.toml` as a Python project signal,
and add a `uv sync --frozen` build step for deterministic package installation.

## Motivation

Current Python detection only checks for `app.py` or `main.py` at the repo root.
Modern Python projects use `pyproject.toml` (PEP 518/621) with `uv` for dependency
management. Without `pyproject.toml` detection, Grimoire-class FastAPI apps cannot
deploy on BigBase.

## Changes

### 1. DetectAppType() — `components/deploy/deploy.go`

- After existing `package.json` and `go.mod` checks, before `app.py`/`main.py`:
  - Check for `pyproject.toml` in buildDir
  - If present and contains `[project.scripts]` or `[tool.uv]` section, return `AppPython`

### 2. Build step — `components/deploy/deploy.go`

- When `AppPython` and `pyproject.toml` exists:
  - Run `uv sync --frozen` in the build directory
  - Set `VIRTUAL_ENV` and `PATH` for the uv-managed venv

### 3. InitManifest() — `components/deploy/manifest.go`

- When `pyproject.toml` exists (in addition to existing `app.py`/`main.py` checks):
  - Set `framework: python`
  - Set `build.command: uv sync --frozen`
  - Parse `[project.scripts]` for start command
  - Set `start.port: 8000`

## Verify

```
go test ./components/deploy/... -run "Python\|Detect\|pyproject"
golangci-lint run ./components/deploy/...
```

## BCP: 2
