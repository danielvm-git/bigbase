# Impact Assessment — Close All Open GitHub Issues (Bands 1+2)

Generated: 2026-07-10
Scope: 10 open issues against v3.0.0 planning

## Summary

| Issue | Epic/Area | Risk | Action |
|-------|-----------|------|--------|
| #60 | e65 WSJF rescore | Low | YAML edit only |
| #42 | Dead onGitHubPush event | Low | 2-line removal + ADR fix |
| #62 → e73 | Python (FastAPI) Runtime | Medium | 4 stories, 8 BCP, all in components/deploy/ |
| #58 → e70 | Site Deploy Manifest | Medium (blocked by e57) | Plan-only, 4 stories, 7 BCP |
| #41 → e61 | Secrets Management | Medium (blocked by e57) | Deferred — EnvResolver gap note |
| #43 → e66 | Multi-User Platform | Medium (blocked by e57) | Deferred — PolicyGate prerequisite |
| #44 | Auth Verifier seam | Low | Close — no forcing function |
| #45 | ConfigSchema activation | Low | Leave open — someday-marker |

## Detailed Analysis

### Issue #60 — WSJF Rescore for e65 (Preview Environments)

- **Target**: `specs/release-plan.yaml` — `e65.wsjf` field
- **Dependents**: None. Pure YAML metadata. Zero code impact.
- **Affected Stories**: None directly. Moves e65 from WSJF 2.5 → 10.5, prioritizing preview environments for big-library content review pipeline.
- **Test Coverage**: N/A — no code change.
- **Risk**: Low
- **Recommended Action**: Single-line YAML edit. Verify with grep.

### Issue #42 — Dead onGitHubPush Event

- **Target**: `components/github/github.go` (line ~448-457), `specs/adr/0003-github-app-sites.md` (line 15)
- **Dependents**: Zero subscribers. Verified: `rg "onGitHubPush"` across codebase shows only the emitter and the ADR reference.
- **Affected Stories**: None. Removing dead code. ADR already documents the correct delegate-based architecture (line 16: "Sites component delegates deploy via TriggerDeploy callback injected from main.go").
- **Test Coverage**: `go test ./components/github/...` — existing tests cover webhook handling but not the specific event emission path.
- **Risk**: Low
- **Recommended Action**: Remove 10-line Emit block. Fix ADR line 15. Verify no regressions.

### Issue #62 → e73 Python (FastAPI) Runtime

- **Target**: `components/deploy/deploy.go`, `deploy_runner.go`, `supervisor.go`, `manifest.go`
- **Dependents**: All deploy callers (sites, API, admin UI). Changes are additive — new detection paths, new build steps. Existing Node.js/Go/static site flows unchanged.
- **Affected Stories**: 4 new stories (e73s01-s04). Reference app: Grimoire (FastAPI 3.13, uv, SQLite).
- **Test Coverage**: ~12 test references in `components/deploy/`. New tests needed for `DetectAppType`, `getStartCommand`, health polling, system deps parsing.
- **Risk**: Medium. Four stories all touching the deploy subsystem. Key risk areas:
  - `DetectAppType()` — must not regress existing app detection (Node, Go, static)
  - `getStartCommand()` — uvicorn path must not override existing python3 fallback
  - Supervisor health polling — must not interfere with crash-loop restart for non-Python apps
  - Disk allocation — must isolate per-deployment directories
- **Recommended Action**: Full build-epic 9-step cycle per story. Security-review (Step 0) on uv/uvicorn process execution. Enforce-first on new tests.

### Issue #58 → e70 Site Deploy Manifest (Plan Only)

- **Target**: `components/sites/sites.go`, `components/deploy/manifest.go`, `go.mod`
- **Dependents**: Blocked by e57 (Project Scoping Backend) — needs site schema with deploy_defaults field. e70s01 cannot build until e57 ships.
- **Affected Stories**: 4 planned: e70s01 (persist defaults), e70s02 (bigbase.toml parser), e70s03 (CI templates), e70s04 (static-sidecar).
- **Test Coverage**: Sites component has 1003 lines. Manifest parser is YAML-only (262 lines), no TOML support. New dependency: `BurntSushi/toml` or `pelletier/go-toml/v2`.
- **Risk**: Medium (blocked). Plan-only in this execution. Hard gate: e57 must ship before e70s01.
- **Recommended Action**: Planning spine only. Write complete capsule with epic.yaml + tasks.yaml. Register in release-plan.yaml. Do not build.

### Issue #41 → e61 Secrets Management (Deferred)

- **Target**: `components/deploy/env.go`, `specs/epics/e61-secrets/`
- **Dependents**: Blocked by e57 (project-scoped secrets need project schema).
- **Affected Stories**: 2 existing (e61s01-s02, 3 BCP). Missing: EnvResolver seam.
- **Risk**: Medium (deferred). The EnvResolver interface (layering: platform env → user env → secrets, precedence rules, redaction) must be specified in e61s01.
- **Recommended Action**: Add EnvResolver gap note to e61 epic.yaml. Defer build to Band 3.

### Issue #43 → e66 Multi-User Platform (Deferred)

- **Target**: `specs/epics/e66-multi-user-platform/`
- **Dependents**: Blocked by e57 (project-scoped roles need project schema).
- **Affected Stories**: 3 planned (e66s01-s03, 8 BCP). Missing: PolicyGate prerequisite.
- **Risk**: Medium (deferred). Role/invite stories need a PolicyEnforcer interface in auth/proxy layer.
- **Recommended Action**: Add PolicyGate prerequisite story note to e66 epic.yaml. Defer build to Band 3.

### Issue #44 — Auth Verifier Seam

- **Target**: None
- **Dependents**: None. Issue's own recommendation: "do not schedule."
- **Risk**: Low
- **Recommended Action**: Close via `gh issue close 44` with note "Not planned — no forcing function."

### Issue #45 — ConfigSchema Activation

- **Target**: None
- **Dependents**: None. Priority 6. No user-facing benefit.
- **Risk**: Low
- **Recommended Action**: Leave open as someday-marker. Activate only if e70 TOML parsing creates concrete need for schema validation.

## BCP Totals

### This Execution (Bands 1+2)

| Epic | Stories | BCP | Status |
|------|---------|-----|--------|
| — | Issue #42 | 0 | Build (Band 1) |
| e73 | 4 | 8 | Build (Band 1) |
| **Total** | **4 stories** | **8 BCP** | |

### Future Band 3

| Epic | Stories | BCP | Blocks |
|------|---------|-----|--------|
| e57 | 5 | 17 | e70, e61, e66 |
| e61 | 2 | 3 | — |
| e66 | 3 | 8 | — |
| e70 | 4 | 7 | — |
| **Total** | **14 stories** | **35 BCP** | |

## Files Affected (Bands 1+2)

| File | Band | Change |
|------|------|--------|
| `specs/release-plan.yaml` | 1 | #60 WSJF + e73 registration |
| `components/github/github.go` | 1 | #42 dead event removal |
| `specs/adr/0003-github-app-sites.md` | 1 | #42 stale claim fix |
| `components/deploy/deploy.go` | 1 | e73s01 + e73s02 |
| `components/deploy/deploy_runner.go` | 1 | e73s03 |
| `components/deploy/supervisor.go` | 1 | e73s03 + e73s04 |
| `components/deploy/manifest.go` | 1 | e73s04 |
| `specs/state.yaml` | 2 | Handoff context |
| `specs/product/SCOPE-e70.yaml` | 2 | #58 scope |
| `specs/epics/e70-site-deploy-manifest/` | 2 | #58 capsule |
| `specs/epics/e61-secrets/epic.yaml` | 2 | #41 EnvResolver note |
| `specs/epics/e66-multi-user-platform/epic.yaml` | 2 | #43 PolicyGate note |
