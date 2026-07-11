# e78s02: Dashboard & Navigation

## 1. Title
Dashboard & Navigation

## 2. Description
Browser E2E tests for the main dashboard interface and sidebar navigation routing.

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: Dashboard Navigation

  Scenario: View dashboard widgets (P1)
    Given a logged in user
    When they navigate to the dashboard
    Then the system stats and recent projects widgets should render successfully

  Scenario: Sidebar navigation routing (P1)
    Given a user on the dashboard
    When they click the "Sites" link in the sidebar
    Then they are routed to the Sites list page
```
