<!-- THIN ADAPTER — canonical engineering rules live in AGENTS.md -->
<!-- Edit AGENTS.md, not this file. Gemini CLI-specific additions only below. -->

# BigBase — Gemini CLI

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

## Model Routing Matrix (Gemini)

| Task Category | Optimal Model | Reason |
| :--- | :--- | :--- |
| **Global Planning & ADRs** | `Gemini 3.1 Pro (High)` | Deep reasoning, architecture trade-offs |
| **Codebase Context Search** | `Gemini 3.1 Pro (Low)` | 2M+ context window for large file trees |
| **Feature Coding & TDD Loops** | `Gemini 3.1 Pro (High)` | Precision coding, exact syntax execution |
| **Verification & Utility Tasks** | `Gemini 3.5 Flash (High/Medium)` | Ultra-low latency for lint/test/compile |
| **Browser UI Testing** | `Gemini 3.5 Flash (Low)` | Fast image/visual processing |
| **Structured Docs & Summaries** | `Gemini 3.5 Flash (Medium)` | Structured prose, YAML/JSON synthesis |
