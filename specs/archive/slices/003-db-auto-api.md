# Slice 3: DB + Auto API — "See JSON"

**type:** epic  
**status:** done  
**verify:** `curl http://localhost:9999/api/collections/` → `{"data":["posts","comments"]}`

## Purpose

SQLite database component plus auto-generated REST CRUD API. Any table created is immediately accessible via REST.

## Implementation

### components/db/db.go

- Pure-Go SQLite via `modernc.org/sqlite` (no CGo)
- `MaxOpenConns(1)` — SQLite single-writer constraint
- Passthrough methods: `ExecContext`, `QueryContext`, `QueryRowContext`, `PrepareContext`
- `Migrate(migrations map[int]string)` — runs ordered migrations
- Auto-creates `bigbase.db` on first use

### components/api/api.go

- **Auto CRUD** for any `sqlite_master` table (excluding system tables)
- Methods: `GET /api/collections/:name` (list), `GET /api/collections/:name/:id`, `POST`, `PATCH`, `DELETE`
- Pagination via `?limit=` and `?offset=`
- Collection name sanitization — only `[a-zA-Z_][a-zA-Z0-9_]*`
- JSON blob columns — tables use SQLite JSON text columns
- `GET /api/collections/` — list all user collections

### Auth Enforcement

API handler wraps all collection routes behind `auth.Middleware`. Unauthenticated requests get 401.

## Configuration

```jsonc
{ "db": { "driver": "sqlite" } }
```

## Verify

```bash
curl http://localhost:9999/api/collections/       # list (requires auth)
curl http://localhost:9999/api/collections/posts   # list records
curl http://localhost:9999/api/collections/posts/1 # get by id
```

## Files

```
components/db/
└── db.go
components/api/
└── api.go
```
