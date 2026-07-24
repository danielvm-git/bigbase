# e82 — Artifact-first deploy

## North star

One **artifact-upload deploy contract** shared by GitHub Actions (`bigbase-deploy`) and Admin UI Site Deploy. Both send the same multipart payload; BigBase unpacks and starts — **no host rebuild**.

## API contract (e82s01)

- `POST /api/sites/{id}/deploy` as `multipart/form-data`
- Fields: `artifact` (required in artifact mode), `app_type`, `ref`, `passthrough_paths`
- Missing/empty artifact → **4xx** + JSON `{ "error": "build artifact required", "hint": "..." }` and deploy log entry `"no build artifact found"`
- Legacy git-pull deploy may remain behind explicit `mode=git` for rollback

## Clients

| Client | Auth | e82 story |
|--------|------|-----------|
| GitHub Action | `bb_dep_*` site key | e82s03 |
| Admin UI file picker | JWT session | e82s04 |

## Stack handoff (host receives, does not rebuild)

| Stack | CI produces | Host does |
|-------|-------------|-----------|
| static / Vite / SvelteKit | `dist/` tarball | Serve static |
| Node SSR | build + prod deps or standalone | Start process |
| Go | linux/amd64 binary | chmod + run |
| Python | wheel / frozen app dir | Existing python path |
| PHP | public tree | serve / php-fpm |

## Explicitly deferred

- GitHub App VCS webhooks
- Consumer repo migrations (tracked via consumer issues)
- `bigbase.toml` (e70) unless needed for passthrough_paths

## Prerequisite

Horizon A P0: deploy-key site ownership in `requireSiteOwnership` (BUG-2026-07-24-deploy-key-organization-required).
