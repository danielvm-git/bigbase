# Plan Audit — BigBase e67 (MCP Provisioning Tools)
**Date:** 2026-07-07 · **Verdict:** NOT READY

Audited: `specs/epics/e67-mcp-provisioning-tools/` capsule (4 stories, 7 BCPs), `specs/release-plan.yaml` v3.0.0, and build-epic pre-flight gates.

---

## Principles Alignment

| Check | Status | Note |
|-------|--------|------|
| Vertical slices | ✅ | 4 independent stories (create_repo → create_site → provision_ci_credentials → get_ci_template); each shippable alone |
| Scope bounded | ⚠️ | Per-story §16/§18 Out of Scope present; **e67 not listed in `specs/product/SCOPE_LATEST.yaml`** (still v2.0 scope) |
| Success criteria | ✅ | Gherkin AC in all four story specs; every task has runnable `verify:` command |
| HARD GATE candidates | ✅ | `hard_gate: e67s03` in epic.yaml — site-scoped deploy keys + cross-site enforcement |
| Domain language | ⚠️ | Terminology consistent in specs (`site_id`, `bb_dep_`, `GitCreator`); **`specs/product/GLOSSARY_LATEST.yaml` absent** |

---

## Conventions Completeness

| Check | Status | Note |
|-------|--------|------|
| `CLAUDE.md` / `AGENTS.md` | ✅ | Present with Go commands, ECC rules, ctxo |
| `CONVENTIONS.md` | ✅ | Solo-git, Conventional Commits, semantic-release |
| `specs/` layout | ✅ | Epic capsule with `epic.yaml`, `*-spec.md`, `*-tasks.yaml` |
| Conventional Commits | ✅ | Documented; CI uses semantic-release |
| Git workflow mode | ✅ | Solo-git (`CONVENTIONS.md` §Git & Workflow) |
| Tech-stack doc | ✅ | `specs/tech-architecture/tech-stack.md` exists |
| release-plan.yaml | ✅ | e67 registered, WSJF 8.0, `depends_on: [e57]` |

---

## Pre-flight Answers

| Question | Value |
|----------|-------|
| Test command | `go test ./components/mcp/ -run Test<Story> -v -count=1` (per story) + `go test ./... -count=1` (full suite) |
| Build command | `go build .` |
| Lint command | `golangci-lint run ./...` |
| Typecheck command | `go vet ./...` |
| CI platform | GitHub Actions (`.github/workflows/ci.yml` — SQLite + PostgreSQL matrix) |
| Solo or team | Solo-git |
| Language + framework | Go 1.26 / ECC kernel + `modelcontextprotocol/go-sdk` |
| Greenfield or existing | Existing codebase (19+ components) |

---

## Plan Artifact Quality (e67)

| Artifact | Status | Detail |
|----------|--------|--------|
| `epic.yaml` | ✅ | 4 stories, BCPs, hard_gate, depends_on documented |
| Story specs | ✅ | 17–20 sections incl. Requirements delta tags (ADDED/MODIFIED on s03) |
| Task YAML | ✅ | All 29 tasks `status: failing`; risk + security fields on s03 |
| Verify commands | ✅ | Every task ends with runnable `go test` / `go build` |
| Security spec (s03) | ✅ | Cross-site 403, `bb_dep_` prefix order, `kernel.CtxSiteID` in scope.go |
| Test plan artifact | ⚠️ | No `specs/tech-architecture/e67-TEST_PLAN_LATEST.md` — heuristics used instead |
| plan-consistency-check | ❌ | `scripts/lib/plan-consistency-check.sh` not in repo — plan-work gate cannot run mechanically |

---

## Story Readiness

| Story | BCPs | Plan ready? | Blockers |
|-------|------|-------------|----------|
| e67s04 get_ci_template | 1 | ✅ | e57 declared dependency; no code deps in practice |
| e67s01 create_repo | 2 | ✅ | e57 declared dependency |
| e67s02 create_site | 2 | ✅ | e57 declared dependency; HTTP still requires `name` (MCP-only defaulting) |
| e67s03 provision_ci_credentials | 2 | ✅ | Hard gate — auth middleware + deploy handler changes |

Recommended build order (lowest risk first): **e67s04 → e67s01 → e67s02 → e67s03**

---

## Open Gaps (Blocking)

- [ ] **`specs/security/epics/e67/THREAT_MODEL.md` missing** — build-epic step 0 requires threat model before coding. e57/e51/e49/e48 have models; e67 does not.
- [ ] **e57 dependency unsatisfied** — all e67 stories declare `depends_on: [e57]`; `execution-status.yaml` shows e57 + all e57 stories `planned`. Building e67 now risks rework when project scoping lands (sites/auth/git tables, `kernel.ProjectIDFromContext`).

## Open Gaps (Non-Blocking)

- [ ] Add e67 to `specs/product/SCOPE_LATEST.yaml` under v3.0 release scope
- [ ] Create `specs/product/GLOSSARY_LATEST.yaml` or refresh from archive
- [ ] Add `scripts/lib/plan-consistency-check.sh` (plan-work HARD GATE from skill v1.18+)
- [ ] Record e57 waiver in `specs/state.yaml` `open_decisions` if proceeding in yolo mode without e57

---

## Yolo Mode Assessment

User intent: run `build-epic` s01→s04 without checkpoints.

| build-epic gate | Current state | Yolo impact |
|-----------------|---------------|-------------|
| Step 0 threat model | ❌ Missing | **Block** — run `security-review` first (~10 min) |
| Step 3 kickoff-branch | On `main` with dirty tree | **Block** — need feature branch before first commit |
| e57 depends_on | All planned | **Risk** — e67s03 touches auth/deploy/sites; may conflict with in-flight e57 branch |
| Plan artifacts | ✅ Complete | Safe to code once gates above cleared |

**Yolo-safe subset:** e67s04 only (embedded JSON, no component wiring, no e57 table changes). s01–s03 should wait for e57 or explicit waiver.

---

## Verdict

**NOT READY** — 2 blocking gaps before `build-epic`:

1. Create `specs/security/epics/e67/THREAT_MODEL.md` (run `security-review`)
2. Resolve e57 dependency — **either** complete e57 first **or** record explicit waiver + accept rework risk

After closing blockers → **READY** → `survey-context` → `kickoff-branch` → `develop-tdd` (e67s04 first).

→ verify: `test -f specs/PLAN-AUDIT_LATEST.md && grep -q 'Verdict' specs/PLAN-AUDIT_LATEST.md && echo OK || echo FAIL`
