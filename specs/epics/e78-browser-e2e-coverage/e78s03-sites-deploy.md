# e78s03: Sites, Deploy & CI/CD

## 1. Title
Sites, Deploy & CI/CD

## 2. Description
Browser E2E tests covering Site deployment lifecycle (GitHub connect, rollback, drain), Site Detail tabs (Manifest, Env Vars, Cache, Logs), and CI/CD pipelines/DAG.

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: Sites and Deploy

  Scenario: Create Site via GitHub (e13, e15)
    Given a user on /deploy/new
    When they complete the GitHub connect flow
    And trigger a one-click deploy with app-type detection
    Then the site begins deploying

  Scenario: Site Detail Tabs & Actions (e36, e40, e41, e42, e44, e45)
    Given a user on a Site Detail page
    When they navigate through Manifest, Env Vars, and Cache tabs
    Then they can inline-edit bigbase.yaml, bulk import .env, and clear cache
    When they trigger a Rollback or Zero-downtime drain
    Then the status timeline reflects the pulsing drain state

  Scenario: CI/CD Pipeline & Streaming Logs (e09, e17, e26, e27, e39)
    Given a deploying site
    Then the TerminalLogViewer shows streaming build logs (WebSocket)
    And they can filter Request logs
    When they view the CI/CD pipeline history
    Then the workflow DAG and StreamLog components render correctly
```
