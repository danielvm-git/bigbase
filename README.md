# BigBase

[![Site](https://img.shields.io/badge/Site-bigbase.click-4f46e5)](https://bigbase.click)
![License](https://img.shields.io/badge/License-MIT-yellow.svg)
![Version](https://img.shields.io/badge/version-2.10.0-blue.svg)
![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)

> Single-binary, component-based BaaS platform using Entity-Component-Construct (ECC) architecture.

[BigBase](https://bigbase.click) is an open-source Backend-as-a-Service platform designed for speed, portability, and modularity. It packs Auth, Database, Storage, Functions, Messaging, and Git Repos into a single Go binary that runs anywhere—from your local machine to a VPS—with zero external dependencies.

Built on the **Entity-Component-Construct (ECC)** pattern, BigBase is highly extensible. Each feature is a pluggable component that communicates via a decoupled event bus, making the system easy to customize and scale from SQLite to PostgreSQL.

## Prerequisites

- **Runtime**: Go 1.22+
- **Database**: SQLite (default) or PostgreSQL
- **Linter**: `golangci-lint` (for development)

## Quick Start

```bash
# Clone and run immediately
git clone https://github.com/danielvm-git/bigbase
cd bigbase
go run . serve --port 8080
```

Open [http://localhost:8080](http://localhost:8080) to access the landing page, or [http://localhost:8080/admin/](http://localhost:8080/admin/) for the dashboard.

## Features

- **ECC Architecture**: Modular "kernel + components" design for maximum decoupling.
- **Built-in Auth**: Email/password, Google OAuth, and JWT session management.
- **Data Studio**: Browse and query your SQLite/PostgreSQL data via SQL Editor.
- **Git Repos & Deploy**: Host your own Git repositories and trigger CI/CD pipelines.
- **Functions**: Serverless JavaScript runtime for custom backend logic.
- **Realtime**: Live event broadcasting via WebSockets.

## Development

BigBase follows a strict TDD and Spec-Driven Development workflow.

```bash
# Run tests
go test ./...

# Build binary
go build -o bigbase .

# Linting
golangci-lint run ./...

# Coverage report
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out
```

## Configuration

BigBase uses a "config merge" strategy (Flags > Env > JSON).

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--port` | `PORT` | `8080` | HTTP server port |
| `--db` | `DB_PATH` | `bigbase.db` | SQLite database path |
| `--sites-domain` | `SITES_DOMAIN` | | Domain for deployed sites |

## Architecture

BigBase is composed of a central **Kernel** and pluggable **Components**.

- **Kernel**: Manages component lifecycles, event bus, and shared database connections.
- **Components**: `auth`, `db`, `api`, `storage`, `git`, `deploy`, `proxy`, `realtime`, `messaging`.
- **Communication**: Components emit and subscribe to events; they **never** import each other directly.

See [specs/plans/TECH_STACK_LATEST.md](specs/plans/TECH_STACK_LATEST.md) for deeper architectural details.

## Contributing

1. Read [CONVENTIONS.md](CONVENTIONS.md) for coding standards.
2. Fork the repo and create a feature branch (`fix/BUG-id` or `feat/story-id`).
3. Commit using [Conventional Commits](https://www.conventionalcommits.org/).
4. Open a Pull Request for review.

## License

MIT — see [LICENSE](LICENSE) for details.

## Credits

Built with [bigpowers](https://github.com/danielvm-git/bigpowers).
