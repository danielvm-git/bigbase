# Plan Audit — BigBase e67 (Re-Audit)
**Date:** 2026-07-05 · **Verdict:** READY — 0 blocking, 0 concerns

Re-audited after: all 7 gaps closed, ECC violation fixed, name ownership unified.

---

## Closed Gaps

| Gap | Severity | Fix |
|-----|----------|-----|
| GAP-1 (hard_gate) | ⚠️ | Added to epic.yaml |
| GAP-2 (ConstantTimeCompare) | ⚠️ | Clarified: SQLite UNIQUE index lookup — no timing channel |
| GAP-3 (org_id sentinel) | 🔴 | Standardized on `org_id = 0` (existing pattern in git.go:165) |
| GAP-4 (cross-site deploy) | 🔴 | `kernel.CtxSiteID` + deploy.HandleCreate 403 enforcement |
| GAP-5 (domain in create_site) | 🔴 | Domain removed; `site_id` only; domain is deploy property |
| GAP-6 (ConstantTimeCompare over-spec) | 🟡 | Removed; clarified SQLite does the comparison |
| GAP-7 (site existence) | 🟡 | `SELECT 1 FROM sites WHERE id = ?` before INSERT |
| ECC concern (ctxSiteID in auth) | 🔴 | Moved to `kernel/` — neutral, imported by both |
| Name ownership (two defaults) | 🟡 | Sites.CreateSite is single owner; MCP is passthrough |

---

## ECC Architecture Check

| Check | Status |
|-------|--------|
| `ctxSiteID` + `SiteIDFromContext()` location | ✅ `kernel/` — neutral, imported by both auth and deploy |
| `components/deploy` imports `components/auth` | ❌ No — and stays that way |
| `components/auth` imports `kernel` | ✅ Already true |
| `components/deploy` imports `kernel` | ✅ Already true |

`kernel.CtxSiteID` and `kernel.SiteIDFromContext(ctx)` follow the same pattern as existing kernel shared types (`kernel.DBer`, `kernel.Logger`, `kernel.Version`). No new packages, no component coupling.

---

## Name Defaulting Ownership

`Sites.CreateSite` is the **single owner** of name defaulting:
- When `name` is empty, queries `git_repos` for repo name → sets as `FullName`
- Returns `(id, name string, err error)` — including the resolved name
- MCP tool is a **passthrough** — validates inputs, calls interface, returns result
- Deterministic: same input always produces same name in response

---

## Lens 1 — Bigpowers Pre-flight

| Question | Answer | Status |
|----------|--------|--------|
| Test command | `go test ./components/mcp/ -run Test<Story> -v -count=1` (per-story) + `go test ./... -count=1` (full suite) | ✅ |
| Build command | `go build .` | ✅ |
| Lint | `golangci-lint run ./...` | ✅ |
| Typecheck | `go vet ./...` | ✅ |
| CI platform | GitHub Actions + semantic-release | ✅ |
| Solo or team | Solo | ✅ |
| Language + framework | Go 1.26.3 / ECC kernel + `modelcontextprotocol/go-sdk` v1.6.1 | ✅ |
| Greenfield or existing | Existing — 19 components | ✅ |

---

## Lens 2 — Conventions Completeness

| Check | Status | Note |
|-------|--------|------|
| `AGENTS.md` | ✅ | ctxo + orca-cli instructions |
| `CONVENTIONS.md` | ✅ | Conventional Commits + solo-git documented |
| `specs/` layout | ✅ | Full bigpowers capsule pattern |
| Conventional Commits | ✅ | `semantic-release` in CI |
| Git workflow mode | ✅ | Solo-git documented |
| Tech-stack doc | ✅ | `specs/tech-architecture/tech-stack.md` |
| ADR index | ✅ | 5 ADRs; e67 introduces no new architectural decisions |
| release-plan.yaml | ✅ | e67 registered with depends_on: [e57], WSJF 8.0 |

---

## Lens 3 — Principles Alignment (Re-Validated)

### Vertical Slices

| Story | BCPs | Independent? | Shippable alone? |
|-------|------|-------------|-----------------|
| e67s01 create_repo | 2 | ✅ | Yes |
| e67s02 create_site | 2 | ✅ | Yes — returns site_id only; domain comes from deploy |
| e67s03 provision_ci_credentials | 2 | ✅ | Yes — extends org_api_keys, reuses existing hash infrastructure |
| e67s04 get_ci_template | 1 | ✅ | Yes |

### Scope Boundaries

All four have explicit §16 Out of Scope. e67s02 no longer claims to return a domain — it correctly returns only `site_id`, delegating domain computation to the deploy component (where `deploymentURL()` lives).

### Success Criteria

Gherkin AC with happy path + error handling + edge cases in all four stories. e67s03 now includes cross-site authorization rejection (403) and auth context enforcement.

### HARD GATE

`hard_gate: e67s03` present in `epic.yaml`. Security-sensitive: introduces site-scoped deployment keys with middleware authorization enforcement.

### Prior Gap Cross-Reference

e67 clean against all prior e56–e65 gaps: no IDOR routes, no env injection, no route conflicts, no slug validation gaps.

---

## Lens 4 — Security Review (Re-Validated)

### GAP-4 — Deploy cross-site enforcement ✅ FIXED

`e67s03-spec.md` §5 now defines `kernel.CtxSiteID` context key + deploy.HandleCreate enforcement without direct auth→deploy imports. Story tasks include task 6 (deploy handler check) and task 11 (cross-site integration test). Acceptance criteria includes "Scenario: cross-site deployment is rejected → 403."

### GAP-5 — Domain in create_site ✅ FIXED

`e67s02-spec.md` now clearly states domain is a deployment property, not a site property. MCP tool returns `site_id` + `name` only. Domain comes from `deploy_site`/`get_deploy_status`. Interface signature: `CreateSite(ctx, gitRepoID, name, branch) (id string, error)`.

### GAP-3 — org_id sentinel ✅ FIXED

`e67s03-spec.md` §4/§6 standardized on `org_id = 0` as sentinel for site-scoped keys. References existing pattern in `git.go:165`. Tasks reflect: `INSERT ... org_id=0`.

### GAP-6 — ConstantTimeCompare ✅ CLARIFIED

`e67s03-spec.md` §6 clarifies hash comparison is performed by SQLite `WHERE key_hash = ?` on `UNIQUE` column — single-row index lookup, no timing channel. No Go-level constant-time comparison needed.

### GAP-7 — Site existence validation ✅ FIXED

`e67s03-spec.md` §6 step 4 now includes `SELECT 1 FROM sites WHERE id = ?` before INSERT. Task 2 updated to "validate site exists." Testing includes site existence scenario.

### GAP-8 — Handler extraction surface ✅ ACCEPTED

Both e67s01 and e67s02 extract existing HTTP handler logic into exported methods that the MCP interfaces call. This refactoring surface is expected and explicitly noted in tasks.

### All Other Security Dimensions

| Check | Status |
|-------|--------|
| Secrets in code | ✅ No secrets; tokens generated via `crypto/rand` |
| Parameterized queries | ✅ `WHERE key_hash = ?`, `WHERE id = ?` |
| Input validation | ✅ All tool args validated |
| Generic error messages | ✅ "invalid site key" not "hash mismatch" |
| Token storage | ✅ SHA-256 hash stored, raw token returned once |
| Token prefix for fast-path | ✅ `bb_dep_` before `bb_` check |
| Token revocation | ✅ `revoked` column, `WHERE revoked = 0` |
| Authorization enforcement | ✅ `ctxSiteID` → deploy handler 403 check |
| Log redaction | ✅ Logs `key_id` and `site_id`, never raw token |

---

## Open Gaps

None — all gaps from re-audit closed.

---

## Minor Notes (Non-Blocking)

| # | Note |
|---|------|
| 9 | release-plan.yaml note describes sequential dream workflow; epic.yaml says any order. Both correct — stories are independent but the workflow narrative is linear. |
| 10 | e57 epic.yaml does not list e67 in `blocks` — cosmetic. e67 depends_on [e57] already enforces ordering. |
| 11 | Prior audit said e67 "does not touch deploy.go" — now inaccurate. e67s03 adds deploy.HandleCreate cross-site check. |

---

## Story Readiness

| Story | Ready? | Verdict |
|-------|--------|---------|
| e67s04 get_ci_template | ✅ | Start after e57. Lowest risk — embedded JSON knowledge file. |
| e67s01 create_repo | ✅ | Start after e57. Follows proven MCP interface-wiring pattern. |
| e67s02 create_site | ✅ | Start after e57. Domain correctly delegated to deploy. Name defaults from repo. |
| e67s03 provision_ci_credentials | ✅ | Start after e57. Hard gate cleared — org_id=0, cross-site enforcement, ctxSiteID all spec'd. |

---

## Verdict

**READY** — proceed with `survey-context` when e57 hard gate (e57s04) lands, then `develop-tdd` in order: e67s04 → e67s01 → e67s02 → e67s03.

→ verify: `test -f specs/PLAN-AUDIT_LATEST.md && grep -q 'Verdict' specs/PLAN-AUDIT_LATEST.md && echo OK || echo FAIL`
