# e77s01: API Surface E2E Tests

## 1. Title
API Surface E2E Tests

## 2. Description
API E2E request-level tests covering all 17 components to ensure org isolation, correct CRUD behaviors, and data security.

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: API Surface

  Scenario: Org Isolation (P0)
    Given two separate tenants A and B exist
    When Tenant A attempts to read Tenant B's sites
    Then the system returns 404 or 403
    
  Scenario: Path Traversal Guard (P2)
    Given a file exists in storage
    When a download request contains `../` patterns
    Then the request is rejected as invalid path
```
