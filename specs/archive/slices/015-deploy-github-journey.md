# 015 — Sites: Deploy from GitHub (UI-first slices)

**status:** done  
**verify:** See slice verification blocks below

## Problem

Deploy is split across **Git Repos** and **Deploy** with no guided “create site from GitHub” journey. This epic delivers Appwrite Sites–inspired UI first, then wires existing APIs, then full GitHub App integration.

## Slices

| Slice | Goal | Backend |
|-------|------|---------|
| **A** | Layout, wizard, mock/preview | None |
| **B** | Deploy from BigBase git repos | `/api/deploy`, `/api/git/repos` |
| **C** | GitHub connect + repo picker | `components/github`, `/api/sites` |
| **D** | Auto-deploy on push | GitHub webhooks |

## Routes (admin UI)

| Route | Page |
|-------|------|
| `#/deploy` | Sites list |
| `#/deploy/new` | Create site wizard |
| `#/deploy/:siteId` | Site detail |

Preview: `#/deploy?preview=1` or `VITE_SITES_PREVIEW=1` in `ui/.env.development`.

## Slice A verification (no Go)

```bash
cd ui && npm run dev
# http://localhost:5173/admin/#/deploy?preview=1
cd ui && npm run build
```

**Acceptance:** Empty list, populated list, wizard steps 1–4, site detail, preview banner, skeletons.

## Slice B verification

```bash
go test ./components/deploy/... ./components/git/...
go run . serve --port 9999
# Git Repos → create repo → Sites → existing repo path → deploy
```

## Slice C verification

```bash
go test ./components/github/...
# Configure --github-app-id, --github-app-private-key-path, --github-webhook-secret
# Sites → Connect GitHub → pick repo → deploy
```

## Slice D verification

Push to production branch on linked repo → new deployment without manual action.

## API (Slice C+)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/github/status` | GitHub App connected? |
| GET | `/api/github/install` | Redirect to install |
| POST | `/api/github/webhook` | Push events (Slice D) |
| GET | `/api/github/repos` | List repos |
| POST | `/api/github/repos/connect` | Mirror to internal git |
| GET | `/api/sites` | List sites |
| GET | `/api/sites/:id` | Site detail |
| POST | `/api/sites` | Create site + deploy |
| POST | `/api/sites/:id/deploy` | Redeploy |
| POST | `/api/deploy` | Legacy deploy (unchanged) |

## Appwrite reference

[Sites quick start](https://appwrite.io/docs/products/sites/quick-start) — Create site, connect GitHub, branch, deploy, preview URL.
