# e48s03: DAST baseline scan and scheduled automation
## Story ID: e48s03 | Epic: e48 | BCPs: 2 | Status: planned
Covers: OWASP ZAP baseline scan script, Mozilla Observatory check, weekly cron.
## Acceptance Criteria
```gherkin
Scenario: ZAP baseline scan runs against staging
  When scripts/zap-baseline.sh is executed
  Then a zap-report.html is generated with no HIGH alerts
```
