# Plan Audit — Epic e72: Monitoring Enhancements

**Date:** 2026-07-08 · **Verdict:** READY

**Subject:** `specs/epics/e72-monitoring-enhancements/` + ADR 0007  
**Operator constraint:** DeepSeek API as default LLM provider ([api-docs.deepseek.com](https://api-docs.deepseek.com))

---

## Principles Alignment

| Check | Status | Note |
|-------|--------|------|
| Vertical slices | ✅ | 4 stories (e72s01–s04), each shippable with own acceptance criteria and verify commands |
| Scope bounded | ⚠️ | Per-story `Non-goals` / `Out of scope` present; no epic-level `in_scope`/`out_of_scope` block in `epic.yaml`. Waived — indexed in `release-plan.yaml` with WSJF + depends_on |
| Success criteria | ✅ | Gherkin AC in each story `.md`; per-task `verify:` in all four `*-tasks.yaml` |
| HARD GATE candidates | ✅ | ADR 0007 accepted — bus contracts, AlertIncident, composition-root seams resolved before build |
| Domain language | ✅ | Observability vocabulary in `tech-stack.md`; ADR 0007 canonical terms |

---

## Conventions Completeness

| Check | Status | Note |
|-------|--------|------|
| `CLAUDE.md` / `AGENTS.md` | ✅ | Present; ECC, TDD, specs-first documented |
| `CONVENTIONS.md` | ✅ | Go conventions, solo-git, Conventional Commits |
| `specs/` layout | ✅ | Epic capsule, story specs, task YAMLs, ADR 0007 |
| Commit conventions | ✅ | Conventional Commits + semantic-release in CONVENTIONS |
| Git workflow mode | ✅ | **solo-git** (CONVENTIONS § Git & Workflow) |

---

## Pre-flight Answers

| Question | Value |
|----------|-------|
| **test** | `go test ./...` (per-task: `go test -run 'Test…' ./components/... -count=1`) |
| **build** | `go build -o bigbase .` |
| **lint** | `golangci-lint run ./...` |
| **typecheck** | `go vet ./...` (Go — no separate tsc) |
| **CI platform** | GitHub Actions (`.github/workflows/ci.yml` — SQLite + Postgres matrix) |
| **Solo or team** | solo-git |
| **Language + framework** | Go 1.26 · ECC kernel · SQLite/PostgreSQL |
| **Greenfield or existing** | Existing codebase — monitoring + deploy components |

---

## Epic Summary

| Story | BCPs | Build order | Key deliverable |
|-------|------|-------------|-----------------|
| e72s02 | 2 | 1st | `pipeline_timeline` on deployments |
| e72s03 | 2 | 2nd | `eventrecorder` + related-events snapshot |
| e72s01 | 3 | 3rd | `deploy.failed` + DeepSeek diagnosis |
| e72s04 | 3 | 4th | AlertIncident + investigation |

**Prerequisites (cross-cutting):** deploy emits `deploy.failed`; fix dead `"deploy"` bus hook; proxy/api emit enriched events with `site_id`.

**Architecture anchor:** `specs/adr/0007-e72-observability-seams.md`

---

## LLM Provider — DeepSeek (operator decision, spec-updated)

| Setting | Value | Source |
|---------|-------|--------|
| API style | OpenAI-compatible `POST /chat/completions` | [DeepSeek API docs](https://api-docs.deepseek.com) |
| Default base URL | `https://api.deepseek.com` | ADR 0007 §7, e72s01 spec |
| Default model | `deepseek-chat` | Cost-effective one-shot diagnosis |
| API key | `BIGBASE_LLM_API_KEY` (primary) or `DEEPSEEK_API_KEY` (fallback) | e72s01-tasks.yaml task 1 |
| Override model | `BIGBASE_LLM_MODEL=deepseek-v4-pro` | Harder investigations (optional) |
| Disable AI | Unset API key — graceful 404 on diagnosis/investigation summary | e72s01 AC |

**Implementation note:** `internal/llm` must **not** append `/v1` (DeepSeek base is `https://api.deepseek.com`, not OpenAI's `/v1` path). Tests use `httptest` mock.

---

## Risk Register (audit findings)

| Risk | Severity | Mitigation in plan |
|------|----------|-------------------|
| Bus hook mismatch (`"deploy"` vs `deploy.state_changed`) | P1 | ADR 0007 + e72s01 task 4 |
| Alert re-fire every 30s | P1 | AlertIncident dedup in e72s04 task 1 |
| `request`/`mutation` events lack `site_id` | P1 | e72s03 task 0 prerequisites |
| LLM secrets in build logs | P1 | Strip before `Complete()` — e72s01 task 1 |
| Cross-component import violation | P2 | Composition-root reader interfaces — ADR 0007 §4 |
| e72s04 before e72s03 | P2 | `implementation_order` in epic.yaml |

---

## Open Gaps

- [x] Architecture decisions unresolved → **closed** via ADR 0007 (prior session)
- [x] LLM provider unspecified → **closed** — DeepSeek defaults applied to ADR 0007, e72s01, tech-stack
- [ ] Epic-level `out_of_scope` block in `epic.yaml` — **waived** (per-story non-goals sufficient for 10 BCP epic)
- [ ] `SCOPE_LATEST.yaml` does not list e72 — **waived** (e72 indexed in `release-plan.yaml` tier 2)

---

## Verdict

**READY** — proceed with `kickoff-branch` → `build-epic` (story order: e72s02 → e72s03 → e72s01 → e72s04).

Do **not** skip e72s03 before e72s04. Wire DeepSeek at implementation time:

```bash
export BIGBASE_LLM_API_KEY="<your-deepseek-key>"
# optional overrides:
# export BIGBASE_LLM_MODEL=deepseek-v4-pro
# export BIGBASE_LLM_BASE_URL=https://api.deepseek.com
```

---

## Recommended Next Skill

`kickoff-branch` (feature branch + clean test baseline) → `build-epic` starting **e72s02**.
