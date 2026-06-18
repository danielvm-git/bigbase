# Story e30s03: Collections API — filter and sort

**type:** feat
**context:** domain
**epic:** e30 — Backend for Bots & Integrations
**bcps:** 3
**status:** planned
**wsjf:** 3.67 (BV=5 TC=3 RR=3 / JS=3)
**scope:** `components/api/api.go`, `ui/`

## Zoom-Out: `components/api/`

| Dimension | Detail |
|-----------|--------|
| **Purpose** | Auto-CRUD over JSON blob collections in SQLite. Creates tables on first access, exposes REST endpoints under `/api/collections/`. |
| **Callers** | HTTP clients, Admin UI. No internal callers outside proxy routing. |
| **Contracts** | `handleCollection`, `listRecords`, `getRecord`, `createRecord`, `updateRecord`, `deleteRecord`. Query params: `limit`, `offset`. Response: `{data: [...]}`. |

## Context

`GET /api/collections/:name` currently only accepts `?limit=N&offset=M`, orders fixed by `id`, and has no filtering. Users must use `/api/sql` with `json_extract(data, '$.field')` for any non-trivial query. This story adds `?filter` and `?sort` query params to make the collections API self-sufficient for common queries.

## Scope boundaries

- **In scope**: `?filter=key=value` (equality via json_extract), `?filter=key:op=value` operators, `?sort=field` (ASC) / `?sort=-field` (DESC), multiple filters chained with AND
- **Out of scope**: OR filters, nested field traversal (dot notation), full-text search, pagination beyond limit/offset

## Acceptance Criteria (§17)

### AC1: equality filter
**Given** a collection with records `{name: "alice"}` and `{name: "bob"}`
**When** `GET /api/collections/test?filter=name=alice`
**Then** returns only the record with name "alice"

### AC2: sort ascending
**Given** records with values 3, 1, 2 in field "priority"
**When** `GET /api/collections/test?sort=priority`
**Then** records returned in order 1, 2, 3 (ascending)

### AC3: sort descending
**Given** same records
**When** `GET /api/collections/test?sort=-priority`
**Then** records returned in order 3, 2, 1

### AC4: multiple filters (AND)
**Given** records `{status: "active", role: "admin"}` and `{status: "active", role: "user"}`
**When** `GET /api/collections/test?filter=status=active&filter=role=admin`
**Then** returns only the admin record

### AC5: operator support
**Given** records with values 10, 20, 30
**When** `GET /api/collections/test?filter=value:gt=15`
**Then** returns records with value 20, 30

### AC6: filter + sort combined
**Given** multiple matching records
**When** `GET /api/collections/test?filter=status=active&sort=name`
**Then** only active records, sorted by name ascending

### AC7: backward compatibility
**Given** existing clients using `?limit=10&offset=0`
**When** no filter or sort params provided
**Then** behavior unchanged (ordered by id, limit/offset apply)

## Out of scope

- OR filter chaining
- Nested path traversal (e.g., `filter=user.name=alice`)
- Full-text / LIKE search (operator `like` is basic pattern matching only)
- Index creation on JSON paths

## Risks

| Risk | Detection | Mitigation |
|------|-----------|------------|
| json_extract perf on large tables | Benchmark with 10k+ rows | Phase 1: accept limitation; Phase 2: expression indexes |
| SQL injection via field name | Test with `'; DROP TABLE` | Existing `sanitize()` validates alphanumeric + underscore |
| Operator parser breaks on colons in values | Test with `value:like` | Parse first `:` as operator separator only |
