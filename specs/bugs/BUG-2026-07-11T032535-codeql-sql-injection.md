# Database query built from user-controlled sources (8 instances)

**Source:** GHS Code Scanning (CodeQL)
**Severity:** MAJOR
**CWE:** CWE-89 (SQL Injection)
**GitHub Alerts:** #8, #9, #10, #11, #12, #13, #14, #15

## Description
CodeQL detected 8 instances where SQL queries are built by concatenating user-controlled input. These are in the API component's endpoint handlers where filter/sort parameters are interpolated into SQL.

## Recommendation
Replace string concatenation with parameterized queries. Use query builders or prepared statements with bound parameters for all user-controlled WHERE, ORDER BY, and LIMIT clauses.

## Status
triage

## Source
seal.github_code_scanning

## Discovered
2026-07-11
