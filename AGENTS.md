# BigBase — OpenCode

Read CONVENTIONS.md before any GitHub or git operation.

## Project
Single-binary, component-based BaaS platform using Entity-Component-Construct (ECC)
architecture. Stack: Go 1.22+ / ECC Kernel + Plugins / SQLite + PostgreSQL.

## Commands
| Action | Command |
|--------|---------|
| Run (serve) | `go run . serve [--port PORT] [--db PATH] [--google-client-id ID] [--google-client-secret SECRET]` |
| Run (CLI)   | `go run . status` / `go run . version` / `go run . components list` |
| Test   | `go test ./...` |
| Build  | `go build -o bigbase .` |
| Lint   | `golangci-lint run ./...` |
| Test coverage | `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` |
| Setup  | `bash scripts/setup.sh` (idempotent) |

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
| Health check | `curl http://localhost:9999/health` |
| Component status | `go run . status` |
| List components | `go run . components list` |
| Monitoring | `curl http://localhost:9999/api/monitoring/logs` (auth required) |

Logging is structured JSON via `slog.JSONHandler` in serve mode.
CLI output uses plain text.

## Agent Rules
- Read specs/ before writing code.
- All planning and specifications MUST be written to `specs/` before code.
- Write minimum code that solves the stated problem.
- Run tests after every change. Show evidence before declaring done.
