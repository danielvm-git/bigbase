# Slice 13: Deploy — "See Apps Live"

**type:** epic  
**status:** done  
**verify:** Open preview URL → app is running

## Purpose

One-click deployment of web applications. Provisions infrastructure, builds from Git repos, and serves the app with a public URL.

## Scope

- One-click deploy from Git repo
- Build detection (Node.js, Go, Python, static)
- Preview URL per deployment (UUID-based)
- Environment variable management
- Deployment logs and status
- Rollback to previous deployment
- Branch previews (deploy each push)

## Design Decisions

- Deployments run as child processes on the host
- Each deployment gets a unique port (PID-based), proxied via a subdomain
- Build logs streamed via WebSocket
- No container runtime — deploy directly to filesystem
- Preview URLs: `https://<hash>.bigbase.local`

## Implementation Plan

### components/deploy/deploy.go

```go
type Deploy struct {
    db      *db.DB
    proxy   *proxy.Proxy
    logger  Logger
    buildsDir string
}

type Deployment struct {
    ID        string    `json:"id"`
    RepoID    string    `json:"repo_id"`
    Branch    string    `json:"branch"`
    CommitSHA string    `json:"commit_sha"`
    Status    string    `json:"status"` // building, running, failed
    URL       string    `json:"url"`
    Port      int       `json:"port"`
    Env       map[string]string `json:"env,omitempty"`
    CreatedAt time.Time `json:"created_at"`
}
```

### Build Detection

| Project Type | Detection | Build Command | Start Command |
|-------------|-----------|---------------|---------------|
| Node.js | `package.json` exists | `npm run build` | `npm start` |
| Static | `index.html` exists | None | Serve files |
| Go | `go.mod` exists | `go build -o app` | `./app` |
| Python | `requirements.txt` | `pip install -r requirements.txt` | `python app.py` |

### API Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/deploy` | Yes | Deploy from repo |
| GET | `/api/deploy` | Yes | List deployments |
| GET | `/api/deploy/:id` | Yes | Get deployment |
| POST | `/api/deploy/:id/rollback` | Yes | Rollback |
| GET | `/api/deploy/:id/logs` | Yes | Build logs |

## Verify

```bash
# Deploy a repository
curl -X POST /api/deploy \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"repo_id":"myproject","branch":"main"}'

# Get deployment info (includes preview URL)
curl -H "Authorization: Bearer $TOKEN" /api/deploy/<id>

# Open preview URL in browser
open https://<hash>.bigbase.local
```
