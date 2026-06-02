# Refactoring Opportunities

## 1. DBer Interface Duplication

**Severity:** Medium
**Scope:** 6 components

Six components define their own `DBer` interface with identical method signatures:

- `components/monitoring/monitoring.go` (line 19)
- `components/storage/storage.go`
- `components/git/git.go`
- `components/forge/forge.go`
- `components/cici/cici.go`
- `components/functions/functions.go`

Each duplicates `ExecContext`, `QueryContext`, `QueryRowContext`, `Migrate`.

**Proposed fix:** Extract shared `DBer` interface into `kernel/dber.go` and have all
components import it. This is a prerequisite for Epic 017 (Multi-DB) since the
generalized interface is the foundation for driver abstraction.

**Benefit:** Single source of truth, easier to add new DB methods, enables driver
swapping at the kernel level.

## 2. Proxy Home Template Embedded in Go Source

**Severity:** Low
**Scope:** `components/proxy/proxy.go`

The commercial landing page is a ~200-line HTML string literal inside `proxy.go`.
This is hard to maintain — no syntax highlighting, no templating, no partials.

**Proposed fix:** Extract to `components/proxy/home_template.go` as a raw string
literal (backtick-quoted), or better, use `//go:embed` for a `.html` file in a
template directory.

**Benefit:** Better tooling support, separation of concerns, easier to preview.

## 3. Route Registration Coupling in main.go

**Severity:** Low
**Scope:** `main.go` (lines 89–191)

All component instantiation, route registration, and middleware wiring happen in
a single `startProxy()` function. Adding a new component requires touching 3+
sections of this function.

**Proposed fix:** Each component that provides HTTP routes should self-register
via a hook or a `RegisterRoutes(mux)` method called automatically by the kernel.
This is a larger architectural change but would make the composition root much
cleaner.

**Benefit:** New components don't require main.go changes. Follows Open/Closed principle.

## 4. Ad-Hoc Migration Pattern

**Severity:** Low
**Scope:** All components with `Start()` methods

Multiple components run `CREATE TABLE IF NOT EXISTS` in their `Start()` methods.
There is no version tracking, no rollback, no ordered migration.

**Proposed fix:** Epic 017-E addresses this with a proper versioned migration system.
Individual component migrations should migrate into a shared migration registry.

**Benefit:** Idempotent, reversible, auditable schema changes.

## 5. Latency Slice Fixed Size in Monitoring

**Severity:** Low
**Scope:** `components/monitoring/monitoring.go` (line 169)

Latency tracking uses a fixed-size slice with manual eviction. A circular buffer
or rolling window would be more memory-efficient and produce more accurate
percentile calculations.

**Proposed fix:** Replace with a circular buffer of configurable size (default 1000).

**Benefit:** O(1) append, no GC pressure from slice reallocation, accurate p99
from windowed data.

## 6. Test Coverage Gaps

**Severity:** Medium
**Scope:** Cross-component

While all packages have tests, coverage is not measured or enforced. Integration
tests between components (e.g., realtime receiving API mutations) are minimal.

**Proposed fix:** Epic 021 addresses this with coverage gates, contract tests,
and E2E tests.

**Benefit:** Prevent regressions, document expected behavior, enable confident refactoring.
