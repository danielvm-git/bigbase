<!-- THIN ADAPTER — canonical engineering rules live in AGENTS.md -->
<!-- Edit AGENTS.md, not this file. Claude-specific additions only below. -->

# BigBase — Claude Code

Read CONVENTIONS.md before any GitHub or git operation.
Read AGENTS.md for the full engineering rule set (architecture, ctxo, bts, opensrc, VPS, Orca).

## Project
Single-binary, component-based BaaS platform using Entity-Component-Construct (ECC) architecture.
Stack: Go 1.26.3 / ECC Kernel + Plugins / SQLite + PostgreSQL

## Commands
| Action | Command |
|--------|---------|
| Run    | `go run .` |
| Test   | `go test ./...` |
| Build  | `go build -o bigbase .` |
| Lint   | `golangci-lint run ./...` |
| Test coverage | `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` |

## Specs (bigpowers YAML)
Read `specs/state.yaml`, `specs/release-plan.yaml`, and the active epic under `specs/epics/`.
Architecture: `specs/tech-architecture/tech-stack.md`. Legacy: `specs/archive/` only.

## Agent Rules
- Read specs YAML (not `specs/archive/`) before writing code.
- All planning specifications MUST be written under `specs/` before code.
- Write minimum code that solves the stated problem.
- Run tests after every change. Show evidence before declaring done.

## Model Routing Matrix (Claude)

| Task Category | Optimal Model | Reason |
| :--- | :--- | :--- |
| **Global Planning & ADRs** | `Claude Opus 4.6 (Thinking)` | Deep reasoning, architecture trade-offs |
| **Codebase Context Search** | `Claude Sonnet 4.6` | Balanced context window + speed |
| **Feature Coding & TDD Loops** | `Claude Sonnet 4.6 (Thinking)` | Precision coding, exact syntax |
| **Verification & Utility Tasks** | `Claude Haiku 3.5` | Low latency for lint/test/compile |
| **Browser UI Testing** | `Claude Haiku 3.5` | Fast image/visual processing |

<!-- BEGIN sqz-claude-guidance (auto-installed by sqz init; remove this block to disable) -->

## sqz — Context Compression (READ FIRST)

sqz is installed in this project. Prefer `sqz_read_file`, `sqz_grep`, `sqz_list_dir` MCP tools
over the built-in Read/Grep/Glob for any file > ~2KB. Repeat reads return a `§ref:HASH§` token.

<!-- END sqz-claude-guidance -->
