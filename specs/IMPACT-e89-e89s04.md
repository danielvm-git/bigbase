type: impact-assessment
context: domain
epic: e89
story: e89s04
mode: lightweight

# Impact Assessment: e89s04

## Target

Expose policy-separated Project secret REST routes with masked metadata, explicit value
reads, rate limits, and value-free audit events.

## Module purpose, callers, contracts

- Purpose: REST adapters translate authenticated HTTP requests into SecretManager
  actions; Auth supplies organization and role context; audit/monitoring records safe
  actor and scope metadata.
- Callers: Admin UI, automation clients, contract tests, and the s05/s07 adapter wave.
- Contracts: `/api/projects/{project}/environments/{env}/secrets` collection routes;
  separate `/value` read route; JSON 401/403/non-disclosing 404 shapes; one shared
  policy helper; metadata/masked mutation responses; no plaintext/ciphertext audit data;
  parameterized SQL and bounded mutations.

## Impact and risks

Critical. Authorization paths can diverge across list, version, mutation, and value
read actions. The REST branch must consume, not alter, the s03 storage contract. Route
mounting and SecretManager injection are coordinator-owned `main.go` work.

## Coverage

Scenarios: SC-e89s04-P0-01 through SC-e89s04-P0-04 and SC-e89s04-P1-05/P1-06.

## Recommended action

Start in parallel with s06 only after s03 contract freeze. Complete policy matrix,
contract, audit, rate-limit, and SQL-safety checks before opening the UI/MCP wave.
