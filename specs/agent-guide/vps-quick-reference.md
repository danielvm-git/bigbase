# VPS Quick Reference

source: AGENTS.md
references: [AGENTS.md]

### VPS Quick Reference
- **IP:** 89.116.26.187 (vmi3338033)
- **User:** root (SSH key required)
- **Contabo Customer ID:** 15027696
- **BigBase service:** `systemctl status bigbase`
- **Health check:** `/opt/bigbase/scripts/healthcheck.sh`
- **Backups:** `/backup/bigbase-YYYYMMDD.db` (2AM daily, 90-day rotation)
- **Contabo creds:** `/opt/bigbase/.env` (CONTABO_CLIENT_ID, CONTABO_CLIENT_SECRET, CONTABO_API_USER, CONTABO_API_PASSWORD — deployed via GitHub Actions). Local dev: add to `.envrc`.
