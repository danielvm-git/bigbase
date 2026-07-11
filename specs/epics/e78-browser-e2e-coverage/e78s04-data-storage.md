# e78s04: Data, SQL & Storage

## 1. Title
Data, SQL & Storage

## 2. Description
Browser E2E tests for the Data Studio, SQL query editor, and Storage browser.

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: Data and SQL

  Scenario: SQL Editor Execution (P0)
    Given a user on the SQL Editor page
    When they input a SELECT query
    And click Run
    Then the dataset columns and rows are displayed in the grid

  Scenario: Data Studio Navigation Link (P2)
    Given a user viewing a table schema in Data Studio
    When they click "Query this"
    Then they are forwarded to the SQL editor with the table pre-selected
```
