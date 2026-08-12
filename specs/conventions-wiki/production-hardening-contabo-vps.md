# Production Hardening (Contabo VPS)

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

### Production Hardening (Contabo VPS)
The production VPS at Contabo (vmi3338033, 89.116.26.187) follows a three-layer hardening model. See `.agents/skills/harden-vps/SKILL.md` for the full guide and `.agents/skills/harden-vps/REFERENCE.md` for script templates.

**Layer 1 — Ubuntu OS:** UFW (22,80,443 only), fail2ban SSH jail (maxretry=3, bantime=1h), unattended-upgrades (security-only, 4AM reboot), SSH (no root, no password).

**Layer 2 — BigBase:** systemd unit runs as `bigbase` user with NoNewPrivileges, ProtectSystem=full, ProtectKernelTunables, RestrictAddressFamilies, LimitNOFILE=65536. Monitoring alerts (disk>80%, CPU>90%, RAM>85%) via SQLite. Daily SQLite backup at 2AM with 90-day rotation.

**Layer 3 — Contabo VPS:** Health check script every 5min (disk, RAM, BigBase alive, reboot flag). cntb CLI for API access. Monthly snapshot on the 1st at 4AM (reads credentials from `/opt/bigbase/.env` — same file used by BigBase systemd, deployed by GitHub Actions). See `.agents/skills/harden-vps/` for the full guide.

**Verification:** Run the 8-gate check in `harden-vps` SKILL.md after any VPS or configuration change.
