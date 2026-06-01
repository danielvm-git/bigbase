# BigBase Production Deployment (Contabo)

## Production URL

| Service | URL |
|---------|-----|
| Admin UI & API (HTTP) | http://89.116.26.187/ |
| Health (via Caddy) | http://89.116.26.187/health |
| Monitoring health | http://89.116.26.187/api/monitoring/health |

HTTPS and a custom domain are deferred until DNS is configured. Update `/etc/caddy/Caddyfile` on the VPS when ready.

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
