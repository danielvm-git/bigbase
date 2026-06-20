# Story e40s03: Instrument Database Segment Tracing

**type:** feat
**context:** infra
**bcps:** 2
**status:** done

## Context

Wrap database queries in New Relic datastore segments to provide visibility
into query performance. Each ExecContext, QueryContext, and QueryRowContext
call that occurs within an active NR transaction (set by e40s02 HTTP middleware)
creates a `DatastoreSegment` with query text.

## Implementation

- Added `nrDatastoreSegment` helper that reads NR transaction from context
- Wrapped `ExecContext`, `QueryContext`, `QueryRowContext` with optional segments
- Segments are nil-safe: background tasks without NR context get no segment
- Long queries truncated to 200 runes to limit payload size

## Acceptance Criteria

1. Build succeeds with db importing newrelic
2. All tests pass
3. DB queries within HTTP requests create NR datastore segments
