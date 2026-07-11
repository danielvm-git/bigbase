# e78s03: Sites & Deploy UI

## 1. Title
Sites & Deploy UI

## 2. Description
Browser E2E tests for the sites listing, deploy token modal, environment variables editor, and site detail views.

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: Sites and Deploy

  Scenario: Deploy Token UI lifecycle (P0)
    Given a user on a site detail page
    When they open the deploy keys tab
    And create a new key
    Then the key value is displayed once
    And the new key appears in the list
    When they click revoke
    Then the key is removed from the list

  Scenario: UI Env Vars Editor (P1)
    Given a user on the site env vars tab
    When they add a new key and value
    And save
    Then the variables are saved and displayed
```
