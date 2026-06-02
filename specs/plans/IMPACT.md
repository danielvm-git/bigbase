# Impact Analysis: Epic 017 — Multi-DB Support

## Summary

Adding PostgreSQL support requires generalizing the database access layer across
all components that currently depend on SQLite directly. This analysis maps the
blast radius, identifies risk areas, and proposes a safe incremental approach.

## Dependency Map

```
Epic 017 changes:
    kernel/dber.go (NEW) ── shared interface extracted from 6 components
         │
         ├── components/db/ ── driver selection + PG implementation
         │
         └── Affected consumers:
              ├── components/monitoring/ (DBer interface)
              ├── components/storage/ (DBer interface)
              ├── components/git/ (DBer interface)
              ├── components/forge/ (DBer interface)
              ├── components/cici/ (DBer interface)
              ├── components/functions/ (DBer interface)
              ├── components/auth/ (direct *sql.DB usage)
              ├── components/api/ (direct *sql.DB usage)
              └── components/messaging/ (direct *sql.DB usage)
```

## Components That Touch the Database

| Component | DB Usage Pattern | Impact Level |
|-----------|-----------------|--------------|
| `components/db/db.go` | Core SQLite driver | **High** — needs driver abstraction |
| `components/auth/auth.go` | Uses `*db.DB` directly | **Medium** — uses custom `DBer` already |
| `components/api/api.go` | Uses `*db.DB` directly | **Medium** — needs interface |
| `components/monitoring/` | Own `DBer` interface | **Low** — swap import to kernel |
| `components/storage/` | Own `DBer` interface | **Low** — swap import to kernel |
| `components/git/` | Own `DBer` interface | **Low** — swap import to kernel |
| `components/forge/` | Own `DBer` interface | **Low** — swap import to kernel |
| `components/cici/` | Own `DBer` interface | **Low** — swap import to kernel |
| `components/functions/` | Own `DBer` interface | **Low** — swap import to kernel |
| `components/messaging/` | Uses `*db.DB` directly | **Medium** — needs interface |
| `components/deploy/` | Uses `*db.DB` directly | **Medium** — needs interface |

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Interface mismatch when replacing `*db.DB` with `DBer` | Low | Moderate | Write interface first, compile-test, then swap each consumer one at a time |
| SQLite-specific SQL in components (e.g., `datetime()` function) | Medium | Low | Replace with Go-side timestamp generation; PG uses `NOW()` |
| Migration differences between SQLite and PG | Medium | Medium | Use portable DDL (TEXT for timestamps, INTEGER for bools) |
| Performance regression on SQLite path | Low | Low | Keep SQLite path unchanged; interface is pass-through |
| Missing test coverage for PG path | High | High | Epic 021-B (contract tests) runs against both drivers |

## Safe Incremental Plan

1. **Extract `kernel/dber.go`** — define the shared interface. No behavior change.
2. **Swap low-impact consumers** — 6 components with their own `DBer` just change the import path. Compile-test after each.
3. **Adapt medium-impact consumers** — `auth`, `api`, `messaging`, `deploy` switch from `*db.DB` to `DBer`. SQL compatibility check.
4. **Build PostgreSQL driver** — new file `components/db/postgres.go`. Implements the same `DBer` interface.
5. **Add driver selection** — `db.New()` takes a driver config instead of a hardcoded SQLite path.
6. **Run full test suite** — both drivers, all tests pass.

## Files That Must Change

| File | Change |
|------|--------|
| `kernel/dber.go` | NEW: shared DBer interface |
| `components/db/db.go` | Implement `DBer`, add driver factory |
| `components/db/postgres.go` | NEW: PostgreSQL DBer implementation |
| `components/monitoring/monitoring.go` | Replace local `DBer` with `kernel.DBer` |
| `components/storage/storage.go` | Replace local `DBer` with `kernel.DBer` |
| `components/git/git.go` | Replace local `DBer` with `kernel.DBer` |
| `components/forge/forge.go` | Replace local `DBer` with `kernel.DBer` |
| `components/cici/cici.go` | Replace local `DBer` with `kernel.DBer` |
| `components/functions/functions.go` | Replace local `DBer` with `kernel.DBer` |
| `components/auth/auth.go` | Switch from `*db.DB` to `kernel.DBer` |
| `components/api/api.go` | Switch from `*db.DB` to `kernel.DBer` |
| `components/messaging/messaging.go` | Switch from `*db.DB` to `kernel.DBer` |
| `components/deploy/deploy.go` | Switch from `*db.DB` to `kernel.DBer` |
| `main.go` | Pass driver config to `db.New()` |

## Total Files Changed

14 files (2 new, 12 modified). No breaking changes to external API contracts.
