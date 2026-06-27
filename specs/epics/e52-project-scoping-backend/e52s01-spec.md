# e52s01: Kernel Interface Hardening

**Story ID:** e52s01 | **Epic:** e52 — Project Scoping Backend | **BCPs:** 4 | **Status:** planned

## 1. Type & Context

**type:** feat
**context:** infra
**maturity:** 3 — Countable

## 2. Story Statement

**As a** component developer,
**I want** standardized project-scoping context keys and helpers in the kernel,
**so that** every component can reliably inject and extract `project_id` from request context without ad-hoc duplication.

## 3. Context

This story **hardens the kernel** to support project-scoped multi-tenancy. The existing codebase already uses `context.Context` with typed keys for `org_id` (`ctxOrgID`), `user_id` (`ctxUserID`), `user_email`, and `user_role` — but these keys are duplicated across the `api` and `auth` packages. Adding `project_id` scoping requires the same pattern but standardized at the kernel level so every component imports from one canonical source.

The kernel currently provides:
- `kernel.Context` — carries `Kernel`, `Logger`, `Components`, and `Config`
- `kernel.DBer` — shared database abstraction
- `kernel.EventBus` — cross-component communication

This story adds project-scoping context helpers without changing any existing interface contracts.

### Zoom-Out Summary
- **Kernel's purpose:** Component lifecycle, event bus, config merge, and now: standardized scoping context.
- **Callers:** Every component using DBer — `api`, `auth`, `sites`, `deploy`, `storage`, `functions`, `forge`, `cici`, `git`, `messaging`, `monitoring`, `webhooks`, `backup`, `mcp`.
- **Contracts preserved:** `DBer` interface unchanged. `Component` interface unchanged. `Context` struct unchanged.

## 4. Domain Model

```
kernel.ProjectIDFromContext(ctx) → (int64, bool)
kernel.WithProjectID(ctx, projectID) → context.Context
kernel.ProjectIDKey — typed context key (unexported)
```

No new tables. Purely kernel-level utility code.

## 5. Contract / Interface

```go
// kernel/scope.go

type projectIDKeyType string

const projectIDKey projectIDKeyType = "project_id"

// WithProjectID returns a child context with the project ID set.
func WithProjectID(ctx context.Context, projectID int64) context.Context

// ProjectIDFromContext extracts the project ID from context.
// Returns (0, false) when no project ID is set.
func ProjectIDFromContext(ctx context.Context) (int64, bool)
```

## 6. Implementation Strategy

Follow the existing pattern from `components/api/api.go` and `components/auth/auth.go` (context keys `ctxOrgID`, `ctxUserID`) but place them in the `kernel` package as the canonical source. This avoids the current duplication where `api` and `auth` define their own `ctxOrgID` keys.

## 7. Data Flow

```
auth.Middleware → verifyJWT(Claims.ProjectID) → WithProjectID(ctx, projectID)
                                                        ↓
api.ServeHTTP → ProjectIDFromContext(ctx) → add WHERE project_id = ? to queries
```

## 8. Error Handling

- `ProjectIDFromContext` returns `(0, false)` for missing key — caller decides behavior
- No panics. No log-in-helper. Callers log at their discretion.

## 9. Testing Strategy

- Unit: `TestProjectIDContextRoundTrip` — inject, extract, verify round-trip
- Unit: `TestProjectIDContextMissing` — verify `ok == false` when not set
- Unit: `TestProjectIDContextMultiple` — verify multiple WithProjectID calls chain correctly

## 10. Migration / Rollback

No database migration. Pure code addition. Rollback: delete `kernel/scope.go` — no callers yet in this story.

## 11. Documentation

Add `project_id` to the kernel package godoc. Update `tech-stack.md` context flow diagram.

## 12. Dependencies

- **Depends on:** Nothing (kernel has no dependencies)
- **Depended on by:** e52s02, e52s03, e52s04

## 13. Observability

None needed at kernel level. Components that use project scoping log at their own level.

## 14. Security

- Context key is unexported to prevent external packages from injecting spoofed project IDs
- Type-safe `projectIDKeyType` prevents collision with other string keys

## 15. Performance

Negligible — `context.WithValue` is O(1) and already used heavily in the request path.

## 16. Alternatives Considered

| Option | Decision |
|--------|----------|
| ScopedDBer wrapper that regex-rewrites SQL to add WHERE project_id | Rejected — fragile, false positives on JOINs, subqueries |
| Per-project SQLite database files | Rejected — breaks cross-project queries, complex migration |
| Kernel-level context helpers (this plan) | **Accepted** — follows existing org_id pattern |

## 17. Acceptance Criteria

```gherkin
Scenario: Project ID context round-trip
  Given no project ID in context
  When kernel.WithProjectID(ctx, 42) is called
  Then kernel.ProjectIDFromContext(ctx) returns (42, true)

Scenario: Missing project ID
  Given a context with no project ID set
  When kernel.ProjectIDFromContext(ctx) is called
  Then it returns (0, false)

Scenario: Key isolation
  Given an untyped string key "project_id" set in context
  When kernel.ProjectIDFromContext(ctx) is called
  Then it returns (0, false) — typed key prevents spoofing
```

## 18. Out of Scope

- ScopedDBer wrapper (deferred to e52s03)
- Project CRUD operations (e52s02)
- Auth token project_id claims (e52s04)
- Actually adding project_id WHERE clauses to queries (e52s03, e52s04)

## 19. Risks

- **Risk:** Existing `api` and `auth` packages have their own `ctxOrgID` keys — they could diverge from kernel versions.
- **Mitigation:** This story does NOT migrate existing `ctxOrgID` usage. That's a separate refactor (out of scope). The kernel's `ProjectID` helpers are additive and don't conflict.

## 20. Verification Script

1. Run `go test ./kernel/ -run TestProjectID -v` — all project ID context tests pass
2. Run `go build ./...` — no compilation errors
3. Confirm `kernel/scope.go` exists with `WithProjectID` and `ProjectIDFromContext` exported
4. Confirm context key is unexported (lowercase `projectIDKey`)
