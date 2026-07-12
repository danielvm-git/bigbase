# Ubuntu OS security hardening

> **Epic:** e80 — Production VPS Security Hardening — Contabo
> **Layer:** OS

## User Story

**As a** platform operator, **I want** the Ubuntu VPS at Contabo to be hardened against common attack vectors, **so that** brute-force SSH attempts, unpatched CVEs, and resource exhaustion cannot compromise the production BigBase instance.

## Acceptance Criteria (Gherkin)

```gherkin
Scenario: Firewall allows only required ports
  Given the VPS is running Ubuntu
  When I run "sudo ufw status verbose"
  Then the firewall is active
  And only ports 22, 80, 443 are allowed inbound
  And all outbound traffic is allowed

Scenario: fail2ban blocks repeated SSH failures
  Given fail2ban is configured with maxretry=3 and bantime=3600
  When an IP fails SSH authentication 4 times within 10 minutes
  Then that IP is banned for at least 3600 seconds

Scenario: fail2ban blocks repeated BigBase auth failures
  Given the custom bigbase-auth filter is active
  When an IP receives 11 HTTP 401 responses on POST /api/auth/login within 5 minutes
  Then that IP is banned for 1800 seconds

Scenario: SSH is hardened against password and root attacks
  Given /etc/ssh/sshd_config is hardened
  When I run "sudo sshd -T"
  Then PermitRootLogin is "no"
  And PasswordAuthentication is "no"
  And PubkeyAuthentication is "yes"

Scenario: Security patches are applied automatically
  Given unattended-upgrades is installed and active
  When security updates are available in the Ubuntu repository
  Then they are applied within 24 hours
  And the VPS reboots automatically at 4AM if a kernel update requires it

Scenario: BigBase systemd service runs with minimal privileges
  Given /etc/systemd/system/bigbase.service is hardened
  When I run "systemctl show bigbase"
  Then NoNewPrivileges is "yes"
  And ProtectSystem is "strict"
  And ProtectHome is "yes"
  And ReadWritePaths includes only /opt/bigbase/data and /opt/bigbase/logs

Scenario: Service survives reboot
  Given systemctl is-enabled bigbase returns "enabled"
  When the VPS reboots
  Then BigBase starts automatically within 30 seconds
```

## Scope

| In scope | Out of scope |
|----------|-------------|
| UFW firewall rules | Cloud firewall (Contabo panel) |
| fail2ban SSH + BigBase auth | WAF / DDoS protection |
| SSH hardening (no root, no password) | SSH key rotation |
| unattended-upgrades for security patches | Non-security package updates |
| systemd service hardening | Container migration |
| Auto-reboot for kernel updates | Livepatch (Canonical subscription) |
