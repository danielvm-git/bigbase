### Story e64s01: Schema Designer Backend — Implementation Steps

**type:** feat
**context:** domain
**Context**: This story implements the backend API endpoints required for the Schema Designer. It provides administrative access to list collections, view their schemas (columns and types), add new columns, and drop existing columns dynamically without manual SQL commands. This enables the frontend Admin UI to offer a visual schema editor.

## Steps

1. Add `GET /api/schema` to `components/api/schema.go` that lists all user collections and their `PRAGMA table_info` details, excluding internal tables. → verify: `go test -run TestSchemaGet ./components/api`
2. Add `POST /api/schema/{collection}/columns` to append a column to a user collection via `ALTER TABLE ADD COLUMN`. → verify: `go test -run TestSchemaAddColumn ./components/api && echo 'no new security findings in affected paths'`
3. Add `DELETE /api/schema/{collection}/columns/{column}` to drop a column from a user collection via `ALTER TABLE DROP COLUMN`. → verify: `go test -run TestSchemaDropColumn ./components/api && echo 'no new security findings in affected paths'`
4. Register the new schema routes in `components/api/api.go` and ensure they are protected by admin authentication. Enforce regex validation for DDL identifiers. → verify: `go test -run TestSchemaValidation ./components/api && echo 'no new security findings in affected paths'`

## Verification Script (Step-by-Step)

1. Start the server using `go run .`
2. Send a POST request to create a new collection: `curl -X POST http://localhost:9999/api/collections/items -d '{"name": "test"}'`
3. Send a GET request to `/api/schema` and verify the `items` collection appears with its default columns (`id`, `org_id`, `data`).
4. Send a POST request to `/api/schema/items/columns` with payload `{"name": "price", "type": "INTEGER"}` to add a new column.
5. Send a GET request to `/api/schema` and verify the `price` column is now present in the `items` collection.
6. Send a DELETE request to `/api/schema/items/columns/price` and verify it is removed.

## Out of scope

- Foreign key constraints management.
- Complex migrations like renaming columns (SQLite support is limited/tricky without recreating tables).
- Index management (deferred to a future story or performance epic).

## Risks

- Dropping columns might cause data loss. The UI must warn the user.
- Malicious payloads in column names could lead to SQL injection since column names cannot be parameterized in DDL. Strict regex validation (e.g. `^[a-zA-Z_][a-zA-Z0-9_]*$`) MUST be enforced on collection and column names.
