# BigBase — OpenCode

Read CONVENTIONS.md before any GitHub or git operation.

## Project
Single-binary, component-based BaaS platform using Entity-Component-Construct (ECC)
architecture. Stack: Go 1.22+ / ECC Kernel + Plugins / SQLite + PostgreSQL.

## Commands
| Action | Command |
|--------|---------|
| Run (serve) | `go run . serve [--port PORT] [--db PATH] [--sites-domain DOMAIN] [--google-client-id ID] [--google-client-secret SECRET]` |
| Run (CLI)   | `go run . status` / `go run . version` / `go run . components list` |
| Test   | `go test ./...` |
| Build  | `go build -o bigbase .` |
| Lint   | `golangci-lint run ./...` |
| Test coverage | `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` |
| Setup  | `bash scripts/setup.sh` (idempotent) + `bash scripts/setup-newrelic.sh` (monitoring) |

## Architecture
ECC pattern: Kernel (discovery, lifecycle, event bus, config merge) + pluggable
components (proxy, auth, db, api, storage, git, forge, cici, functions, realtime,
messaging, deploy, admin, monitoring). Components communicate via event hooks,
not direct imports.

All 14 slices implemented. 7 admin UI pages built. Google OAuth social login
via embedded relay (no user-owned Google app required).

## Conventions
- Go standard layout: `kernel/`, `components/<name>/`, `config/`
- Component interface: `Init(ctx, config) → Start(ctx) → Stop(ctx)`
- Event bus for cross-component communication (no direct imports)
- TDD workflow: write test first, see it fail, implement, verify green
- All planning output in `specs/`

## Never
- Hardcode secrets, API keys, or tokens
- Mutate state directly — use spread/immutable patterns
- Commit to main without PR
- Expose internal errors or stack traces to API clients
- Use `any` in Go — prefer concrete types or interfaces

## Observability

| What | Command |
|------|---------|
| View logs | `go run . serve --port 9999` then curl `/health` (JSON to stdout) |
| Health check | `bash scripts/health-check.sh` or `curl http://localhost:9999/health` |
| Component status | `go run . status` |
| List components | `go run . components list` |
| Monitoring | `curl http://localhost:9999/api/monitoring/logs` (auth required) |
| Metrics (Prometheus) | `curl http://localhost:9999/api/monitoring/metrics/prometheus` |
| New Relic (host metrics) | `bash scripts/setup-newrelic.sh` (requires API key + account ID) |
| New Relic dashboard | https://one.newrelic.com > Infrastructure > Hosts |

Logging is structured JSON via `slog.JSONHandler` in serve mode.
CLI output uses plain text. All components log with `key=value` pairs.

## Specs (bigpowers YAML)

| File | Purpose |
|------|---------|
| `specs/state.yaml` | Session flow, git, handoff, active epic |
| `specs/release-plan.yaml` | Epic index (WSJF); no story status |
| `specs/execution-status.yaml` | Story/epic status (sync via `scripts/sync-status-from-epics.sh`) |
| `specs/epics/eNN-*.yaml` | Tasks with `verify:` commands |
| `specs/plans/TECH_STACK_LATEST.md` | Architecture and stack |

Legacy markdown: `specs/archive/` only — not source of truth when YAML exists.

## Agent Rules
- Read `specs/state.yaml` and the active epic shard before writing code.
- All planning output goes under `specs/` (YAML + `plans/`); run `bash scripts/validate-specs-yaml.sh` after spec edits.
- Write minimum code that solves the stated problem.
- Run tests after every change. Show evidence before declaring done.

## Agentic Stack (OpenCode)

### Commands
| Command | Purpose | Agent |
|---------|---------|-------|
| `/check-stack` | Verify agentic stack health (Go + UI + MCP/LSP wiring) | build-error-resolver |
| `/ship` | Push, PR, merge when preflight + CI pass | build |
| `/plan` | Create implementation plan | planner |
| `/tdd` | TDD workflow | tdd-guide |
| `/code-review` | Code quality review | code-reviewer |
| `/security` | Security review | security-reviewer |
| `/build-fix` | Fix build errors | build-error-resolver |
| `/e2e` | Generate and run E2E tests | e2e-runner |

### Preflight (build gate)
```bash
npm run preflight       # Go vet + tests + ui/dist check
npm run preflight:go    # Go-only checks
npm run preflight:ui    # UI build
npm run preflight:build # Go binary build
```

### opensrc learn-before-build
Before implementing against an unfamiliar dependency, run:
```bash
npx opensrc list
npx opensrc fetch github:org/repo  # if not cached
```
Read the opensrc cache path before changing integration code.

### Build and Release Loop
```text
/check-stack → npm run preflight → /ship
```
- Every push/PR runs CI (Go vet + tests + UI build)
- Merge to `main` triggers semantic-release via `release.yml`

### Observability
| What | Command |
|------|---------|
| View logs | `go run . serve --port 9999` then curl `/health` (JSON to stdout) |
| Health check | `curl http://localhost:9999/health` |
| Component status | `go run . status` |
| List components | `go run . components list` |
| Monitoring | `curl http://localhost:9999/api/monitoring/logs` (auth required) |

Logging is structured JSON via `slog.JSONHandler` in serve mode.
CLI output uses plain text.
