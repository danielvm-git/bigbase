# Security Review — e80 Production VPS Hardening

**Date:** 2026-07-12
**Scope:** cab966b1..c9d8e8e8 (14 files, all docs/specs)
**Reviewer:** AI agent via security-review skill

## Scan Summary

| Category | Count |
|----------|-------|
| SQL injection | 0 |
| XSS | 0 |
| SSRF | 0 |
| Command injection | 0 |
| Auth bypass | 0 |
| Unsafe deserialization | 0 |
| Path traversal | 0 |
| IDOR | 0 |
| Weak cryptography | 0 |
| Secrets exposure | 0 (all excluded) |
| Template injection | 0 |
| NoSQL injection | 0 |

**Verdict: PASS** — No code changes. All files are documentation (markdown/yaml). Zero security findings above confidence 8.

## Excluded Findings

| # | File | Finding | Confidence | Exclusion |
|---|------|---------|------------|-----------|
| 1 | AGENTS.md, SKILL.md | VPS IP 89.116.26.187 | 3 | Rule 18 (docs). Public IP, resolves from DNS. |
| 2 | AGENTS.md | Customer ID 15027696 | 2 | Rule 18 (docs). Semi-public, not a credential. |

## Operational Security Assessment

While no code vulnerabilities were found, the VPS audit revealed operational findings:

### FINDING-001: Near-Miss — Unprotected SSH on Production VPS

| Field | Detail |
|-------|--------|
| **Severity** | HIGH (operational, now remediated) |
| **Category** | Missing security control |
| **Confidence** | 10 |
| **Description** | On 2026-07-12, the Contabo VPS (vmi3338033, 89.116.26.187) had NO fail2ban installed, SSH password authentication enabled, and PermitRootLogin enabled. Active brute-force attempts were found in system logs (4 IPs banned retroactively after fail2ban installation). |
| **Timeline** | Connection at ~23:25 CEST → fail2ban installed at ~23:30 CEST → 4 IPs immediately banned → Post-reboot: 10 IPs banned |
| **Remediation** | fail2ban installed (maxretry=3, bantime=3600), SSH PasswordAuthentication disabled, PermitRootLogin disabled, UFW verified active |
| **Post-remediation** | Service survived kernel upgrade reboot (6.8.0-106→134). 6 new attack attempts blocked within 2 minutes of reboot. |

### FINDING-002: SSH Configuration Drift After Reboot

| Field | Detail |
|-------|--------|
| **Severity** | MEDIUM |
| **Category** | Configuration drift |
| **Confidence** | 7 (suppressed — operational, not code) |
| **Description** | Ubuntu cloud-init (`50-cloud-init.conf`) reverted `PasswordAuthentication` to `yes` after kernel upgrade reboot. Fixed by editing cloud-init config. Cloud-init may revert again on future provisioning events. |

## Seal MCP Recommendations

Based on the operational findings, the following Seal actions are recommended:

1. **Create risk event** for "Active SSH brute-force campaign against production VPS" (HIGH severity, Cybersecurity category) documenting the 10 attacker IPs and remediation timeline.

2. **Update residual likelihood** on risk `cdd345cc` (Credential theft via phishing) — reduce residual likelihood from 3 to 2 due to elimination of password-based SSH access as an attack vector.

3. **Add compensating control note** to treatment `07a08f82` (Deploy FIDO2 MFA) documenting SSH hardening as a compensating control while MFA deployment is in progress.
