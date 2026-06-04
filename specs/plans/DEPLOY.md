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

### Deployed sites (subdomains)

| Item | Value |
|------|--------|
| Site URL pattern | `https://<site-slug>.bigbase.click/` |
| DNS | Wildcard A record `*.bigbase.click` → VPS IPv4 (`89.116.26.187`) |
| Caddy | `*.bigbase.click` block in `scripts/setup-vps.sh` (proxies to BigBase on `:8080`) |
| BigBase flag | `--sites-domain bigbase.click` or env `BIGBASE_SITES_DOMAIN` |

After deploying code, **redeploy each site** once so stored URLs and proxy host routes refresh (old rows may still say `http://localhost:…`).

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

`scripts/setup-vps.sh` installs **Node.js 20 LTS** (NodeSource), creates `/opt/bigbase/.npm`, and sets `HOME` / `NPM_CONFIG_CACHE` on the `bigbase` systemd unit so `npm install` works for the `--no-create-home` service user.

```bash
export CONTABO_HOST=89.116.26.187
ssh root@$CONTABO_HOST 'bash -s' < scripts/setup-vps.sh
```

After upgrading an existing VPS, re-run the script (idempotent), then `systemctl daemon-reload && systemctl restart bigbase`, and redeploy Node sites.

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
