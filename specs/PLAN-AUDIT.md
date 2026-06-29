# Plan Audit — BigBase e56–e65
**Date:** 2026-06-29 · **Verdict:** NOT READY — 13 gaps, 3 blocking

Audited against: `/security-review` lens, `/deepen-architecture` open issues (#41–#45 via `gh`), bigpowers principles, and project conventions.

---

## Lens 1 — Bigpowers Pre-flight

| Question | Answer | Status |
|----------|--------|--------|
| Test command | `go test ./...` | ✅ |
| Build command | `go build -o bigbase .` | ✅ |
| Lint | `golangci-lint run ./...` | ✅ |
| Typecheck | `go vet ./...` | ✅ |
| CI platform | GitHub Actions + semantic-release | ✅ |
| Solo or team | Solo (worktree branching, direct-to-main) | ✅ |
| Language + framework | Go 1.26.3 / ECC kernel + SQLite/PostgreSQL + React 19 | ✅ |
| Greenfield or existing | Existing — 19 components, 51/66 epics complete | ✅ |

---

## Lens 2 — Conventions Completeness

| Check | Status | Note |
|-------|--------|------|
| `CLAUDE.md` | ✅ | Present and current |
| `CONVENTIONS.md` | ✅ | Present |
| `specs/` layout | ✅ | Full bigpowers layout |
| Conventional Commits | ❌ | Used in practice; missing from CONVENTIONS.md |
| Git workflow mode | ❌ | Solo-git pattern evident but not documented |
| Tech-stack doc | ✅ | `specs/tech-architecture/tech-stack.md` v2.62.0 |
| ADR index | ✅ | 5 ADRs; 0005 (deploy decomposition) accepted |
| release-plan.yaml | ⚠️ | e50 + e51 show `status: planned`; both are done |

---

## Lens 3 — Principles Alignment

| Check | Status | Note |
|-------|--------|------|
| Vertical slices (e56–e61) | ✅ | Each story independently shippable |
| Scope bounded | ✅ | All specced stories have explicit non-goals |
| Success criteria | ✅ | Gherkin or numbered acceptance in specced stories |
| HARD GATE candidates | ⚠️ | e57s04 (JWT + backfill) is high-risk; not formally marked |
| Domain language | ✅ | Vocabulary consistent with tech-stack.md |
| Wave-2 specs (e62–e65) | ❌ | All placeholder stubs — no stories written |

---

## Lens 4 — Security Review (`/security-review`)

### GAP-SEC-1 — IDOR contract missing from e57s02 🔴 **BLOCKING**

`e57s02-spec.md` §7 Data Flow says `"Claims.OrgID must match URL orgID (owner check)"` but this is buried in prose, not in §5 Contract or §14 Security. The security contract conflates *"user must own the org"* with *"orgID in URL must equal claims.OrgID"* — these are different checks. A user who owns org 1 can still request `GET /api/orgs/2/projects` without an explicit cross-org rejection.

**Fix:** Add to e57s02 §5 (Contract): `"Handler asserts orgIDFromURL == claims.OrgID; returns 403 on mismatch before any ownership check."` Add acceptance scenario:
```gherkin
Scenario: Cross-org access rejected (IDOR)
  Given user A owns org 1 and is authenticated
  When user A calls GET /api/orgs/2/projects
  Then response is 403 Forbidden
```

---

### GAP-SEC-2 — Route conflict between e57s02 and e58s01 🔴 **BLOCKING**

`e57s02` registers project CRUD under `/api/orgs/{orgID}/projects`.
`e58s01-spec.md` §4 defines a separate Go file `components/auth/projects_api.go` with routes at `/api/auth/projects` and `/api/auth/projects/{id}`.

These are **two different URL patterns for the same resource** in the same component. The spec note says *"If e57s02 is done, this endpoint already exists"* but does not reconcile the URL mismatch. Either:
- e58s01 should call the existing e57s02 endpoints (no new routes), or
- e57s02's routes should be moved to `/api/auth/projects` for consistency with the rest of the auth API surface.

**Fix:** Decide canonical URL pattern before writing any handler code. Recommend: keep e57s02's `org-scoped` path (`/api/orgs/{orgID}/projects`) and remove the duplicate handler definition from e58s01.

---

### GAP-SEC-3 — Rate limiting absent from new CRUD endpoints 🟠

`e57s02` adds 5 routes; `e58s01` adds 7 routes under auth; `e61s01` adds 4 routes — total 16 new endpoints across 3 consecutive epics with zero mention of rate limiting in any spec. `CONVENTIONS.md` Defensive Code: *"Rate limit — all API endpoints."*

**Fix:** Each spec must state which rate-limiter the routes use (the existing auth rate-limiter middleware, or explicit deferral with a task).

---

### GAP-SEC-4 — e61s01 secret injection plan directly violates Issue #41 🔴 **BLOCKING**

This is the most structurally important security gap. `e61s01-spec.md` §4 Implementation:
> *"Modify site build and start flows to fetch project secrets, decrypt them, and inject them alongside site env vars."*

This is exactly what **Issue #41** (EnvResolver seam, Priority 2) was filed to prevent:
> *"Without a single resolver owning precedence + redaction, that logic gets duplicated across the build path and the start path — exactly where a redaction bug leaks a secret into a build log."*

`e61s01` adds a third merge/injection point without a central redaction module. §9 Risks acknowledges *"Secret leakage in logs"* as a Medium/High risk with mitigation *"Ensure values are never written to server or build logs"* — but this is a manual discipline, not a structural guarantee. The spec defines merge precedence (`site env vars override project secrets`) in isolation from the existing `env.go` build-env logic, guaranteeing drift.

**Fix:** e61 must be gated behind an `EnvResolver` deepen story. The resolver owns: layering (platform defaults → project secrets → site env vars), conflict precedence (site wins), and a `RedactedView()` for log consumption. e61s01 and e65s01 both plug into this one seam. **This is a depends_on change, not just a spec note.**

---

### GAP-SEC-5 — e56s01 OTP: timing-safe comparison not specified 🟡

`e56s01` stores `code_hash TEXT` and defines `Verify(ctx, email, codeHash string)`. The spec does not specify that the DB comparison must use `subtle.ConstantTimeCompare` (or `hmac.Equal`). A plain string equality `storedHash == incomingHash` over fixed-length hex hashes has minimal timing risk but is not cryptographically correct.

**Fix:** Add to e56s01 §4 (or tests): *"The `dbOTPStore.Verify` implementation uses `subtle.ConstantTimeCompare` to compare hashes, not string equality."*

---

### GAP-SEC-6 — e56s01 stale dependency reference 🟡

`e56s01-spec.md` §7 Dependencies references `e52 (Project Scoping — kernel/scope.go)`. e52 was renumbered to e57 in commit `68a832b1`.

**Fix:** Update reference to `e57 (Project Scoping)`.

---

### GAP-SEC-7 — e57s02 slug validation incomplete 🟡

§14 Security says *"alphanumeric + hyphens only, 1-64 characters"* — validation criteria exist but:
- No regex pattern to enforce in code/tests
- `"default"` not reserved — e57s04 auto-creates a project with slug `"default"` at org creation; a user-created project with the same slug in the same org will hit a `UNIQUE(org_id, slug)` conflict with a cryptic DB error

**Fix:** Add to e57s02 §5: `slug must match ^[a-z0-9][a-z0-9-]{0,63}$`. Reserve `"default"` at the handler level with a 422 response.

---

### GAP-SEC-8 — e57s04 backfill idempotency undocumented 🟡

The §10 Migration SQL is idempotent by design (`INSERT WHERE id NOT IN ...`). But §19 Risks only addresses "large production DB failure" — it doesn't state *"this backfill is safe to re-run after a crash."* A developer reading only §19 would not know the restart behavior.

**Fix:** Add to §19: *"`BackfillSitesToProjects` is idempotent: the INSERT uses `WHERE id NOT IN (...)` and is a no-op on re-run. Add test: run backfill twice, assert site counts identical."*

---

## Lens 5 — Deepen-Architecture (`/deepen-architecture`)

Open issues via `gh issue list --state open`, mapped against e56–e65:

| Issue | Title | Priority | Impacted Epics | Verdict |
|-------|-------|----------|---------------|---------|
| #41 | Env/Secrets resolution seam (env.go) | 2 — imminent | e61 🔴, e65 🔴 | **DIRECTLY VIOLATED** by e61s01 — see GAP-SEC-4 above |
| #42 | Sites→Deploy callback → EventBus | 3 — moderate | e65 | 🟡 e65 is a stub — must address before e65 spec is written |
| #43 | Policy Gate for route access control | 4 — distant | e57, e58, e61 | 🟠 3 epics add 16 routes without it — migration surface growing |
| #44 | Auth Verifier seam | 5 — someday | e57s04 | ✅ Correctly deferred — but e57s04 must note the debt |
| #45 | ConfigSchema() activation | 6 — low | e61 | 🟡 e61 adds `--env-encryption-key` flag without ConfigSchema |

---

### GAP-ARCH-1 — Issue #41 is a build blocker for e61, not just a note 🔴 **BLOCKING**

The issue cites `e47/e61` (Secrets) and `e51/e65` (Preview Environments) as its forcing functions. e61s01 proves the forcing function is now active — its implementation plan will duplicate merge logic. **Issue #41 must be resolved before e61 starts, not added to a note.**

**Fix:** Add to `e61/epic.yaml`:
```yaml
depends_on: [e51, e57, deepen-env-resolver]
```
And create a deepen story `e61-pre: EnvResolver` between e57 and e61 covering `components/deploy/env.go` (build path) + a `RuntimeResolver` (start path). Both e61 and e65 consume this seam.

---

### GAP-ARCH-2 — Issue #43 (Policy Gate) migration surface growing 🟠

The issue says "do this before e66." e57s02 (5 routes), e58s01 (7 routes), and e61s01 (4 routes) add 16 org-scoped routes with manual `OrgIDFromContext()` checks in three consecutive epics before e66. Each is one more route to migrate when the Policy Gate is implemented.

No blocking fix required now, but each story should acknowledge the debt: add to §19 Risks of e57s02, e58s01, and e61s01: *"Routes use manual org-scoping; will be migrated to the Policy Gate (issue #43) before e66."*

---

### GAP-ARCH-3 — Issue #45 (ConfigSchema) compounding with e61 🟡

Issue #45 notes that each deploy epic adds flags without activating ConfigSchema. e61 introduces `--env-encryption-key` / `BIGBASE_ENV_ENCRYPTION_KEY` — a new stringly-typed flag with silent zero-value on misconfiguration. The issue's own priority is 6 (low), so no blocking action, but the compounding pattern makes it worse.

**Observation:** e61 should at minimum validate the key at startup and fail fast with a human-readable error — even without a full ConfigSchema implementation.

---

### GAP-ARCH-4 — e57s04 HARD GATE not marked; auth.go debt invisible ⚠️

`e57s04` is the highest-risk story (JWT schema change + 5 callers + backfill migration). Not formally marked as a HARD GATE in `epic.yaml`. Additionally, e57s04 adds ~120 lines to `auth.go` but §19 Risks has no note acknowledging this increases the depth debt tracked in issue #44.

**Fix (epic.yaml):** Add `hard_gate: e57s04`.
**Fix (e57s04 §19):** Add: *"This story adds ~120 lines to auth.go (project_members, 5 token-issuing paths, middleware). Accepted debt per issue #44 — no forcing function. Reverts cleanly if a Verifier interface is extracted later."*

---

### GAP-ARCH-5 — ADR-0005 quick wins orphaned ⚠️

State.yaml context notes: *"Phase 1 (Logger/DBer hoisting) is 15-minute mechanical cleanup."* These quick wins are not in any task list, not a separate story, and not referenced in e57s01's tasks. They exist only as prose.

**Fix:** Add as task 0 to `e57s01-tasks.yaml` or create a pre-e57s01 task: `"Hoist Logger/DBer interfaces per ADR-0005 Phase 1 quick wins."` This unblocks the Engine extraction path.

---

## Per-Epic Summary

| Epic | Title | Principles | Security | Architecture | Verdict |
|------|-------|-----------|---------|------------|---------|
| e56 | OTP Persistence & Session Audit | ✅ | ⚠️ timing+stale ref | ✅ clean DI | ⚠️ 2 small fixes |
| e57 | Project Scoping Backend | ⚠️ no HARD GATE | 🔴 SEC-1,3 open; SEC-7,8 partial | 🟠 no #41 fwd; no #44 note | ❌ BLOCKING |
| e58 | Project Scoping UI | ✅ | 🔴 route conflict w/ e57 | 🟠 #43 debt note missing | ❌ BLOCKING |
| e59 | Native Feature Port | ✅ epic well-structured | ✅ e59s01 has security rules | ✅ e59s02 correctly gated | ✅ READY |
| e60 | Route Rename | ✅ | ✅ low risk, 301 shim | ✅ | ✅ READY |
| e61 | Secrets Management | ✅ epic.yaml complete | 🔴 violates #41 | 🔴 must depend on EnvResolver | ❌ BLOCKING |
| e62 | CSP Headers | ❌ stub | ❌ stub | ❌ stub | ❌ NOT SPECCED |
| e63 | Usage Dashboard | ❌ stub | ❌ stub | ❌ stub | ❌ NOT SPECCED |
| e64 | Schema Designer | ❌ stub | ❌ stub | ❌ stub | ❌ NOT SPECCED |
| e65 | Preview Environments | ❌ stub | ❌ stub | ❌ must address #42 | ❌ NOT SPECCED |

---

## Open Gaps (Ordered by Priority)

| # | Gap | Sev | Epic | Fix |
|---|-----|-----|------|-----|
| SEC-4 / ARCH-1 | e61s01 build+runtime injection violates Issue #41 (EnvResolver) | 🔴 | e61 | Gate e61 behind deepen story; add `depends_on: deepen-env-resolver` |
| SEC-1 | e57s02 IDOR contract not in §5 or §14 | 🔴 | e57s02 | Add explicit authorization assertion + IDOR acceptance scenario |
| SEC-2 | e58s01 route conflict with e57s02 (`/api/auth/projects` vs `/api/orgs/{id}/projects`) | 🔴 | e58s01 | Pick canonical URL; remove duplicate handler definition |
| ARCH-1 | e57/epic.yaml missing `open_decisions` for Issue #41 as e61 blocker | 🟠 | epic.yaml | Add open_decision entry |
| ARCH-2 | Issue #43 debt not noted in e57s02, e58s01, e61s01 | 🟠 | all 3 | Add §19 Risk note to each |
| SEC-3 | Rate limiting not mentioned across 16 new endpoints (e57s02, e58s01, e61s01) | 🟠 | 3 epics | State rate-limiter reference or explicit deferral in each spec |
| ARCH-4 | e57s04: no HARD GATE in epic.yaml; auth.go debt not in §19 | ⚠️ | e57 epic + s04 | Add `hard_gate: e57s04`; add §19 note |
| ARCH-5 | ADR-0005 quick wins orphaned in state.yaml prose | ⚠️ | e57s01 | Add as task to e57s01-tasks.yaml |
| SEC-5 | e56s01 `Verify()` doesn't specify `subtle.ConstantTimeCompare` | 🟡 | e56s01 | Add to spec §4 implementation note |
| SEC-6 | e56s01 stale reference to e52 (renamed to e57) | 🟡 | e56s01 | Update §7 Dependencies |
| SEC-7 | e57s02 slug validation: no regex, "default" not reserved | 🟡 | e57s02 | Add regex to §5; reserve "default" |
| SEC-8 | e57s04 backfill idempotency not documented | 🟡 | e57s04 | Add to §19 |
| CONV-1 | Conventional Commits not in CONVENTIONS.md | ⚠️ | global | Append Git section |
| CONV-2 | Solo-git workflow mode not documented | ⚠️ | global | Append to Git section |
| CONV-3 | release-plan.yaml: e50 + e51 still `status: planned` | ⚠️ | global | Update both to `done` |
| EPIC-stubs | e62–e65 are placeholder stubs (no story specs) | ❌ | e62-e65 | Run `plan-work` for each |

---

## Verdict

**NOT READY** — 3 blocking gaps (SEC-4/ARCH-1, SEC-1, SEC-2) must close before any `develop-tdd` runs.

### Structural misalignment worth naming directly

The plan has a **vision-execution gap** on secrets. The project vision (ECC, single responsibility, "components communicate via events not imports") demands centralized redaction. The current e61s01 implementation plan contradicts this by threading injection through two separate code paths. This isn't a minor spec tweak — it's a design decision that will create the secret-leakage bug the project is trying to prevent. Issue #41 must become a first-class story, not a note.

### Recommended next steps

1. **Immediately (unblock e57):** Fix SEC-1 (e57s02 §5), SEC-7 (slug), ARCH-4 (epic.yaml HARD GATE + §19 notes), SEC-8 (backfill idempotency), ARCH-5 (ADR-0005 task)
2. **Before e61 kicks off:** Create deepen story `EnvResolver` — `components/deploy/env.go` → layering + redaction + runtime path. e61 and e65 both depend on it.
3. **Resolve SEC-2:** Pick canonical URL pattern for projects resource before e58 starts
4. **Parallel with e57 build:** Run `plan-work` on e62, e63, e64; address Issue #42 in e65 spec before writing it
5. **Conventions:** Add Git section to CONVENTIONS.md; fix release-plan.yaml stale statuses

→ verify: `test -f specs/PLAN-AUDIT.md && grep -q 'Verdict' specs/PLAN-AUDIT.md && echo OK || echo FAIL`
