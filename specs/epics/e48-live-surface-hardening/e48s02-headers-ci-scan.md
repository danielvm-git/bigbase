# e48s02: Add missing security headers and CI scanning
## Story ID: e48s02 | Epic: e48 | BCPs: 2 | Status: planned
Covers: Permissions-Policy, Cache-Control headers + gosec/govulncheck/gitleaks CI integration.
## Acceptance Criteria
```gherkin
Scenario: Permissions-Policy header present
  When I request GET /
  Then response includes Permissions-Policy header
Scenario: gosec runs in CI preflight
  When CI runs npm run preflight
  Then gosec scans ./... with zero HIGH issues
```
