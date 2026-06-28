# e58s01: SQL-over-HTTP Transport

**Story ID:** e58s01 | **Epic:** e58 — Native Feature Port | **BCPs:** 3 | **Status:** planned

## 1. Type & Context

**type:** feat
**context:** domain
**maturity:** 3 — Countable

## 2. Story Statement

**As a** developer using the `@neondatabase/serverless` client or any Neon-wire-compatible HTTP SQL driver,
**I want** BigBase to accept queries in the Neon SQL-over-HTTP format,
**so that** I can point my existing Neon-targeted code at a self-hosted BigBase instance without changing client logic.

## 3. Context

BigBase already exposes `POST /api/sql` (admin-only, org-scoped, read-only) via `components/api/api.go:484`. That endpoint accepts `{"query": "..."}` and returns raw row arrays.

The Neon SQL-over-HTTP wire format differs in two ways:
1. **Request:** `{"query": "SELECT $1::text AS g", "params": ["hello"]}` — parameterised, with `$N` placeholders.
2. **Response:** `{"rows": [...], "fields": [{"name": "g", "dataTypeID": 25}], "rowCount": 1, "command": "SELECT", "rowAsArray": false}` — structured with column metadata.

This story adds a new endpoint `/api/sql/neon` in the existing `api` component that speaks this wire format. The existing `/api/sql` contract is untouched.

For SQLite, `$N` placeholders are converted to `?` before execution; `dataTypeID` is approximated from the Go runtime type of the first non-nil value in each column. For PostgreSQL, `$N` is passed through unchanged and `dataTypeID` is set to `0` (unknown) since `database/sql` does not expose PG column OIDs.

### Zoom-Out Summary
- **Module purpose:** `components/api` provides auto-CRUD for user collections and the admin SQL endpoint.
- **Callers:** `main.go` registers `/api/sql` via the existing `sqlHandler` chain; the new route reuses the same chain. No other component calls into `api` directly.
- **Contracts preserved:** `api.API` struct interface unchanged. Existing `/api/sql` behaviour unchanged. New `handleNeonSQL` method added; new route registered in `Handler()` and `main.go`.

## 4. Domain Model

No new tables. No schema changes.

```
POST /api/sql/neon
  Request:  {"query": string, "params": any[]}
  Response: NeonResult

NeonResult {
  rows:       []map[string]any   // one map per row, keys are column names
  fields:     []NeonField        // column metadata
  rowCount:   int
  command:    string             // "SELECT", "INSERT", "UPDATE", "DELETE"
  rowAsArray: bool               // always false
}

NeonField {
  name:       string
  dataTypeID: int                // approximate PG type OID; 0 when unknown
}
```

## 5. Contract / Interface

```go
// components/api/neon.go

type NeonResult struct {
    Rows       []map[string]any `json:"rows"`
    Fields     []NeonField      `json:"fields"`
    RowCount   int              `json:"rowCount"`
    Command    string           `json:"command"`
    RowAsArray bool             `json:"rowAsArray"`
}

type NeonField struct {
    Name       string `json:"name"`
    DataTypeID int    `json:"dataTypeID"`
}

// convertDollarParams rewrites $1, $2, ... → ?, ?, ... for SQLite.
func convertDollarParams(query string) string

// goTypeToDataTypeID maps a Go runtime type name to an approximate PG OID.
// Returns 0 for unknown types.
func goTypeToDataTypeID(typeName string) int

// handleNeonSQL handles POST /api/sql/neon.
func (a *API) handleNeonSQL(w http.ResponseWriter, r *http.Request)
```

New route in `Handler()`:
```go
mux.HandleFunc("/api/sql/neon", a.handleNeonSQL)
```

New route registration in `main.go` (same `sqlHandler` chain as `/api/sql`):
```go
p.Handle("/api/sql/neon", sqlHandler.ServeHTTP)
```

## 6. Implementation Strategy

1. Create `components/api/neon.go` with `NeonResult`, `NeonField`, `convertDollarParams`, `goTypeToDataTypeID`.
2. Add `handleNeonSQL` on `*API`: POST-only, decode body, apply the same security rules as `handleSQL` (admin role check, internal table deny-list, single-statement, read-only first word, 1 MB body limit, 10 s timeout), convert `$N` → `?` for SQLite using a `driver` field on `API`, execute with params, scan rows into `NeonResult`.
3. Add a `driver string` field to `api.API` and populate it in `Init` from config (or from an option passed at construction) so `handleNeonSQL` can branch on `"sqlite"` vs `"postgres"`.
4. Register the route in `Handler()` and `main.go`.
5. Write tests in `components/api/neon_test.go`.

## 7. Data Flow

```
POST /api/sql/neon
  → proxy middleware
  → auth.Middleware (JWT)
  → RequireAdmin (role check)
  → orgBridge (inject org_id)
  → api.handleNeonSQL
      → decode {"query", "params"}
      → security checks (admin, deny-list, single-stmt, read-only)
      → if driver=="sqlite": convertDollarParams(query)
      → db.QueryContext(ctx, query, params...)
      → scan rows → NeonResult{rows, fields, rowCount, command}
      → writeJSON 200 NeonResult
```

## 8. Error Handling

| Condition | HTTP Status | Body |
|-----------|-------------|------|
| Non-POST method | 405 | `{"error": "method not allowed"}` |
| Not admin role | 403 | `{"error": "forbidden"}` |
| Internal table access | 403 | `{"error": "access to internal table X denied"}` |
| Multi-statement query | 400 | `{"error": "only single-statement queries are allowed"}` |
| Non-read-only query | 400 | `{"error": "only read-only queries are allowed"}` |
| Invalid JSON body | 400 | `{"error": "invalid json or body too large"}` |
| Query execution error | 400 | `{"error": "query execution failed"}` |

## 9. Testing Strategy

- **Unit — `convertDollarParams`:** no-op on `?`-style queries; rewrites `$1`/`$2`; handles `$10`; does not touch `$N` inside string literals (best-effort, documented limitation).
- **Unit — `goTypeToDataTypeID`:** int64→23, float64→701, string→25, bool→16, unknown→0.
- **Integration (httptest):** admin JWT + valid SELECT → 200 NeonResult; no `params` field → treated as empty slice; non-admin JWT → 403; internal table query → 403; write query → 400; multi-statement → 400.

## 10. Migration / Rollback

No schema changes. Rollback = remove the two route registrations and delete `neon.go`.

## 11. Documentation

Update `specs/tech-architecture/tech-stack.md` API section to note `/api/sql/neon` Neon-wire-format compatibility.

## 12. Dependencies

- No new Go packages.
- No dependency on e52 (admin endpoint inherits org scoping; project-scoped isolation deferred).

## 13. Observability

- `a.logger.Info("neon-sql executed", "rows", rowCount, "org_id", orgID)` on success.
- `a.logger.Error("neon-sql failed", "error", err)` on execution failure (query truncated to 200 chars).

## 14. Security

**Security level:** low

- Inherits all `/api/sql` defences: admin-only, internal table deny-list, read-only first-word check, single-statement check, 1 MB body limit, 10 s timeout.
- Params passed as driver-level arguments — never interpolated into the query string — preventing SQL injection via params.
- `$N` → `?` conversion is positional, not value-substitution.

## 15. Performance

Admin-only, infrequent endpoint. Query result is materialised into `[]map[string]any` in memory. Same 10 s timeout as `/api/sql`. Acceptable for the agent/admin use case.

## 16. Alternatives Considered

- **Extend `/api/sql` to detect Neon format:** adds conditional branching and makes the existing response contract ambiguous. Rejected — separate route is cleaner.
- **Full Postgres TCP wire protocol:** incompatible with single-binary/zero-CGo model. Out of scope.

## 17. Acceptance Criteria

```gherkin
Scenario: Neon-format parameterised query succeeds
  Given an admin JWT and a SQLite BigBase instance with table "items(id, name)"
  When POST /api/sql/neon with {"query": "SELECT $1 AS val", "params": ["hello"]}
  Then HTTP 200 with rows=[{"val":"hello"}], fields=[{"name":"val","dataTypeID":25}], rowCount=1, command="SELECT"

Scenario: Non-admin is rejected
  Given a non-admin JWT
  When POST /api/sql/neon with a valid query
  Then HTTP 403

Scenario: Write query is rejected
  Given an admin JWT
  When POST /api/sql/neon with {"query": "INSERT INTO items VALUES (1)", "params": []}
  Then HTTP 400 with {"error": "only read-only queries are allowed"}

Scenario: Internal table access is blocked
  Given an admin JWT
  When POST /api/sql/neon with {"query": "SELECT * FROM users", "params": []}
  Then HTTP 403
```

## 18. Out of Scope

- Full Postgres wire protocol (TCP) compatibility.
- `rowAsArray: true` mode.
- Project-scoped isolation (deferred to e52).
- Neon-specific TLS/SNI routing.

## 19. Risks

- `$N` → `?` conversion may misfire on queries with `$N` inside string literals. Mitigated by the existing read-only guards and documented as a known limitation.
- `dataTypeID` is approximate on SQLite — Neon clients that branch on OIDs may behave unexpectedly. Document as known limitation.

## 20. Verification Script

```bash
go build ./components/api/ && go test ./components/api/ -run TestNeon -v -count=1
```
