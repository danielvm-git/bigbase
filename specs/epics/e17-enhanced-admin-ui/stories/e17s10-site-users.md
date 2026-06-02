---
id: e17s10
title: Site detail and Users enhancements
status: in_progress
legacy_slice: "017-J"
tasks:
  - desc: Write SiteDetailPage.test.tsx — render, status timeline, deployments, redeploy
    verify: "cd ui && npx vitest run src/pages/SiteDetailPage.test.tsx"
  - desc: Write UsersPage.test.tsx — user list, delete with confirmation, error state
    verify: "cd ui && npx vitest run src/pages/UsersPage.test.tsx"
  - desc: Full verify — both page tests pass + UI builds
    verify: "cd ui && npm test -- SiteDetailPage UsersPage -- --coverage && npm run build"

context: |
  SiteDetailPage.tsx (197 lines) exists with: status timeline (pending→building
  →deploying→running), deployment history table, redeploy button with error
  handling, branch display, auto-polling, preview mode banner. UsersPage.tsx (90
  lines) exists with: user list table, delete with confirmation dialog, error
  state, loading state. Neither has test files.
---

## Implementation Steps

**Context**: Both pages are functional and in use. The tests should validate the
observable behaviors through mocked API calls, following the
`FunctionLogsPage.test.tsx` pattern.

### Phase A: SiteDetailPage tests

#### Step 1: Test file scaffolding

Create `ui/src/pages/SiteDetailPage.test.tsx` following the existing test pattern:
- Vitest + React Testing Library + `MemoryRouter` with route params
- Mock `getSite` and `getSiteDeployments` from `../lib/sitesData`
- Mock `globalThis.fetch` for redeploy endpoint

→ verify: `cd ui && grep -q 'describe.*SiteDetailPage' src/pages/SiteDetailPage.test.tsx`

#### Step 2: Basic render — site name, branch, repo info

Mock a site with name "My Site", branch "main", repo "org/mysite". Assert:
- Page header shows "My Site"
- Branch info shows "main"
- Repo info shows "org/mysite"
- "← All sites" link renders

→ verify: `cd ui && npx vitest run src/pages/SiteDetailPage.test.tsx -t "render"`

#### Step 3: Status timeline

StatusTimeline is rendered inline. Mock a site with status "building". Assert:
- 4 timeline steps: pending, building, deploying, running
- "building" step is bold/active (font-weight 600)
- Previous step (pending) has brand-colored dot
- Future steps (deploying, running) have gray dots

Also test "failed" status: failed step shows error color.

→ verify: `cd ui && npx vitest run src/pages/SiteDetailPage.test.tsx -t "timeline"`

#### Step 4: Deployment history table

Mock 2 deployments. Assert:
- Table shows both deployments with commit hashes
- Status badges render with correct variant
- Duration shown for completed deployments
- "No deployments" when array is empty

→ verify: `cd ui && npx vitest run src/pages/SiteDetailPage.test.tsx -t "deployments"`

#### Step 5: Redeploy button

Click "Redeploy". Mock successful POST. Assert:
- Refresh fetches deployments again
- Error state: mock 500 → assert "Redeploy failed" message
- Network error: mock rejection → assert error message

→ verify: `cd ui && npx vitest run src/pages/SiteDetailPage.test.tsx -t "redeploy"`

#### Step 6: Auto-polling and preview mode

- Assert auto-polling starts when there's a "building" deployment
- Assert `setInterval` is cleared on unmount
- Assert preview banner shows when `VITE_SITES_PREVIEW=1` or `?preview=1`

→ verify: `cd ui && npx vitest run src/pages/SiteDetailPage.test.tsx -t "polling"`

### Phase B: UsersPage tests

#### Step 7: Test file scaffolding

Create `ui/src/pages/UsersPage.test.tsx`.

→ verify: `cd ui && grep -q 'describe.*UsersPage' src/pages/UsersPage.test.tsx`

#### Step 8: User list render

Mock `/api/auth/users` returning 3 users with varying roles (owner, admin,
member). Assert:
- All 3 emails appear
- Role badges render (owner→accent, admin→success, member→neutral)
- Verified status shows check or cross indicator

→ verify: `cd ui && npx vitest run src/pages/UsersPage.test.tsx -t "list"`

#### Step 9: Delete user with confirmation

Mock `window.confirm` → true. Click delete for a user. Mock DELETE endpoint.
Assert:
- User row disappears
- Confirm was called with user ID in message

Test cancel: `window.confirm` → false. Assert user remains in list.

→ verify: `cd ui && npx vitest run src/pages/UsersPage.test.tsx -t "delete"`

#### Step 10: Error and edge cases

- Mock fetch failure → assert "Failed to load users"
- Mock delete failure (500) → assert error message, user still in list
- Empty user list → assert empty state or no rows
- Loading state → assert "Loading users..."

→ verify: `cd ui && npx vitest run src/pages/UsersPage.test.tsx -t "error"`

### Phase C: Full verify

#### Step 11: End-to-end verify

→ verify: `cd ui && npm test -- SiteDetailPage UsersPage -- --coverage && npm run build`

## Verification Script (Manual)

1. `cd ui && npx vitest run src/pages/SiteDetailPage.test.tsx` — all pass
2. `cd ui && npx vitest run src/pages/UsersPage.test.tsx` — all pass
3. `cd ui && npm test -- SiteDetailPage UsersPage -- --coverage` — green
4. `cd ui && npm run build` — builds cleanly

## Out of scope
- 5-tab navigation in SiteDetail (tabs already implemented; testing tab content beyond Overview is deferred)
- Users invite modal (not yet built in UsersPage)
- User metric tiles (not yet built in UsersPage)
- E2E tests

## Risks
- SiteDetailPage depends on `lib/sitesData` — mock the module or mock fetch
- `useParams` requires MemoryRouter with initialEntries including `:siteId`
- Auto-polling tests need `vi.useFakeTimers` and cleanup
- `window.confirm` must be restored after each test
