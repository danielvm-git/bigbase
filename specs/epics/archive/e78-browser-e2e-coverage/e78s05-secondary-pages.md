# e78s05: Functions, Messaging & Users

## 1. Title
Functions, Messaging & Users

## 2. Description
Browser E2E tests for Functions (CRUD, run, logs), Messaging (template editor), and Users (delete confirmation).

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: Functions, Messaging and Users

  Scenario: Functions CRUD & Logs (e10, e17)
    Given a user on the Functions page
    When they view the function card grid
    And click a function to see code/triggers/variables/logs tabs
    Then they can run the function and view execution logs

  Scenario: Messaging Template Editor (e12, e17)
    Given a user on the Messaging page
    When they create a new message and use the template editor/preview
    Then they can send the message and verify it in the outbound log

  Scenario: Users Management (e17)
    Given a user on the Users page
    When they click delete on a user
    Then a delete confirmation modal appears
```
