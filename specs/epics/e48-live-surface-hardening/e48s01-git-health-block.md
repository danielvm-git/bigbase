# e48s01: Block .git exposure and harden /health endpoint

## 1. Story ID
e48s01

## 2. Epic
e48 — Security: Live Surface Hardening

## 3. Status
planned

## 4. BCPs
2

## 5. Summary
Block `/.git/*` path access in the proxy middleware and add authentication gating to the `/health` endpoint that currently exposes the full 17-component architecture map publicly.

## 6. Acceptance Criteria (Gherkin)
```gherkin
Feature: Block sensitive paths and harden health endpoint

  Scenario: .git paths return 404
    Given BigBase is running
    When I request GET /.git/config
    Then the response status is 404

  Scenario: .git paths blocked recursively
    When I request GET /.git/HEAD
    Then the response status is 404

  Scenario: /health gated behind token
    Given BigBase is configured with HEALTH_TOKEN=secret123
    When I request GET /health without Authorization header
    Then the response status is 401

  Scenario: /health available with correct token
    Given BigBase is configured with HEALTH_TOKEN=secret123
    When I request GET /health with Authorization: Bearer secret123
    Then the response status is 200
    And the response includes component status data

  Scenario: Other paths unaffected
    When I request GET /
    Then the response status is 200
```

## 7. Definition of Done
- [ ] `/.git/*` paths return 404 from proxy
- [ ] `/health` requires Bearer token when HEALTH_TOKEN env var is set
- [ ] `/health` returns 401 without valid token
- [ ] `go test ./components/proxy/ -run TestGitPathBlocked -v` passes
- [ ] `go test ./components/proxy/ -run TestHealthAuth -v` passes
- [ ] `npm run preflight` passes
