# BigBase — Gemini CLI

Read CONVENTIONS.md before any GitHub or git operation.

## Project
Single-binary, component-based BaaS platform using Entity-Component-Construct (ECC) architecture.
Stack: Go 1.22+ / ECC Kernel + Plugins / SQLite + PostgreSQL

## Commands
| Action | Command |
|--------|---------|
| Run    | `go run .` |
| Test   | `go test ./...` |
| Build  | `go build -o bigbase .` |
| Lint   | `golangci-lint run ./...` |
| Test coverage | `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` |

## Architecture
ECC pattern: Kernel (discovery, lifecycle, event bus, config merge) + pluggable components (proxy, auth, db, api, storage, git, forge, cici, functions, realtime, messaging, deploy, admin, monitoring). Components communicate via event hooks, not direct imports.

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

## Specs (bigpowers YAML)

Read `specs/state.yaml`, `specs/release-plan.yaml`, and the active epic under `specs/epics/`. Architecture: `specs/plans/TECH_STACK_LATEST.md`.

## Agent Rules
- Read specs YAML (not `specs/archive/`) before writing code.
- All planning specifications MUST be written under `specs/` before code.
- Write minimum code that solves the stated problem.
- Run tests after every change. Show evidence before declaring done.

## Model Routing Matrix (Gemini)

### 1. Model Matrix & Allocation

| Task Category | Optimal Model | Selection Reason |
| :--- | :--- | :--- |
| **Global Planning & ADRs** | `Gemini 3.1 Pro (High)` | Deep reasoning, complex design trade-offs, and architectural planning. |
| **Codebase Context Search** | `Gemini 3.1 Pro (Low)` | 2M+ context window. Best for reading large file trees and cross-component structures. |
| **Feature Coding & TDD Loops** | `Gemini 3.1 Pro (High)` | Precision coding, exact syntax execution, and deep test writing. |
| **Verification & Utility Tasks** | `Gemini 3.5 Flash (High/Medium)` | Ultra-low latency, cheap tokens. Best for running linters, tests, and compiling. |
| **Browser UI Testing** | `Gemini 3.5 Flash (Low)` | Fast image/visual processing for browser subagents. |
| **Structured Docs & Summaries** | `Gemini 3.5 Flash (Medium)` or `Gemini 3.1 Pro (Low)` | Strong at structured prose, reports, and YAML/JSON synthesis. |

### 2. Dynamic Delegation Protocol

When spawning sub-agents (via `delegate-task`, `dispatch-agents`, or `browser_subagent`):
- **For file system audits, logs inspection, and linting**: Use `Gemini 3.5 Flash (Low)` to keep token costs minimal.
- **For code generation/refactoring sub-tasks**: Use `Gemini 3.1 Pro (High)`.
- **Context Shaving**: Never pass entire files to sub-agents unless they are targets for modification. Pass only specific function signatures or YAML spec blocks to keep input tokens low.


## bts toolchain

`bts` is installed. Prefer its verbs over ad-hoc shell commands.

| Task | Command | Avoid |
|------|---------|-------|
| Search code | `bts find --print <pattern>` | grep / find / cat |
| Interactive search | `bts find <pattern>` | manual grep pipes |
| Compress for context | `bts compress <file>` or `cmd \| bts compress` | summarising by hand |
| Repo map | `bts map` | listing files by hand |
| Library docs | `bts docs <lib>` | guessing from training data |
| Package source | `bts src <pkg>` | git clone |
| Toolchain health | `bts doctor` | which / command -v |

**Rules**
- Search with `bts find` before opening files to locate a symbol or pattern.
- Pipe anything > 200 lines through `bts compress` before adding to context.
- Run `bts map` when asked for a repo overview.
- Use `bts docs <lib>` before answering questions about library APIs.
- If a tool is missing, say so and run `bts doctor` — do not silently substitute.
