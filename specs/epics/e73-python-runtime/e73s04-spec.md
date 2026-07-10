# e73s04 — System Dependencies + Long-Running Subprocesses

## Summary

Parse `[tools.system_deps]` from pyproject.toml to install apt packages before
the uv sync step. Add `background_processes` to the Spec struct and extend the
Supervisor to track multiple subprocess PIDs.

## Motivation

Python apps often require system-level dependencies (git, curl, ssh, ffmpeg,
libpq-dev for psycopg2) that are not available in the BigBase sandbox. Background
processes (workers, schedulers, task queues) are fundamental to real Python
applications yet currently unsupported.

## Changes

### 1. System dependencies — `components/deploy/manifest.go`

- Parse `[tools.system_deps]` section from pyproject.toml manifest
- Add `SystemDeps []string` field to Manifest struct
- Run `apt-get update && apt-get install -y <deps>` before `uv sync`
- Sandbox: only allowlisted packages (git, curl, ssh, libpq-dev, libssl-dev,
  build-essential, ffmpeg, imagemagick)

### 2. Background processes — `components/deploy/runner.go`

- Add `BackgroundProcesses []string` to Spec struct
- Each entry is a shell command to run as a background process

### 3. Multi-process tracking — `components/deploy/supervisor.go`

- Extend Supervisor to track 1 main process + N background subprocesses
- When the main process exits, signal all background processes to stop
- When a background process exits, restart it (same crash-loop policy as main)
- Track PIDs for each background process

## Verify

```
go test ./components/deploy/... -run "SystemDep\|Background\|MultiPro"
```

## Depends On: e73s01

## BCP: 2
