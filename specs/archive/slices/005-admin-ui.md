# Slice 5: Admin UI — "See the Dashboard"

**type:** epic  
**status:** done (branch: `admin-slice`)  
**verify:** Open `http://localhost:9999/admin/` → login → browse collections

## Purpose

React SPA embedded into the Go binary via `//go:embed`. Provides a browser-based admin panel for managing data.

## Implementation

### Architecture

- **Load strategy:** SPA is pre-built (`npm run build`), output in `ui/dist/`, embedded at compile time via `//go:embed all:dist`
- **Mount:** served at `/admin/` prefix via `http.StripPrefix("/admin/", ad.Handler())`
- **Routing:** Hash-based (`#/login`, `#/`, `#/data`) — no server-side SPA fallback needed
- **No auth middleware on static files** — auth enforced client-side + on API endpoints

### Go Component (`components/admin/admin.go`)

```go
// Embed SPA
//go:embed all:dist
var distFS embed.FS

// Serve embedded files
func (ad *Admin) Handler() http.Handler {
    return http.FileServer(http.FS(ui.DistFS))
}
```

### React SPA (`ui/`)

| Route | Page | Description |
|-------|------|-------------|
| `#/login` | LoginPage | Login/Register form, stores JWT in localStorage |
| `#/` | DashboardPage | User info from JWT, navigation links |
| `#/data` | DataStudioPage | Browse collections and records |
| (404) | NotFoundPage | Catch-all |

- **Stack:** Vite + React 19 + TypeScript + react-router-dom (HashRouter)
- **Assets:** relative path (`base: "./"` in vite.config.ts)
- **Build:** `npm run build` outputs to `ui/dist/`

### API Addition — `GET /api/collections/`

Lists all user-created tables (excludes system tables, `users`). Added to `components/api/api.go` to support Data Studio sidebar.

## Build Order

```bash
cd ui && npm ci && npm run build  # First: build SPA
cd .. && go build -o bigbase .     # Then: embed + compile
```

`ui/dist/` is committed to Git so `go build` works on fresh checkout without Node.js.

## Out of Scope (deferred)

- SQL Editor → Slice 5.1
- User management page → Slice 5.2

## Verify

```bash
go run . serve --port 9999 &
open http://localhost:9999/admin/
# Register → Dashboard → Data Studio → Logout
```

## Files

```
components/admin/
├── admin.go
└── admin_test.go
ui/
├── index.html
├── vite.config.ts
├── package.json
├── embed.go
├── src/
│   ├── main.tsx
│   ├── App.tsx
│   ├── Layout.tsx
│   ├── index.css
│   └── pages/
│       ├── LoginPage.tsx
│       ├── DashboardPage.tsx
│       ├── DataStudioPage.tsx
│       └── NotFoundPage.tsx
└── dist/
    └── ... (built assets)
```
