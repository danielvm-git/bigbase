# e78s02: Dashboard, Layout & Settings

## 1. Title
Dashboard, Layout & Settings

## 2. Description
Browser E2E tests for the main dashboard, IA shell (sidebar, theme picker), onboarding DX, and Org usage charts.

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: Dashboard, Layout & Settings

  Scenario: Responsive Sidebar & Theming (e17, e51)
    Given a logged in user
    When they toggle the sidebar on mobile (375px) vs desktop (1024px)
    And they select a new accent theme from the 12-color picker
    Then the layout responds correctly and CSS variables update

  Scenario: DX Onboarding & Tutorial (e22, e51)
    Given a new user on the Dashboard
    When they interact with the onboarding checklist and interactive tutorial
    Then progress updates and sample-app Deploy buttons are functional

  Scenario: Dashboard Metrics & Org Usage (e05, e17, e23)
    Given a user on the Dashboard
    Then they see the MetricCard, RequestChart, and ComponentHealthGrid
    When they navigate to Org usage
    Then they see usage tracking charts rendered

  Scenario: Settings & Accounts (e17)
    Given a user in Settings
    Then they can view Account, Workspace, and Billing tab stubs
```
