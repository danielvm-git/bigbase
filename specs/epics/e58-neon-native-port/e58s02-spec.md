# e58s02: Database Branching

**Story ID:** e58s02 | **Epic:** e58 — Native Feature Port | **BCPs:** 4 | **Status:** planned

## 1. Type & Context

**type:** feat
**context:** domain
**maturity:** 3 — Countable

## 2. Story Statement

**As a** developer or deploy pipeline,
**I want** to create named snapshots (branches) of a project's database state,
**so that** preview environments (e63) can each start from an isolated copy without affecting the live database.

## 3. Context

BigBase uses SQLite as its primary database (zero-CGo, single binary). A "branch" in this model is a named, read/write snapshot of the database file, tracked in metadata, and addressable by a connection string. This directly enables e63 (Preview Environments) where each preview needs its own isolated datastore.

Postgres CoW branching (Neon-style, using WAL/snapshot isolation) requires superuser privileges and external infrastructure incompatible with the single-binary model. The snapshot/clone approach (file copy for SQLite, `CREATE DATABASE ... TEMPLATE` for Postgres) is the correct scope.

For this story, SQLite branching is fully implemented. Postgres branching is explicitly out of scope (blocked by superuser requirement and the fact that BigBase's Postgres mode is secondary to SQLite).

The `db_branches` table is added in `components/auth` (where projects live, per e52) so branch records co-locate with project records. The `db` component gains a method to open a branch DB connection from a snapshot path.

### Zoom-Out Summary
- **Module purpose:** `components/auth` manages orgs, users, JWT, and (post-e52) projects. Adding branch metadata here keeps it co-located with project ownership.
- **Callers:** New HTTP handlers in `auth.ProtectedHandler()`; future `deploy` component (e63) calls `OpenBranch`; `mcp` migration tools (e58s04) will use branch DBs.
- **Contracts preserved:** Existing auth handlers untouched. `db.DB` gains `OpenBranch` — purely additive. `db.DBer` (kernel interface) is not extended.

## 4. Domain Model

```sql
CREATE TABLE IF NOT EXISTS db_branches (
    id          TEXT     PRIMARY KEY,   -- UUID v4
    project_id  TEXT     NOT NULL,      -- FK → projects.id (e52s02)
    name        TEXT     NOT NULL,
    description TEXT,
    sqlite_path TEXT,                   -- absolute path to SQLite snapshot file
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, name)
);
```

```
Branch {
    ID          string
    ProjectID   string
    Name        string
    Description string
    SQLitePath  string
    CreatedAt   time.Time
}
```

## 5. Contract / Interface

```go
// components/auth/branches.go

type Branch struct {
    ID          string    `json:"id"`
    ProjectID   string    `json:"project_id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    SQLitePath  string    `json:"sqlite_path,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
}

// CreateBranch snapshots the current SQLite DB to a new file under data/branches/,
// records the branch in db_branches, and returns the Branch.
func (a *Auth) CreateBranch(ctx context.Context, projectID, name, description string) (*Branch, error)

// ListBranches returns all branches for a project, ordered by created_at DESC.
func (a *Auth) ListBranches(ctx context.Context, projectID string) ([]Branch, error)

// DeleteBranch removes the db_branches record and deletes the snapshot file.
func (a *Auth) DeleteBranch(ctx context.Context, branchID string) error

// GetBranch returns a single branch by ID.
func (a *Auth) GetBranch(ctx context.Context, branchID string) (*Branch, error)
```

```go
// components/db/db.go (additive)

// OpenBranch opens a read/write SQLite connection to the snapshot file at path.
// Returns an error if the file does not exist or the driver is not sqlite.
func (d *DB) OpenBranch(path string) (*DB, error)
```

HTTP routes added to `auth.ProtectedHandler()`:
```
POST   /api/projects/{projectID}/branches
GET    /api/projects/{projectID}/branches
GET    /api/projects/{projectID}/branches/{branchID}
DELETE /api/projects/{projectID}/branches/{branchID}
```

## 6. Implementation Strategy

1. Add `db_branches` migration in `auth.Start()` (idempotent `CREATE TABLE IF NOT EXISTS`).
2. Implement branch CRUD in `components/auth/branches.go`.
   - `CreateBranch`: generate UUID, compute snapshot path `data/branches/{id}.db`, use `io.Copy` to copy the open SQLite file (via `VACUUM INTO` for a consistent snapshot), insert record.
   - `ListBranches`, `GetBranch`, `DeleteBranch`: standard SQL operations; `DeleteBranch` also `os.Remove`s the file.
3. Add `OpenBranch` to `components/db/db.go`: opens a new `*DB` with `DriverSQLite` and the given path.
4. Add HTTP handlers in `components/auth/branches_handler.go`; register routes in `ProtectedHandler()`.
5. Write unit tests and integration tests.

**SQLite consistent snapshot:** Use `VACUUM INTO 'path'` (supported since SQLite 3.27.0; modernc.org/sqlite includes 3.49+) — this produces a defragmented, consistent copy without needing to close the connection.

## 7. Data Flow

```
POST /api/projects/{projectID}/branches
  → auth.ProtectedHandler (JWT validated, project ownership checked)
  → auth.handleCreateBranch
      → validate name (non-empty, no path separators)
      → VACUUM INTO 'data/branches/{uuid}.db'
      → INSERT INTO db_branches
      → respond 201 Branch JSON

DELETE /api/projects/{projectID}/branches/{branchID}
  → auth.handleDeleteBranch
      → SELECT sqlite_path FROM db_branches WHERE id = ?
      → DELETE FROM db_branches WHERE id = ?
      → os.Remove(sqlite_path)
      → respond 204
```

## 8. Error Handling

| Condition | HTTP Status | Body |
|-----------|-------------|------|
| Duplicate branch name for project | 409 | `{"error": "branch name already exists"}` |
| Project not found or not owned | 404 | `{"error": "project not found"}` |
| Branch not found | 404 | `{"error": "branch not found"}` |
| Invalid name (empty or contains `/`) | 400 | `{"error": "invalid branch name"}` |
| Postgres driver (not supported) | 501 | `{"error": "database branching requires SQLite driver"}` |
| Snapshot file creation fails | 500 | internal log, `{"error": "failed to create branch"}` |

## 9. Testing Strategy

- **Unit — `CreateBranch`:** creates file at expected path, record in DB, returns correct Branch struct.
- **Unit — `DeleteBranch`:** removes record and file; returns 404 on non-existent branch.
- **Unit — `ListBranches`:** returns branches in descending created_at order; empty list when no branches.
- **Integration (httptest):** POST creates branch and file; GET lists branches; DELETE removes branch and file; duplicate name returns 409.
- **Regression:** `go test ./...` after all changes.

## 10. Migration / Rollback

Migration is idempotent (`CREATE TABLE IF NOT EXISTS`). Rollback: drop `db_branches` table, delete `data/branches/` directory, remove routes from `ProtectedHandler()`.

## 11. Documentation

Update `specs/tech-architecture/tech-stack.md` to note DB branching capability and `data/branches/` directory convention. Document `VACUUM INTO` as the snapshot mechanism.

## 12. Dependencies

- Depends on e52s02 (projects table) for `project_id` FK. The `db_branches.project_id` column is a TEXT FK; the migration adds the column without a FOREIGN KEY constraint to avoid SQLite FK enforcement complexity.
- `VACUUM INTO` requires SQLite 3.27.0+; `modernc.org/sqlite` v1.50.1 bundles SQLite 3.49+ — satisfied.
- `data/branches/` directory must be writable at runtime. `CreateBranch` calls `os.MkdirAll("data/branches", 0700)` before snapshot.

## 13. Observability

- `logger.Info("branch created", "id", branch.ID, "project", projectID, "path", path)` on success.
- `logger.Error("branch snapshot failed", "error", err)` on VACUUM INTO failure.
- `logger.Info("branch deleted", "id", branchID)` on delete.

## 14. Security

**Security level:** low

- Branch creation requires a valid project JWT (ProtectedHandler).
- Branch files written to `data/branches/{uuid}.db` — UUID in path prevents traversal.
- Name validation rejects `/`, `.`, and `..` segments.
- Snapshot contains the full DB; access to branch endpoints requires authentication — no public exposure.

## 15. Performance

`VACUUM INTO` on a typical BigBase SQLite (< 100 MB) completes in < 1 s. Branch list queries use the `(project_id, name)` unique index. No hot-path impact — branching is an admin/deploy-time operation.

## 16. Alternatives Considered

- **Postgres CoW branching (TEMPLATE DATABASE):** requires superuser, Postgres-only, not compatible with single-binary zero-CGo model. Out of scope.
- **Copy-on-write via write interception:** complex, untested path, no upstream library without CGo. Premature abstraction.
- **Schema-only branches:** would not give e63 data isolation for seeded preview data. Rejected.

## 17. Acceptance Criteria

```gherkin
Scenario: Create a branch
  Given a valid project JWT and a SQLite BigBase instance
  When POST /api/projects/{projectID}/branches with {"name": "preview-1"}
  Then HTTP 201 with Branch JSON including sqlite_path
  And a file exists at data/branches/{id}.db

Scenario: List branches
  Given two branches exist for a project
  When GET /api/projects/{projectID}/branches
  Then HTTP 200 with both branches in descending created_at order

Scenario: Delete a branch
  Given a branch "preview-1" exists
  When DELETE /api/projects/{projectID}/branches/{branchID}
  Then HTTP 204
  And the snapshot file is removed from disk

Scenario: Duplicate branch name rejected
  Given a branch "preview-1" already exists for a project
  When POST /api/projects/{projectID}/branches with {"name": "preview-1"}
  Then HTTP 409
```

## 18. Out of Scope

- Postgres database branching.
- Branch merging / diff.
- Branch-scoped SQL query execution (future story).
- Automatic branch creation on PR/deploy (e63).

## 19. Risks

- `VACUUM INTO` locks the DB briefly; for high-concurrency BigBase instances this may cause brief write contention. Acceptable for MVP; document as known behaviour.
- `data/branches/` growth is unbounded — caller must prune branches. Add `ListBranches` to surface this.

## 20. Verification Script

```bash
go build ./components/auth/ && go build ./components/db/ && go test ./components/auth/ -run TestBranch -v -count=1
```
