# e78s01: Login & Authentication UI

## 1. Title
Login & Authentication UI

## 2. Description
Browser E2E tests for the login, registration, password reset, and authentication framework preview flows.

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: Login and Authentication UI

  Scenario: Successful Login and Registration (e04)
    Given an anonymous user
    When they fill out the registration form
    Then they are routed to onboarding
    When they log out and use the Login form
    Then a session cookie is set and they enter the dashboard

  Scenario: Password Reset UI (e17)
    Given an anonymous user on the login page
    When they click forgot password and submit email
    Then they receive a reset link confirmation toast

  Scenario: Auth Framework Component Preview (e34)
    Given a developer on /admin/auth
    When they click through React, Vue, and Svelte framework tabs
    Then the <SignIn>, <SignUp>, and <UserButton> preview components render
```
