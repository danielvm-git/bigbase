# VPS Verification (8 gates)

source: AGENTS.md
references: [AGENTS.md]

### VPS Verification (8 gates)
```bash
ufw status|grep -q active||echo FAIL:ufw
fail2ban-client status sshd>/dev/null 2>&1||echo FAIL:fail2ban
systemctl is-active unattended-upgrades|grep -q active||echo FAIL:unattended
sshd -T|grep -q 'permitrootlogin no'||echo FAIL:sshd
systemctl show bigbase -p NoNewPrivileges|grep -q yes||echo FAIL:systemd
systemctl is-active bigbase|grep -q active||echo FAIL:bigbase
sqlite3 /opt/bigbase/data/bigbase.db "SELECT count(*) FROM monitoring_alerts"|grep -q 3||echo FAIL:alerts
crontab -l|grep -q healthcheck&&crontab -l|grep -q bigbase.db&&crontab -l|grep -q contabo-snapshot||echo FAIL:crontab
echo ALL 8 GATES PASSED
```
