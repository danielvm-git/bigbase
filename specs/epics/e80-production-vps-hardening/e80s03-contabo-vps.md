# Contabo VPS operations — snapshots and API monitoring

> **Epic:** e80 — Production VPS Security Hardening — Contabo
> **Layer:** VPS

## User Story

**As a** platform operator, **I want** to automate Contabo VPS snapshots and monitor instance health via the Contabo API, **so that** I can recover from catastrophic VPS failure within minutes and detect infrastructure-level issues before they affect BigBase.

## Acceptance Criteria (Gherkin)

```gherkin
Scenario: cntb CLI is installed and authenticated
  Given the cntb CLI is installed on the VPS
  When I run "cntb get instances"
  Then the BigBase VPS instance is listed with status "running"

Scenario: Monthly snapshot is created automatically
  Given a crontab entry triggers the Contabo snapshot API on the 1st of each month
  When the script runs
  Then a new snapshot is created for the BigBase instance
  And the snapshot appears in the Contabo Customer Control Panel

Scenario: Instance health is queryable
  Given the Contabo API credentials are configured
  When I query GET /v1/compute/instances/{id}
  Then the instance status is "running"
  And CPU, RAM, and disk metrics are returned

Scenario: Troubleshooting checks run on demand
  Given the Contabo API is reachable
  When I POST a check against the VPS instance
  Then a check result is returned with status "successful" or actionable warning
```

## Scope

| In scope | Out of scope |
|----------|-------------|
| cntb CLI installation and authentication | Custom monitoring dashboard |
| Monthly snapshot via crontab + API | Real-time snapshot on every deploy |
| Instance status query | Auto-scaling, multi-region |
| Troubleshooting check on demand | Auto-remediation on check failure |
