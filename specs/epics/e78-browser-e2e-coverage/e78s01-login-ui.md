# e78s01: Login & Authentication UI

## 1. Title
Login & Authentication UI

## 2. Description
Browser E2E tests for the login, registration, and authentication flows, ensuring correct routing and session setup.

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: Login and Registration

  Scenario: Successful Login E2E (P0)
    Given a registered user
    When they enter valid credentials
    And submit the login form
    Then a session cookie is set
    And they are transitioned to the dashboard

  Scenario: Registration form routing (P1)
    Given an anonymous user
    When they fill out the registration form
    And submit it
    Then they are automatically logged in and routed to onboarding
```
