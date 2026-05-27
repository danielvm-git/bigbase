# BigBase Release Plan — Vertical Slices

type: release-plan
context: BigBase BaaS — single-binary, ECC architecture, vertical slices
slopcheck: modernc.org/libc [OK human-approved — verified upstream at github.com/sqlite/sqlite, used by PocketBase], golang.org/x/sys [OK human-approved — official Go stdlib supplement], modernc.org/sqlite [OK human-approved — well-known pure-Go SQLite driver], github.com/mattn/go-isatty [OK human-approved — established Go ecosystem package by mattn]

## Vision
A single-binary BaaS you can see evolving every step of the way.

## Slices

### Slice 1: CLI — "BigBase Breathes"
- Kernel boots, registers components, shows lifecycle
- Commands: `bigbase version`, `bigbase status`, `bigbase components list`
- Visual: Terminal ANSI output with live component table
- Verify: `go run . --version` → `BigBase v0.1.0`

### Slice 2: Proxy + Landing Page — "See It in the Browser"
- `components/proxy/` — HTTP/HTTPS server, router, Let's Encrypt
- Serves a "BigBase is running" styled page on localhost
- Visual: Open `http://localhost:443` in browser → animated welcome page
- Verify: `curl http://localhost:443/api/ping` → `{"status":"alive","version":"0.1.0"}`

### Slice 3: DB + Auto API — "See JSON"
- `components/db/` — SQLite init, migrations runner
- `components/api/` — REST CRUD auto-generated from schema
- Visual: `GET /api/health` → full status JSON
- Verify: `curl http://localhost/api/collections/users` → `{"data":[]}`

### Slice 4: Auth — "See Login"
- `components/auth/` — email/password, JWT, session management
- Login page served by proxy, registration form
- Visual: Browser form → sign up → dashboard
- Verify: `curl -X POST /api/auth/login -d '{"email":"a@b.com","password":"x"}'` → JWT token

### Slice 5: Admin UI — "See the Dashboard"

**type:** feat  
**context:** BigBase admin panel — React SPA embedded in Go binary via `//go:embed`, served at `/admin/`.

#### Reference: PocketBase approach (opensrc)
PocketBase embeds its vanilla JS SPA at `/_/` using `//go:embed all:dist` with `fs.Sub()` to strip the base path. Key patterns adopted here:
- Hash routing (`#/login`, `#/collections`) — no server-side SPA fallback needed
- Vite `base: "./"` — relative asset paths resolve correctly under any base URL
- No auth on static files; auth enforced client-side + on API endpoints
- `ui/dist/` committed to Git (required by `//go:embed`)

#### Implementation

- `components/admin/` — Go component embedding the SPA via `//go:embed all:dist`, using `fs.Sub()` to strip prefix, served via `http.FileServer()`
- `ui/` — Vite + React + TypeScript SPA with hash routing (`#/login`, `#/`, `#/data`)
- Auth via JWT in localStorage, sent as Bearer header to existing API endpoints
- SPA routes: `#/login` → login/register form, `#/` → dashboard, `#/data` → data browser
- No auth middleware on `/admin/` static files (enforced client-side + API layer)
- Vite config: `base: "./"` for relative asset paths
- Verify: Open http://localhost:9999/admin → login → CRUD tables visually

### Out of scope for Slice 5 (core)
- Full admin section management (deferred to later slices)

### Slice 5.1: SQL Editor

**type:** feat  
**context:** In-browser SQL query tool in the admin UI. Send read-only queries to the database and view results as a table.

#### Backend
- `POST /api/sql` endpoint in `components/api/api.go`
- Accepts `{"query": "SELECT ..."}` JSON body
- Validates read-only: only `SELECT`, `EXPLAIN`, `PRAGMA` queries allowed (rejects DDL/DML)
- 10-second query timeout
- Returns `{"columns": [...], "rows": [{...}, ...]}` JSON
- Protected by auth middleware (same as `/api/collections/`)

#### Frontend
- `SqlEditorPage.tsx` at `#/sql`
- Textarea with default query (`SELECT name FROM sqlite_master WHERE type='table'`)
- "Run" button, keyboard shortcut `⌘⏎` / `Ctrl+Enter`
- Results table with dynamic columns
- Error display for invalid SQL or DML attempts
- Nav link in sidebar Layout

#### Verify
```bash
open http://localhost:9999/admin/#/sql
# Default query shows table list → click Run
# Type SELECT * FROM posts → Run → see records
# Type DROP TABLE posts → Run → see error
```

#### Implementation
1. Add `POST /api/sql` handler to `components/api/api.go` with read-only validation and JSON response
2. Add route registration in `Handler()` and wire through auth middleware in `main.go`
3. Create `SqlEditorPage.tsx` with textarea, Run button, results table, error state
4. Add route (`#/sql`) to `App.tsx` and nav link to `Layout.tsx`
5. Rebuild UI → `go test ./...` passes, `go build` succeeds

### Slice 5.2: User Management

**type:** feat  
**context:** User management page in the admin UI. List and delete registered users.

#### Backend
- `GET /api/auth/users` — list all users (id, email, created_at), requires auth
- `DELETE /api/auth/users/:id` — delete a user, requires auth
- Auth enforced internally via JWT check (not through external middleware)

#### Frontend
- `UsersPage.tsx` at `#/users`
- Table with columns: ID, Email, Created
- Delete button per row with confirmation dialog
- Refresh button to reload user list
- Nav link in sidebar Layout

#### Verify
```bash
open http://localhost:9999/admin/#/users
# See list of registered users
# Click Delete on a user → confirm → user removed from list
```

#### Implementation
1. Add `QueryContext` to auth's `DBer` interface
2. Add `GET /api/auth/users` and `DELETE /api/auth/users/:id` handlers to `auth.go`
3. Register routes in `Handler()`
4. Write tests for list, delete, unauthenticated access, not-found, wrong-method
5. Create `UsersPage.tsx` with table, delete, refresh
6. Add route (`#/users`) to `App.tsx` and nav link to `Layout.tsx`

## Steps

1. Scaffold Vite + React + TypeScript app in `ui/` with `base: "./"` in vite.config.ts → verify: `npm run build --prefix ui` exits 0, `ls ui/dist/index.html`

2. Add hash-based routing (`react-router-dom` with `HashRouter`), create placeholder pages: Login, Dashboard, NotFound. Vite config: `base: "./"`, `build.outDir: "dist"`. Commit `ui/dist` to git → verify: open `ui/dist/index.html` directly in browser shows placeholder content (relative paths resolve)

3. Create `components/admin/admin.go` — Go component that embeds `ui/dist/` via `//go:embed all:dist`, strips prefix with `fs.Sub()`, serves via `http.FileServer()`. Implements `kernel.Component` interface. The admin handler serves all requests with `http.FileServer` + index.html fallback for directory roots. No auth middleware on static files → verify: `go build ./components/admin/...`

4. Wire admin component in `main.go` — register with kernel, mount at `/admin/` via `p.Handle("/admin/", adminComp.Handler().ServeHTTP)` → verify: `go run . serve --port 9999` and `curl http://localhost:9999/admin/` returns HTML page

5. Add auth to SPA — login form (`#/login`), JWT in localStorage after register/login API call, `AuthContext` provider wrapping hash router, auto-redirect to `#/` on login, auto-redirect to `#/login` if no valid token → verify: open browser at `/admin`, see login form, register a user, see dashboard at `#/`

6. Add dashboard page (`#/`) — show user email decoded from JWT, navigation links to Data Studio, logout button that clears localStorage → verify: after login, dashboard shows user email and nav links

7. Add Data Studio page (`#/data`) — fetch collection list via `GET /api/collections/` with Bearer header, display as clickable list; on selection, fetch records via `GET /api/collections/:name/` and display in HTML table → verify: navigate to Data Studio, see collection list, click a collection to see its records

8. Admin component test — verify embedded FS serves files correctly, verify 404 for missing files, verify index.html is served for directory root → verify: `go test ./components/admin/... -count=1`

## Verification Script (Step-by-Step)

1. `cd ui && npm run build` → verify `ui/dist/index.html` exists
2. `go build -o bigbase .` → build succeeds
3. `go run . serve --port 9999` → server starts
4. Open `http://localhost:9999/admin/` in browser → login form appears
5. Click "Register", enter email/password → registration succeeds, redirected to `#/`
6. Dashboard shows user email and navigation links
7. Click "Data Studio" → list of collections shown
8. Click a collection → records displayed in a table
9. Click "Logout" → redirected to `#/login`

## Risks

- **Hash routing chosen** to avoid SPA fallback complexity on the server. Risk: URLs use `#/` fragments (less SEO-friendly, but acceptable for an admin panel).
- **Build order**: SPA must be built (`npm run build`) before Go build. The `setup.sh` script should be updated. `ui/dist/` is committed to Git so `go build` works on fresh checkout without npm.
- **No CORS issues**: SPA and API share the same Go server port.
- **Vite base path**: `base: "./"` is critical — without it, asset paths would be absolute and fail under `/admin/` subpath.

### Slice 6: Storage — "See Files"

**type:** feat
**context:** File upload, storage, and retrieval. Local filesystem backend with metadata in SQLite.

#### Scope
- `POST /api/storage/upload` — multipart file upload (MIME validated, size limited)
- `GET /api/storage/files/:id` — download file by UUID
- `GET /api/storage/files` — list uploaded files with metadata
- `DELETE /api/storage/files/:id` — delete file + metadata
- Image thumbnails deferred to Slice 6.1

#### Implementation

- `components/storage/` — Go component implementing `kernel.Component`
- Files stored at `data/storage/<uuid>/<filename>` on local filesystem
- Metadata in SQLite `storage_files` table (id, name, size, mime_type, path, created_at)
- File IDs as hex-encoded random bytes (no external UUID dependency)
- All routes behind auth middleware
- Max upload size: 10 MB (configurable)
- MIME type detection via `net/http` `DetectContentType`

## Steps

1. Create `components/storage/storage.go` with Component interface, `FileInfo` struct, auto-migration for `storage_files` table, file ID generation via `crypto/rand`, and `New()` constructor → verify: `go build ./components/storage/...`

2. Add component compliance test for Name, Version, Dependencies, Hooks → verify: `go test ./components/storage/... -count=1`

3. Implement `POST /api/storage/upload` — parse multipart form, validate MIME type, generate UUID filename, save to `data/storage/`, store metadata in SQLite, return JSON with file info → verify: `curl -X POST /api/storage/upload -F "file=@testdata/test.txt"` returns 201 with file ID

4. Add tests for upload: success, missing file, wrong field name, file too large → verify: `go test ./components/storage/... -count=1`

5. Implement `GET /api/storage/files/:id` — look up metadata by ID, serve file with correct Content-Type and Content-Disposition → verify: `curl /api/storage/files/<id>` returns the file content with correct MIME type

6. Add tests for download: success, not found → verify: `go test ./components/storage/... -count=1`

7. Implement `GET /api/storage/files` — query all `storage_files` rows, return as JSON array → verify: `curl /api/storage/files` returns list

8. Implement `DELETE /api/storage/files/:id` — delete metadata row + file from disk, return 404 if not found → verify: `curl -X DELETE /api/storage/files/<id>` returns 200, subsequent download returns 404

9. Add tests for list and delete: empty list, list after upload, delete not found → verify: `go test ./components/storage/... -count=1`

10. Wire in `main.go` — register storage component, mount routes behind `protectedAPI` → verify: `go build .` succeeds, `go run . serve --port 9999` starts without error

## Verification Script

```bash
# Start server
go run . serve --port 9999 &

# Get token
TOKEN=$(curl -s -X POST http://localhost:9999/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"s@b.com","password":"secret123"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")

# Upload
echo "hello world" > /tmp/test.txt
UPLOAD=$(curl -s -X POST http://localhost:9999/api/storage/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/tmp/test.txt")
echo "$UPLOAD"
ID=$(echo "$UPLOAD" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

# Download
curl -s -o /tmp/downloaded.txt \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:9999/api/storage/files/$ID
cat /tmp/downloaded.txt  # → "hello world"

# List
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:9999/api/storage/files

# Delete
curl -s -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:9999/api/storage/files/$ID

# Verify deleted
curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:9999/api/storage/files/$ID  # → 404
```

## Out of scope
- Image thumbnails (deferred to Slice 6.1)
- S3/cloud storage backend
- Admin UI file browser page (deferred)

## Risks
- Disk space: no quota enforcement yet. Files accumulate until explicitly deleted.
- No external storage dependency — files live on the server filesystem.

### Slice 7: Git — "See Repos"
- `components/git/` — SSH server, repo create/clone/push, LFS
- Visual: `git clone ssh://localhost:2222/myrepo`
- Verify: `git clone` → work on files → `git push` → reflected in admin UI

### Slice 8: Forge — "See Issues"
- `components/forge/` — issues, PRs, kanban board, wiki
- Visual: Admin UI shows kanban with issue cards
- Verify: `POST /api/forge/issues` → issue appears in kanban

### Slice 9: CI/CD — "See Pipelines"
- `components/cici/` — workflow parser, runner
- Visual: Pipeline status in admin UI
- Verify: `git push` → workflow runs → green checkmark

### Slice 10: Functions — "See Code Run"
- `components/functions/` — JS runtime (goja), scheduler
- Visual: Write a function in admin UI → see it execute on trigger
- Verify: `POST /api/functions/run` → function output in response

### Slice 11: Realtime — "See Live Updates"
- `components/realtime/` — WebSocket subscriptions, broadcast, presence
- Visual: Browser shows live-updating data without refresh
- Verify: `wscat connect ws://localhost/realtime` → receives mutations

### Slice 12: Messaging — "See Notifications"
- `components/messaging/` — push (FCM/APNs), email (SMTP), SMS
- Visual: Send test email from admin UI
- Verify: `POST /api/messaging/email` → email arrives in inbox

### Slice 13: Deploy — "See Apps Live"
- `components/deploy/` — 1-click deploy, DB provisioning, branch preview
- Visual: Deploy a Next.js app from admin UI with 1 click
- Verify: Open preview URL → app is running

### Slice 14: Monitoring — "See Metrics"
- `components/monitoring/` — metrics, logs, uptime dashboard
- Visual: Grafana-like dashboard in admin UI
- Verify: Dashboard shows CPU, requests, error rates
