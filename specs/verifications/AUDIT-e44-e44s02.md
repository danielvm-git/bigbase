# Audit Report — e44s02 Admin UI Rollback

**Epic:** e44 — Deploy: One-Click Rollback
**Story:** e44s02 — Admin UI rollback button + rollback history timeline
**Commit:** b97fd9f17fe4f8f9feb34ea9d32cb2c43c5c7c8a
**Audit date:** 2026-06-26
**Gate mode:** --gate

## Results

| Section | Status |
|---------|--------|
| Supply Chain & Security | PASS |
| Provenance & Metadata | PASS |
| Law of Demeter | PASS |
| CONVENTIONS.md Compliance | FAIL |
| Scope | PASS |
| Boy Scout Rule | PASS |
| Types and Safety | PASS |
| Test Coverage | FAIL |
| SOLID and Heuristics | PASS |
| Code Style | PASS |
| Agent Readability | PASS |

## Detailed Checklist

### Supply Chain & Security
✓ No new npm dependencies added (only type definitions used)
✓ No secrets in diff (no API keys, auth tokens, .env values)
✓ Fetch calls use existing auth framework (inherited from main router)
✓ Input validation: rollback button only shows for `status === 'running'` with previous deployments
✓ Dialog prevents accidental clicks (confirmation required, stopPropagation)

### Provenance & Metadata
✓ Epic spec already has `type:` and `context:` metadata
✓ No new spec artefacts needed

### Law of Demeter
✓ No method chains through unrelated objects
✓ Collaborators (fetch, navigate) are immediate neighbors
✓ `handleRollbackConfirm` calls `rollbackDeployment()` then refreshes UI

### CONVENTIONS.md Compliance
✗ **Pre-existing linting violations in SiteDetailPage.tsx** (lines 142, 326):
  - ESLint rule `react-hooks/set-state-in-effect` flagged
  - Both violations are in existing code (`loadManifest()` and `load()` functions)
  - The `load()` function was modified to also fetch rollback events (line 321-322), inheriting this pattern
  - **Issue:** Should use `useCallback` with proper dependencies or refactor to avoid setState in effect body
  - **Status:** Pre-existing violations, not introduced by rollback feature, but now wider scope due to load() modification

### Scope
✓ Changes limited to e44s02: UI rollback button, confirmation dialog, rollback history, API helpers
✓ No speculative features added
✓ Files touched: SiteDetailPage.tsx (rollback button + dialog + history), sitesData.ts (API helpers), types/sites.ts (RollbackEvent), deployment/deploy.go (route endpoints)

### Boy Scout Rule
✓ No dead code left behind
✓ No commented-out code blocks
✓ All changes are additions, no cleanup beyond scope

### Types and Safety
✓ RollbackEvent type defined with explicit fields
✓ Deployment type extended with `rolled_back` and `replaced` statuses
✓ TypeScript strict mode: no `any` types, all function returns typed
✓ `canRollback()` returns boolean, explicit type
✓ All state setters properly typed (`Deployment | null`, `boolean`, `string | null`, `RollbackEvent[]`)

### Test Coverage
✗ **FAIL: No tests for new UI functions**
  - `rollbackDeployment()` in sitesData.ts: new exported function, no test
  - `getRollbackEvents()` in sitesData.ts: new exported function, no test
  - `handleRollbackClick()`: tested implicitly via manual UI testing, but no unit test
  - `handleRollbackConfirm()`: tested implicitly via manual UI testing, but no unit test
  - `canRollback()`: tested implicitly via manual UI testing, but no unit test
  - **Note:** Backend rollback tests (e44s01) are comprehensive; UI tests should verify:
    * Rollback button only appears for running deployments with previous versions
    * Confirmation dialog shows deployment details correctly
    * Rollback API call succeeds and UI updates
    * Rollback API failure displays error message
    * Rollback history timeline renders correctly

### SOLID and Heuristics
✓ Single Responsibility: rollback button triggers rollback, dialog confirms, history displays
✓ Open/Closed: extended via new functions, not modifying core SiteDetailPage logic
✓ Dependency Inversion: rollbackDeployment() and getRollbackEvents() abstract fetch/parse
✓ No unmaintainable patterns
✓ Comments explain intent (lines 424, 602-604)

### Code Style
✓ Functions concise:
  - `handleRollbackClick`: 3 lines
  - `handleRollbackConfirm`: ~15 lines
  - `canRollback`: 4 lines
✓ Early returns used (line 406)
✓ Max indentation ~3 levels (acceptable for React component)
✓ Comments explain WHY, not WHAT

### Agent Readability
✓ Unique, grep-able names (`rollbackTarget`, `rollbacking`, `RollbackEvent`)
✓ Explicit types on all public APIs
✓ Deep nesting avoided via early returns
✓ Clear intent in function names

## Verdict

**CONDITIONAL FAIL**

### Why This Fails

1. **Test Coverage (CRITICAL):** New exported functions (`rollbackDeployment`, `getRollbackEvents`) have zero test coverage. Per project convention, every new function must have at least one test.
2. **Linting Violations (INHERITED):** Pre-existing ESLint errors in SiteDetailPage.tsx now affect a larger codebase scope due to the `load()` modification. While not introduced by this story, the violations should be fixed before merge.

### Remediation

To pass audit:

1. **Add tests to sitesData.test.ts:**
   ```typescript
   describe('rollbackDeployment', () => {
     // Test happy path: rollback succeeds, returns event
     // Test error path: API returns 409/500, returns error
     // Test preview mode: returns mock event
   })
   
   describe('getRollbackEvents', () => {
     // Test happy path: returns events array
     // Test empty array: no rollback history
     // Test preview mode: returns mock events
   })
   ```

2. **Add tests to SiteDetailPage.test.tsx:**
   ```typescript
   describe('Rollback UI', () => {
     // Test rollback button visibility (running + previous deployment)
     // Test rollback button hidden (non-running deployment)
     // Test confirmation dialog appears on button click
     // Test dialog cancel closes without action
     // Test dialog confirm calls rollbackDeployment()
     // Test rollback success reloads deployments
     // Test rollback error displays error message
     // Test rollback history timeline renders
   })
   ```

3. **Optional but recommended: Fix pre-existing ESLint violations:**
   - Refactor the `useCallback` pattern to avoid setState in effect body, or
   - Use a separate effect for the initial load vs polling interval

## Notes

- Backend implementation (e44s01) is solid with good test coverage
- UI implementation is functionally correct and types are sound
- The missing tests are the only blocker — functionality itself passes all spot checks
- Type checking passes (no TypeScript errors)
- UI builds clean

---

**Status:** FAIL — blocked by missing test coverage. Ready to proceed after tests are added.
