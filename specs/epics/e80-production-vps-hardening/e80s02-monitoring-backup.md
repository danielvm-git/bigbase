# BigBase monitoring alerts and backup automation

> **Epic:** e80 — Production VPS Security Hardening — Contabo
> **Layer:** BigBase

## User Story

**As a** platform operator, **I want** BigBase to alert on resource thresholds and the SQLite database to be backed up daily, **so that** I know about disk/CPU/RAM saturation before it causes downtime and can recover from data loss.

## Acceptance Criteria (Gherkin)

```gherkin
Scenario: Alert fires when disk usage exceeds 80%
  Given a BigBase alert named "VPS Disk > 80%" is configured with threshold=80 and operator=gt
  When the VPS disk usage surpasses 80% for more than 300 seconds
  Then an incident is created in monitoring_alert_incidents
  And the incident is visible at GET /api/monitoring/incidents?status=open

Scenario: Health check cron script runs every 5 minutes
  Given the healthcheck.sh script is installed at /opt/bigbase/scripts/
  When the cron timer fires
  Then the script checks disk, RAM, and BigBase service status
  And logs warnings to /var/log/bigbase/health.log if any threshold is breached
  And automatically restarts BigBase if the service is not active

Scenario: Daily SQLite backup runs at 2 AM
  Given a crontab entry copies bigbase.db to /backup/ with a date-stamped filename
  When the clock reaches 2:00 AM
  Then the backup file exists at /backup/bigbase-YYYYMMDD.db
  And backups older than 90 days are deleted

Scenario: Alerts are queryable via API
  Given 3 alerts are configured (disk, CPU, RAM)
  When I call GET /api/monitoring/alerts
  Then the response contains exactly 3 alert definitions
```

## Scope

| In scope | Out of scope |
|----------|-------------|
| BigBase alert creation via API | External alerting (PagerDuty, Slack) |
| Health check shell script on VPS | Health check as Go code inside BigBase |
| Daily SQLite backup via crontab | Off-site backup replication |
| 90-day backup rotation | Encrypted backups |
