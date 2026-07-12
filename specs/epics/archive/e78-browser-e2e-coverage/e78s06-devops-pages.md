# e78s06: DevOps, Monitoring & Forge

## 1. Title
DevOps, Monitoring & Forge

## 2. Description
Browser E2E tests for Monitoring (hardware, alerts), Realtime Inspector, Events Canvas, Git Repos, and Forge tracker.

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: DevOps, Monitoring & Forge

  Scenario: Hardware Monitoring & Alerts (e14, e25)
    Given a user on the Monitoring page
    Then they see CPU/memory charts, disk gauge, and network sparkline
    And they can configure Alert rules

  Scenario: Realtime & Events (e11, e17, e22)
    Given a user on Realtime page
    Then they can view connections/subscriptions
    When they navigate to Events
    Then the Event Bus visualizer canvas renders

  Scenario: Forge Kanban & Wiki (e08)
    Given a user on the Forge page
    Then they can view the Issue tracker, Kanban board, and Wiki UI

  Scenario: Git Repo Management (e07)
    Given a user on the Repos page
    Then they can view the Repo management screen
```
