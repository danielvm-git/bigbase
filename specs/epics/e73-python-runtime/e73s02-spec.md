# e73s02 — ASGI Server (uvicorn) Runtime

## Summary

Detect uvicorn in pyproject.toml dependencies and generate the correct
`uv run uvicorn <app>:<app> --host 0.0.0.0 --port $PORT` start command.
Fall back to `python3 app.py` when uvicorn is not present.

## Motivation

Modern Python web apps use ASGI servers (uvicorn) rather than the stdlib
HTTP server. BigBase must auto-detect uvicorn and configure it correctly
so apps start without manual configuration.

## Changes

### 1. getStartCommand / StartCmd — `components/deploy/deploy.go`

- Parse pyproject.toml `[project.dependencies]` and `[project.optional-dependencies]`
  for `uvicorn`
- If uvicorn found, determine the app module from `[project.scripts]` or
  auto-detect the module containing the FastAPI/Starlette app
- Generate: `uv run uvicorn <module>:<app> --host 0.0.0.0 --port $PORT`

### 2. Fallback — `components/deploy/deploy_runner.go`

- When no uvicorn detected, fall back to `python3 app.py`
- This preserves backward compatibility with existing `app.py`/`main.py` patterns

## Verify

```
go test ./components/deploy/... -run "StartCommand\|uvicorn\|Python"
```

## Depends On: e73s01

## BCP: 2
