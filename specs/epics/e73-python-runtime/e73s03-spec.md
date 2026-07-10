# e73s03 — Writable Disk + Health Check Polling

## Summary

Allocate a persistent writable directory per Python deployment and add
runtime health polling to the Supervisor. The Supervisor polls GET /health
on the deployment's port and triggers a restart after 3 consecutive failures.

## Motivation

Python web apps need writable disk for logs, SQLite databases, uploaded files,
and temporary processing. Without persistence, data is lost on restart.
Health polling ensures crashed-but-not-exited apps are detected and restarted.

## Changes

### 1. Persistent writable dir — `components/deploy/deploy_runner.go`

- Allocate a deployment-scoped writable directory under `data/writable/<deployID>/`
- Pass writable directory path to the deployment process as `WRITABLE_DIR` env var
- Create the directory during build step if it doesn't exist

### 2. Health polling — `components/deploy/supervisor.go`

- Add health polling goroutine per supervised instance
- Poll `GET http://localhost:<port>/health` every 10 seconds
- After 3 consecutive non-200 responses or connection failures, call `Stop()` and
  let the Supervisor restart loop kick in
- Emit `deploy.health_failed` event on threshold trip

### 3. Health config — `components/deploy/manifest.go`

- Add `health_check` to the existing Manifest struct (already present)
- Python apps default to `GET /health`, 3 consecutive failures before restart

## Verify

```
go test ./components/deploy/... -run "Health\|Supervisor\|Writable"
```

## Depends On: e73s01

## BCP: 2
