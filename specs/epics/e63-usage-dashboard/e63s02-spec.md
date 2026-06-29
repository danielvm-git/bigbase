### Story e63s02: Usage Dashboard UI — Implementation Steps

**type:** feat
**context:** domain
**Context**: This story implements the Usage Dashboard page in the Admin UI. It fetches both the 30-day API usage history (`GET /api/orgs/{id}/usage`) and the current resource snapshot (`GET /api/orgs/{id}/resources`) to display usage trends and current limits for a project/organization.

**Zoom-Out Check**:
- **Purpose**: Provide a visual dashboard for org admins to understand their resource utilization and API traffic.
- **Callers**: End-users via the browser navigating to the Admin UI.
- **Contracts**: Adheres to the established Design System (e51) and communicates with the `monitoring` backend endpoints.

## Packages (Slopcheck)
- `recharts` [OK] - if charting library is needed for the 30-day usage graph, though we may use native CSS.

## Steps

1. Create a `UsageDashboard.tsx` page component in `ui/src/pages/` that uses the standard page layout. → verify: `cd ui && npm run build`
2. Implement data fetching hooks for `GET /api/orgs/{id}/usage` and `GET /api/orgs/{id}/resources`. → verify: `cd ui && npm run tsc`
3. Build the UI layout with stat cards for Database Size, Storage, Active Sites, and API Requests. → verify: `cd ui && npm run build`
4. Register the `/orgs/:id/usage` route in the application router (`ui/src/App.tsx` or `ui/src/routes.tsx`). → verify: `cd ui && npm run build`

## Verification Script (Step-by-Step)

1. Start the BigBase server with `go run . serve` and open the Admin UI in a browser.
2. Navigate to an Organization's dashboard and click the "Usage" tab or link.
3. Verify that the Usage Dashboard renders without errors.
4. Verify that the stat cards show the current resource limits and the 30-day request usage trend is visible.

## Out of scope

- Setting or enforcing billing quotas from the UI.
- Real-time live-updating of metrics via WebSockets (page refresh is sufficient for v1).

## Risks

- Ensuring the UI gracefully handles loading states and errors if the backend aggregation takes longer to compute on large datasets.
