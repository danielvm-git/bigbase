# Threat Model — e73 Python Runtime (uv/uvicorn)

## Scope

Process execution surface for Python deployments using uv and uvicorn within
the BigBase deploy component. The sandbox is a Linux VPS with BigBase running
as a single binary — no container isolation.

## Assets

| Asset | Sensitivity | Exposure |
|-------|-------------|----------|
| Deployment process stdout/stderr | Low | Logged to DB, WebSocket streamed |
| Build directory (`data/builds/<id>/`) | Medium | Contains app source, secrets in env vars |
| Writable directory (`data/writable/<id>/`) | Medium | Runtime data, SQLite DBs, uploads |
| Port allocation (`:10000+`) | Low | Public via proxy, bound to localhost |
| Env vars (PORT, DB_DSN, WRITABLE_DIR) | Medium | Passed to subprocess |

## Threats

### T1: Malicious pyproject.toml command injection
- **Vector**: `[project.scripts]` contains shell-injected commands
- **Impact**: Arbitrary code execution during build (uv sync) or runtime (uvicorn)
- **Mitigation**: No shell interpretation — use `exec.Cmd` with argument list, not `sh -c`.
  uv and uvicorn are invoked via `exec.CommandContext(ctx, "uv", "run", "uvicorn", ...)` —
  no shell metacharacter expansion.
- **Status**: ACCEPTED — uv/uvicorn run in the same trust domain as npm/node/go builds.

### T2: uv dependency supply chain
- **Vector**: Malicious PyPI package pulled by `uv sync --frozen`
- **Impact**: RCE in deployment subprocess
- **Mitigation**: `--frozen` flag ensures lockfile integrity (no automatic upgrades).
  Users control their own pyproject.toml and uv.lock. Same threat model as npm install.
- **Status**: ACCEPTED — no BigBase-layer mitigation.

### T3: Port binding race
- **Vector**: Two deployments allocated the same port
- **Impact**: Port conflict, deployment failure
- **Mitigation**: Existing port allocator (`nextPort` incremented atomically). No change.
- **Status**: MITIGATED — existing allocation is safe.

### T4: Disk exhaustion from writable directories
- **Vector**: Python app writes unbounded data to writable disk
- **Impact**: VPS disk fill, BigBase crash
- **Mitigation**: ACCEPTED — writable dirs are user-managed. Same risk as build caches.
- **Status**: ACCEPTED — deferred to operational concern.

### T5: Health endpoint information disclosure
- **Vector**: GET /health exposes app internals
- **Impact**: Information leakage about app version, dependencies
- **Mitigation**: `/health` is on localhost port (not proxy-exposed). Only Supervisor polls it.
- **Status**: MITIGATED — localhost-only by design.

## Residual Risk

**Medium**. Python process execution adds no net-new attack surface beyond existing
Node.js/Go deployment paths. All builds run with the same user privileges.
The primary risk is supply-chain (PyPI packages), which is user-owned.
