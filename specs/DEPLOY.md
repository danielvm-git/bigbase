# BigBase Production Deployment (Contabo)

## Production URL

| Service | URL |
|---------|-----|
| Main site (HTTPS) | https://bigbase.click/ |
| Admin UI | https://bigbase.click/admin/ |
| Health | https://bigbase.click/health |
| Monitoring health | https://bigbase.click/api/monitoring/health |
| Backend (internal) | http://localhost:8080 (via Caddy reverse proxy) |

HTTPS is handled automatically by Caddy (Let's Encrypt). Update `/etc/caddy/Caddyfile` on the VPS if the domain changes.

## Infrastructure

| Item | Value |
|------|--------|
| Provider | Contabo Cloud VPS 20 SSD |
| IPv4 | `89.116.26.187` |
| Region | EU (Hub Europe) |
| Instance ID | `203338033` |
| SSH user | `root` (keys via Contabo API secrets) |

## VPS layout

```
/opt/bigbase/
├── bin/bigbase          # deployed binary
├── data/bigbase.db      # SQLite
├── backups/             # DB backups on deploy
├── releases/            # binary rollback history
├── .env                 # optional Google OAuth (from GitHub Secrets)
├── status.sh
└── rollback.sh
```

## CI/CD

- Workflow: [`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml)
- Triggers: push to `main`, manual `workflow_dispatch`
- GitHub environment: `production`
- Required secrets: `CONTABO_HOST`, `CONTABO_USER`, `CONTABO_SSH_KEY`

## One-time setup

```bash
export CONTABO_HOST=89.116.26.187
ssh root@$CONTABO_HOST 'bash -s' < scripts/setup-vps.sh
```

## Operations

```bash
# Status
ssh root@89.116.26.187 '/opt/bigbase/status.sh'

# Logs
ssh root@89.116.26.187 'journalctl -u bigbase -f'

# Rollback
ssh root@89.116.26.187 '/opt/bigbase/rollback.sh'
```

## Local readiness check

```bash
export CONTABO_HOST=89.116.26.187
./scripts/check-contabo.sh
```
