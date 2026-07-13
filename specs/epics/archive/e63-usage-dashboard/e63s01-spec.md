### Story e63s01: Usage Metrics Aggregation Backend — Implementation Steps

**type:** feat
**context:** domain
**Context**: This story adds a new endpoint to the `monitoring` component to aggregate resource usage for an organization (database size, storage bytes, and active deployments). This provides the data foundation required for the new Usage Dashboard (e63s02) in the admin UI.

**Zoom-Out Check**:
- **Purpose**: Aggregate project-level resource utilization metrics (db, storage, sites) into a single API response.
- **Callers**: Admin UI (Usage Dashboard page).
- **Contracts**: Must return a JSON object with `database_size_bytes`, `storage_size_bytes`, `total_users`, and `active_deployments`.

## Steps

1. Add `ResourceUsage` struct and `GetOrgResourceUsage(orgID)` query method to `components/monitoring/org_usage.go` that aggregates counts from users, storage_objects, and deployments. → verify: `go test ./components/monitoring -run TestGetOrgResourceUsage`
2. Add `handleOrgResources` to expose `GET /api/orgs/{id}/resources` and register it in the monitoring HTTP router. → verify: `go test ./components/monitoring -run TestHandleOrgResources`
3. Wire the new endpoint to the `api` component router if required (or if monitoring mounts its own subrouter, ensure it is accessible). → verify: `go build ./...`

## Verification Script (Step-by-Step)

1. Start the BigBase server with `go run . serve`
2. Make a curl request to `GET /api/orgs/1/resources` using a valid admin token.
3. Verify the JSON response contains `database_size_bytes`, `storage_size_bytes`, `total_users`, and `active_deployments` fields with numeric values.

## Out of scope

- Historical resource tracking over time (only current snapshot is provided for these resources).
- Billing or quota enforcement (purely informational).

## Risks

- Cross-component querying: `monitoring` component might need to query tables owned by `deploy`, `auth`, or `storage`. Ensure queries use the standard shared DB abstraction and don't violate the ECC pattern (which allows DB reads across components if no internal business logic is duplicated).
