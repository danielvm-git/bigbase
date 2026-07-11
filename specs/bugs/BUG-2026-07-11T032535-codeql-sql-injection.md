# Database query built from user-controlled sources (8 instances)

**Source:** GHS Code Scanning (CodeQL)
**Severity:** MAJOR
**CWE:** CWE-89 (SQL Injection)
**GitHub Alerts:** #8, #9, #10, #11, #12, #13, #14, #15

## Description
CodeQL detected 8 instances where SQL queries are built by concatenating user-controlled input. These are in the API component's endpoint handlers where filter/sort parameters are interpolated into SQL.

## Recommendation
Replace string concatenation with parameterized queries. Use query builders or prepared statements with bound parameters for all user-controlled WHERE, ORDER BY, and LIMIT clauses.

## Investigation — 2026-07-11

### Alert Locations

The 8 CodeQL alerts are on **passthrough wrapper methods** in the DB component. These are NOT injection points themselves — the risk depends on what callers pass to them:

**`components/db/db.go`** (alerts at lines 123, 131, 139, 174):
- `ExecContext(ctx, query, args...)` → passes through to `sql.DB.ExecContext`
- `QueryContext(ctx, query, args...)` → passes through to `sql.DB.QueryContext`
- `QueryRowContext(ctx, query, args...)` → passes through to `sql.DB.QueryRowContext`
- `Migrate(migration)` → calls `sql.DB.Exec(migration)`

**`components/db/postgres.go`** (alerts at lines 88, 93, 98, 118):
- Same 4 wrapper methods, backed by `pgxpool` via the `stdlib` adapter.

CodeQL correctly traces tainted user input through these wrappers to the database driver.

### Caller Analysis

All call sites were analyzed across the codebase. Results grouped by file:

---

#### 1. `components/api/api.go` — `collection` interpolated via `fmt.Sprintf` (7 CRUD methods + Migrate)

| Line | Method | SQL Pattern | User Input | Risk |
|------|--------|-------------|------------|------|
| 145 | `ensureTable` | `CREATE TABLE IF NOT EXISTS %s (...)` via Migrate | `collection` from URL path → sanitized | **needs-audit** |
| 149 | `ensureTable` | `ALTER TABLE %s ADD COLUMN ...` via Migrate | `collection` from URL path → sanitized | **needs-audit** |
| 242 | `listRecords` | `SELECT FROM %s%s%s LIMIT ? OFFSET ?` via QueryContext | `collection`, `whereClause`, `sortClause` → table/filter/sort field names sanitized; values parameterized | **needs-audit** |
| 294 | `getRecord` | `SELECT FROM %s WHERE id = ?` via QueryRowContext | `collection` from URL path → sanitized | **needs-audit** |
| 357 | `createRecord` | `INSERT INTO %s (...) VALUES (?, ?)` via ExecContext | `collection` from URL path → sanitized | **needs-audit** |
| 388 | `updateRecord` | `SELECT FROM %s WHERE id = ?` via QueryRowContext | `collection` from URL path → sanitized | **needs-audit** |
| 407 | `updateRecord` | `UPDATE %s SET ... WHERE id = ?` via ExecContext | `collection` from URL path → sanitized | **needs-audit** |
| 432 | `deleteRecord` | `SELECT FROM %s WHERE id = ?` via QueryRowContext | `collection` from URL path → sanitized | **needs-audit** |
| 443 | `deleteRecord` | `DELETE FROM %s WHERE id = ?` via ExecContext | `collection` from URL path → sanitized | **needs-audit** |
| 581 | `handleSQL` | Raw user SQL via QueryContext | Full SQL query string from JSON body | **needs-audit** |

**Guard:** `sanitize()` (line 739) restricts collection/field names to `[a-zA-Z0-9_]+`. Values are parameterized via `?` placeholders.
**handleSQL** is gated by: (1) admin role required, (2) read-only (SELECT/EXPLAIN/PRAGMA/WITH only), (3) blocks internal tables, (4) blocks JOIN/UNION/WITH/subqueries when org context present.
**Assessment:** Architecturally fragile — table names cannot be parameterized in SQL, so sanitize is the only defense. A bypass would be catastrophic. **Low risk in practice, but should be hardened.**

---

#### 2. `components/functions/runtime.go` — `name` interpolated via `fmt.Sprintf` (6 CRUD methods + Migrate)

| Line | Method | SQL Pattern | User Input | Risk |
|------|--------|-------------|------------|------|
| 172 | `injectDB` (closure) | `CREATE TABLE IF NOT EXISTS %s (...)` via Migrate | `name` from JS `db.collection("name")` → validated | **needs-audit** |
| 188 | `col.create` | `INSERT INTO %s (data) VALUES (?)` via ExecContext | `name` from JS code → validated | **needs-audit** |
| 200 | `col.list` | `SELECT FROM %s ORDER BY id` via QueryContext | `name` from JS code → validated | **needs-audit** |
| 231 | `col.get` | `SELECT FROM %s WHERE id = ?` via QueryRowContext | `name` from JS code → validated | **needs-audit** |
| 254 | `col.update` | `SELECT FROM %s WHERE id = ?` via QueryRowContext | `name` from JS code → validated | **needs-audit** |
| 272 | `col.update` | `UPDATE %s SET ... WHERE id = ?` via ExecContext | `name` from JS code → validated | **needs-audit** |
| 287 | `col.delete` | `DELETE FROM %s WHERE id = ?` via ExecContext | `name` from JS code → validated | **needs-audit** |

**Guard:** `validateCollectionName()` (line 345) restricts to `[a-zA-Z0-9_]+` — same pattern as `sanitize()`.
**Assessment:** Same architectural concern as api.go. JS function authors could theoretically craft function code with malicious collection names, but the validate function blocks all special SQL characters. **Low risk in practice.**

---

#### 3. `components/api/onboarding.go` — String concat with table name (2 call sites)

| Line | Method | Pattern | User Input | Risk |
|------|--------|---------|------------|------|
| 71 | `tableHasRows` | `"SELECT COUNT(*) FROM "+table` via QueryRowContext | Hardcoded as `"functions"` or `"users"` | **safe** |
| 86 | `tableHasMoreThanOneRow` | `"SELECT COUNT(*) FROM "+table` via QueryRowContext | Hardcoded as `"functions"` or `"users"` | **safe** |

---

#### 4. `components/sites/sites.go` — WHERE clause concatenation (2 call sites)

| Line | Method | Pattern | User Input | Risk |
|------|--------|---------|------------|------|
| 673 | `listSiteRequestLogs` | `"SELECT COUNT(*) FROM ... WHERE "+where` via QueryRowContext | `where` built from hardcoded status literals + parameterized path | **safe** |
| 675 | `listSiteRequestLogs` | `"SELECT ... WHERE "+where+" ORDER BY ..."` via QueryContext | Same as above | **safe** |

**Why safe:** `statusClass` is switched on hardcoded `"2xx"/"4xx"/"5xx"` strings. `pathPrefix` goes through `?` parameterization.

---

#### 5. `components/internal/eventrecorder/eventrecorder.go` — `?` placeholder construction

| Line | Method | Pattern | User Input | Risk |
|------|--------|---------|------------|------|
| 117 | `Query` | `query += "\` AND hook IN (` + strings.Join(placeholders, ",") + `)\`` via QueryContext | Placeholders are `"?"` strings; hook values are parameterized | **safe** |

---

#### 6. `components/deploy/env_vars.go` — Column name switch

| Line | Method | Pattern | User Input | Risk |
|------|--------|---------|------------|------|
| 25 | `fetchEnvVars` | `"SELECT ... WHERE site_id = ? AND " + col + " = 1"` via QueryContext | `col` is `"is_runtime"` or `"is_build_time"` — hardcoded | **safe** |

---

#### 7. `components/cici/runs.go` — Parameterized query building

| Line | Method | Pattern | User Input | Risk |
|------|--------|---------|------------|------|
| 165 | `handleRuns` | Incremental `query += "...?" + args` via QueryContext | All dynamic values parameterized: `repo_id`, `limit`, `offset` | **safe** |

---

#### 8. `components/monitoring/observability.go` — Hardcoded WHERE addition

| Line | Method | Pattern | User Input | Risk |
|------|--------|---------|------------|------|
| 327 | `handleIncidents` | `query += "\` WHERE resolved_at IS NULL\`` via QueryContext | Hardcoded string literal | **safe** |

---

#### 9. `components/backup/backup.go` — Table from database metadata

| Line | Method | Pattern | User Input | Risk |
|------|--------|---------|------------|------|
| 101 | backup routine | `fmt.Sprintf("SELECT * FROM %q", table)` via QueryContext | `table` from `sqlite_master` (actual DB table names) | **safe** |

---

### Migrate() callers

All 50+ `Migrate()` callers in production code pass hardcoded string literal SQL, **except** the two already identified in api.go and functions/runtime.go (which use sanitized collection names).

### Verdict

**Status: needs-audit — low risk, architectural hardening recommended**

| Category | Count | Risk |
|----------|-------|------|
| Safe call sites (hardcoded/parameterized) | ~60 | safe |
| Call sites needing audit (table name interpolation via sanitize) | ~16 | **needs-audit** (low) |
| Actual injection risk (uncontrolled concatenation) | 0 | none |

The 8 CodeQL alerts are **technically correct** (tainted input reaches the wrappers) but are **false positives in the practical sense** because:
1. The `sanitize()` / `validateCollectionName()` guards restrict table/field names to `[a-zA-Z0-9_]+`
2. All user-provided VALUES use `?` parameterization
3. The `handleSQL` endpoint is gated by admin role + read-only restrictions

The real vulnerability is architectural: **table names cannot be parameterized**, so every table-name interpolation is a single `sanitize()` bypass away from being a critical SQL injection. A defense-in-depth improvement would use a table ID lookup (e.g., map collection names to integer IDs) so that user input never appears in the SQL string at all.

## Status
needs-audit

## Source
seal.github_code_scanning

## Discovered
2026-07-11
