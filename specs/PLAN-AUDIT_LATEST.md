# Plan Audit — e75: Deploy Hardening (Log Eviction + Flaky Tests)

**Date:** 2026-07-10 · **Verdict:** READY — all 4 gaps closed

## Principles Alignment

| Check | Status | Note |
|-------|--------|------|
| Vertical slices | ⚠️ | e75s01 is a true vertical slice (struct refactor + FIFO logic + test). e75s02 groups three unrelated flaky fixes into one story — each is vertical on its own but the story isn't |
| Scope bounded | ❌ | No `in_scope` / `out_of_scope` block. `covers:` lists only 3 files but the log eviction refactor will touch `logs.go` internals, `Deploy` struct fields, `appendDeployLog`, `getDeployLogs`, `deleteDeployLogs`, and their tests |
| Success criteria defined | ⚠️ | Each task has a `verify:` command, but flaky-test tasks use `-count=5 -v` which is a smoke check, not a statistical proof of fix. Flaky test success criteria should be: "passes 10 consecutive parallel runs with -count=1" |
| HARD GATE candidates | ❌ | No gates defined. Candidates: (a) before e75s01 — ensure existing log tests still pass with new FIFO struct, (b) before e75s02 — gate each flaky fix on `-count=10` stability |
| Domain language | ✅ | "FIFO eviction", "deterministic log retention", "port race", "concurrent test interference" — consistent with tech-stack.md domain terms |

## Conventions Completeness

| Check | Status | Note |
|-------|--------|------|
| AGENTS.md | ✅ | ctxo usage, git discipline, tool rules all defined |
| CONVENTIONS.md | ✅ | ECC pattern, Go conventions, naming, testing, security |
| specs/ layout | ✅ | `specs/epics/`, `specs/adr/`, `specs/tech-architecture/` |
| Commit conventions | ✅ | Conventional Commits + semantic-release |
| Git workflow | ✅ | Solo-git mode documented |

## Pre-flight Answers

| Question | Answer |
|----------|--------|
| **Test command** | `go test -count=1 ./...` |
| **Build command** | `go build ./...` |
| **Lint command** | `go vet ./...` |
| **Typecheck command** | `go build ./...` (Go build doubles as typecheck) |
| **CI platform** | GitHub Actions + semantic-release |
| **Solo or team** | Solo-git |
| **Language + framework** | Go 1.26 · SQLite · ECC (Entity-Component-Construct) |
| **Greenfield or existing** | Existing — BigBase (192+ Go source files, 19 components) |
| **Full verify** | `go test -count=1 ./... && go build ./... && go vet ./...` |

## Open Gaps

(All closed — 2026-07-10)

- [x] **Gap 1 — in_scope / out_of_scope block** added to e75 YAML. Scoped to logs.go methods + test file only; excludes gateway/engine/orchestrator/constants.
- [x] **Gap 2 — HARD GATEs** defined for both stories: (pre-e75s01) existing TestDeployLogs passes with new FIFO struct — no regressions; (post-e75s02) all three flaky tests pass 10 consecutive runs with 0 failures.
- [x] **Gap 3 — Flaky test verify** tightened from `-count=5` to `-count=10` with explicit "0 failures" criterion across all 3 tasks.
- [x] **Gap 4 — e75s02t3 ambiguity** resolved: Option A (mutex-protected `pickPort()` counter) selected. Code sketch included in task notes.

## Verdict

**READY** — proceed with `kickoff-branch` → `develop-tdd`

All 4 gaps closed in `specs/epics/e75-deploy-hardening.yaml`:
| # | Gap | Resolution |
|---|------|------------|
| 1 | No in_scope/out_of_scope | ✅ Added — 4 log methods in scope, gateway/engine/orch excluded |
| 2 | No HARD GATEs | ✅ Added — regression gate (pre-e75s01) + stability gate (post-e75s02) |
| 3 | Verify -count=5 too weak | ✅ Tightened to -count=10 for all 3 flaky test tasks |
| 4 | 4 ambiguous fix options | ✅ Option A selected — mutex-protected pickPort() counter |

## Recommended next skill

→ `kickoff-branch` → `develop-tdd`
