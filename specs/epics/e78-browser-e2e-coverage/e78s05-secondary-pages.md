# e78s05: Secondary Pages

## 1. Title
Secondary Pages

## 2. Description
Browser E2E tests for Functions, Messaging, Users, and Settings pages.

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: Secondary Pages

  Scenario: UI Password Reset Form (P1)
    Given a user on the Settings page
    When they input a new password and submit
    Then they see a success toast
    And can login with the new password
```
