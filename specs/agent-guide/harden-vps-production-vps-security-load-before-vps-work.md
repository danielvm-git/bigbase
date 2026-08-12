# harden-vps — Production VPS Security (LOAD BEFORE VPS work)

source: AGENTS.md
references: [AGENTS.md]

## harden-vps — Production VPS Security (LOAD BEFORE VPS work)

**ALWAYS load `.agents/skills/harden-vps/SKILL.md` before SSHing into the production VPS.** The skill contains the three-layer hardening model, gotchas (crontab % escaping, fail2ban missing-log crash, BigBase alerts require SQLite insert), and the 8-gate verification matrix.
