# e76s01: Token Lifecycle E2E Tests

## 1. Title
Token Lifecycle E2E Tests

## 2. Description
API E2E request-level tests covering full session lifecycle (JWT, refresh, anon, MCP) and org key lifecycle to prevent authentication regressions.

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: Token Lifecycle

  Scenario: Full session lifecycle (P0)
    Given a new user registers
    When they log in and receive tokens
    And they refresh their token
    And they attempt to use the old rotated token
    Then the old token should be rejected (401)
    When they trigger logout-all
    Then all family tokens should be invalidated

  Scenario: Org API key lifecycle (P0)
    Given an organization exists
    When an admin generates a `bb_*` key
    And authenticates with the key
    Then access is granted
    When the key is revoked
    And they authenticate again
    Then access is rejected with 401
```
