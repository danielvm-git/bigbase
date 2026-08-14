# BigBase

[![Site](https://img.shields.io/badge/Site-bigbase.click-4f46e5)](https://bigbase.click)
![License](https://img.shields.io/badge/License-MIT-yellow.svg)
![Version](https://img.shields.io/badge/version-2.89.0-blue.svg)
![Go](https://img.shields.io/badge/Go-1.26.3-00ADD8?logo=go)

> Single-binary, component-based BaaS platform using Entity-Component-Construct (ECC) architecture.

BigBase is an open-source Backend-as-a-Service platform built with Go. It packs Auth, Database, Storage, Functions, Messaging, Git Repos, CI/CD, Realtime, and Site hosting into a single binary — zero external dependencies required.

Built on the **ECC pattern**: a kernel discovers, starts, and connects independent components via an event bus. No direct imports between components.

## Prerequisites

- **Runtime**: Go 1.26.3+
- **Database**: SQLite (default, zero-CGO via `modernc.org/sqlite`) or PostgreSQL
- **Linter**: `golangci-lint` (development only)

## Declared toolchain contract

The deploy runtime assumes certain binaries exist on the build/VPS host
(`node`, `npm`, `pnpm`, `python3`, `pip`, `uv`, `go`, `git`). Historically a
"missing tool on deploy host" defect recurred repeatedly (#179) because each
tool was added as a one-off patch in a Go code path. These tools are now
**declared** in [`toolchain.toml`](toolchain.toml) — the single source of truth
for the deploy-host toolchain.

`scripts/verify-toolchain.sh` reads the contract and fails with a clear
`TOOLCHAIN_MISSING:<tool>` / `TOOLCHAIN_VERSION_TOO_LOW:<tool>` message when a
required binary is absent or below its declared minimum:

```bash
bash scripts/verify-toolchain.sh
```

The `toolchain-parity` CI job (`.github/workflows/test-build-release.yml`) provisions the
same tools `scripts/setup-vps.sh` installs on the VPS and then runs the
verifier — so a tool missing in CI is a tool that would be missing on the VPS,
caught pre-deploy instead of mid-build.

**Adding a tool**: when you add a new `exec.Command`/`exec.LookPath` call in
`components/deploy/`, add an entry under `[tools.required]` (always needed) or
`[tools.optional]` (only for some app types) in `toolchain.toml`, set `min` to
the real floor, and CI enforces it automatically.

## Quick Start

```bash
git clone https://github.com/danielvm-git/bigbase
cd bigbase
go run . serve --port 8080
```

Open [http://localhost:8080](http://localhost:8080) for the landing page, or [http://localhost:8080/admin/](http://localhost:8080/admin/) for the admin dashboard.

## CLI Commands

| Command | Description |
|---------|-------------|
| `bigbase serve` | Start HTTP server (default port 8080) |
| `bigbase status` | Show kernel and component status |
| `bigbase components list` | List registered components |
| `bigbase version` | Show version |
| `bigbase init` | Generate default `bigbase.yaml` manifest |
| `bigbase deploy` | Deploy a git repo via the API |
| `bigbase backup` | Dump SQLite database to SQL file |
| `bigbase restore` | Replay SQL dump into SQLite database |
| `bigbase migrate up\|down\|status` | Run database migrations |

## Features

- **ECC Architecture**: Modular kernel + 19 pluggable components, event-bus communication.
- **Multi-Tenant Auth**: Email/password, Google OAuth, API keys, JWT, orgs, invites, roles.
- **Auto CRUD + SQL**: Collection-based REST API with filter/sort, SQL editor, PostgreSQL dual-driver.
- **File Storage**: Upload/download/delete with MIME detection.
- **Git Repos**: Bare repo management, SSH clone, GitHub App integration.
- **Serverless Functions**: JavaScript runtime (goja) with fetch, DB, env, request context, cron scheduling.
- **CI/CD Pipelines**: Git push → workflow engine → deploy.
- **Realtime**: WebSocket subscriptions with mutation event broadcasting.
- **Messaging**: Email (SMTP), SMS, Push, Webhook, Telegram providers.
- **Site Hosting**: Deploy web apps from GitHub, custom domains with TLS.
- **Monitoring**: Metrics (Prometheus), structured JSON logs, alerts, health checks, SSE events.
- **Production Hardening**: Three-layer security model — Ubuntu OS (UFW + fail2ban + unattended-upgrades), BigBase (systemd + alerts + backups), Contabo VPS (snapshots + health checks). See `.agents/skills/harden-vps/SKILL.md`.
- **Admin UI**: Embedded React SPA — 8 pages, dark mode, responsive.
- **MCP Server**: Model Context Protocol (SSE and stdio transports) for AI tool integration.

## Configuration

Config merge priority: CLI flags > environment variables > defaults.

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--port` | — | `8080` | HTTP server port |
| `--db` | — | `bigbase.db` | SQLite database path (legacy) |
| `--db-driver` | `BIGBASE_DB_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `--db-dsn` | `BIGBASE_DB_DSN` | (from `--db`) | DSN for database connection |
| `--google-client-id` | `GOOGLE_CLIENT_ID` | — | Google OAuth client ID |
| `--google-client-secret` | `GOOGLE_CLIENT_SECRET` | — | Google OAuth client secret |
| `--github-app-id` | `GITHUB_APP_ID` | — | GitHub App ID |
| `--github-app-slug` | `GITHUB_APP_SLUG` | — | GitHub App slug |
| `--github-app-private-key-path` | `GITHUB_APP_PRIVATE_KEY_PATH` | — | GitHub App private key path |
| `--github-webhook-secret` | `GITHUB_WEBHOOK_SECRET` | — | GitHub webhook secret |
| `--sites-domain` | `BIGBASE_SITES_DOMAIN` | — | Parent domain for deployed sites |
| `--log-level` | — | `info` | Log level: debug, info, warn, error |
| `--cors-allowed-origins` | — | — | Comma-separated CORS origins |
| `--auth-post-login-redirect` | — | `/admin/` | OAuth post-login redirect |
| `--auth-spa-origin-allowlist` | — | — | Allowed SPA origins for OAuth tokens |
| `--mcp-disabled` | — | `false` | Disable MCP server |
| `--mcp-port` | — | `3900` | MCP server port |
| `--mcp-transport` | — | `http` | MCP transport: stdio, http |

## Architecture

BigBase = **Kernel** + **19 Components**. Components communicate exclusively via the event bus — no direct imports between components.

### Component Catalog

| # | Component | Path | Purpose |
|---|-----------|------|---------|
| — | Kernel | `kernel/` | Discovery, lifecycle, event bus, config merge |
| 1 | Proxy | `components/proxy/` | HTTP server, routing, landing page |
| 2 | DB | `components/db/` | Database access, migrations (SQLite + Postgres) |
| 3 | API | `components/api/` | Auto CRUD, SQL endpoint, filter/sort, org scoping |
| 4 | Auth | `components/auth/` | Register, login, JWT, Google OAuth, API keys, orgs |
| 5 | Admin | `components/admin/` | Embedded SPA server (8 pages, dark mode) |
| 6 | Storage | `components/storage/` | File upload/download/delete |
| 7 | Git | `components/git/` | Bare repo management, SSH clone |
| 8 | Forge | `components/forge/` | Issues, labels, kanban, wiki |
| 9 | CICI | `components/cici/` | Pipeline/workflow engine |
| 10 | Functions | `components/functions/` | JS runtime (goja) with fetch, db, cron |
| 11 | Realtime | `components/realtime/` | WebSocket subscriptions, event broadcast |
| 12 | Messaging | `components/messaging/` | Email/SMS/Push/Webhook/Telegram |
| 13 | Deploy | `components/deploy/` | Build/run/delete web apps, TLS |
| 14 | Monitoring | `components/monitoring/` | Metrics, logs, alerts, health, SSE |
| 15 | Sites | `components/sites/` | Deploy from GitHub, logs UI |
| 16 | GitHub | `components/github/` | GitHub App auth, repo listing |
| 17 | Webhooks | `components/webhooks/` | Outbound webhook delivery, retry |
| 18 | Backup | `components/backup/` | Backup/restore, migration tooling |
| 19 | MCP | `components/mcp/` | Model Context Protocol (SSE/stdio) |

See [specs/tech-architecture/tech-stack.md](specs/tech-architecture/tech-stack.md) for the full architecture diagram and event flow.

## Development

```bash
# Run tests
go test ./...

# Build binary
go build -o bigbase .

# Linting
golangci-lint run ./...

# Coverage report
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

# Preflight (build gate)
npm run preflight
```

## Contributing

1. Read [CONVENTIONS.md](CONVENTIONS.md) for coding standards and conventions.
2. Read [AGENTS.md](AGENTS.md) for the agentic development workflow.
3. Fork, create a feature branch (`fix/BUG-id` or `feat/story-id`).
4. Commit using [Conventional Commits](https://www.conventionalcommits.org/).
5. Open a Pull Request.

## License

MIT — see [LICENSE](LICENSE) for details.
