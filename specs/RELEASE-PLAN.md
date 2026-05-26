# BigBase Release Plan — Vertical Slices

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
- `components/admin/` — React SPA embedded via `//go:embed`
- Data Studio, SQL Editor, user management
- Visual: Full admin interface in browser at `/admin`
- Verify: Open http://localhost/admin → login → CRUD tables visually

### Slice 6: Storage — "See Files"
- `components/storage/` — file upload/download, local filesystem
- Visual: Upload a file via admin UI → download it back
- Verify: `curl -X POST /api/storage/upload -F "file=@photo.png"` → file URL

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
