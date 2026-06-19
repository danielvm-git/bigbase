# BUG-2026-06-19T173242: Python deploys fail on VPS — pip blocked by PEP 668

## Problem

On the production VPS (bigbase.click), deploying a Python site fails during the
build phase with:

```
exec: "pip": executable file not found in $PATH
```

After installing pip, it fails again with:

```
error: externally-managed-environment

This environment is externally managed. To install Python packages
system-wide, try apt install python3-xyz. If you wish to install a
non-Debian-packaged Python package, create a virtual environment or
pass --break-system-packages.
```

The deploy component runs `pip install -r requirements.txt` blindly, which
Ubuntu 24.04's PEP 668 enforcement blocks without `--break-system-packages`.

A secondary issue from the same log batch: the alert checker repeatedly fails
with `SQL logic error: no such column: duration_seconds` because the production
DB's `monitoring_alerts` table was created before that column was added.

## Root Cause Analysis

### Reproduce

Deploy a Python site to the production VPS. The build fails at the `pip install`
step. The alert checker error appears every 30 seconds in the server logs.

### Isolate

1. **Python deploy**: The build command for Python apps hardcodes
   `pip install -r requirements.txt` without the `--break-system-packages` flag
   required by Ubuntu 24.04's externally-managed Python environment.

2. **Alert checker**: The monitoring component queries `duration_seconds` from
   `monitoring_alerts`, but the production table was created by an older
   migration that didn't include this column. The `CREATE TABLE IF NOT EXISTS`
   pattern silently skips the migration because the table already exists.

### Hypothesize

1. Adding `--break-system-packages` to the pip install command will allow
   Python deploys to succeed on Ubuntu 24.04.
2. Adding an `ALTER TABLE ADD COLUMN` migration for `duration_seconds` with
   duplicate-column handling will fix the alert checker.

### Verify

Root cause confirmed: the deploy package hardcodes a pip invocation incompatible
with Ubuntu 24.04, and the monitoring package lacks a column migration.

Risk level: **High** (blocks all Python deployments on production).

## TDD Fix Plan

1. **RED**: Deploy a Python site to a test server without `--break-system-packages`.
   **GREEN**: Add `--break-system-packages` to the pip install args in the
   build command.
   **verify**: `go test ./components/deploy/ -run TestBuild -v`

2. **RED**: Run the alert checker against a DB where `monitoring_alerts` lacks
   `duration_seconds`.
   **GREEN**: Add an `ALTER TABLE ADD COLUMN` migration in Start() that
   silently ignores "duplicate column" errors.
   **verify**: `go test ./components/monitoring/ -run TestAlert -v`

## Acceptance Criteria

- [ ] Python deploys succeed on Ubuntu 24.04 VPS
- [ ] Alert checker stops logging "no such column: duration_seconds"
- [ ] Existing deploy and monitoring tests still pass
- [ ] No regression for Go or Node.js deploys

## Resolution

<!-- filled in by validate-fix -->
