# e78s04: Data, SQL & Storage

## 1. Title
Data, SQL & Storage

## 2. Description
Browser E2E tests for Data Studio (filter/sort, schema editor), SQL editor, and Storage grid/preview functionality.

## 17. Acceptance Criteria (Gherkin)

```gherkin
Feature: Data and Storage

  Scenario: Data Studio Schema & Filtering (e05, e17, e30)
    Given a user in Data Studio
    When they toggle between Data and Schema views
    And they Add/Edit/Delete a column
    Then they can filter/sort the collection list and click "Query this"

  Scenario: SQL Editor (e05)
    Given a user on the SQL Editor page
    When they input a SELECT query and click Run
    Then the dataset columns and rows display in the grid

  Scenario: Storage UI Features (e06, e17)
    Given a user in Storage
    When they upload a file via drag-drop
    Then they can toggle grid/list view, open the preview modal, and delete the file
```
