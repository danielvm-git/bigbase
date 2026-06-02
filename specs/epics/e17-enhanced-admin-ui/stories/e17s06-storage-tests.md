---
id: e17s06
title: Storage page test coverage
status: in_progress
legacy_slice: "017-F"
tasks:
  - desc: Create StoragePage.test.tsx — render, empty state, upload error
    verify: "cd ui && npx vitest run src/pages/StoragePage.test.tsx"
  - desc: Test grid/list toggle, file preview modal open/close
    verify: "cd ui && npx vitest run src/pages/StoragePage.test.tsx"
  - desc: Test file delete with confirmation, network error handling
    verify: "cd ui && npx vitest run src/pages/StoragePage.test.tsx"
  - desc: Full StoragePage test suite — verify all scenarios pass
    verify: "cd ui && npm test -- StoragePage -- --coverage"

context: |
  StoragePage.tsx (182 lines) exists and renders file list with grid/list toggle,
  upload form, image preview modal, delete, and error state. No test file exists.
  Pattern to follow: FunctionLogsPage.test.tsx (mock fetch, assert rendering
  states, test error paths).
---

## Implementation Steps

**Context**: `StoragePage.tsx` already renders. We need a test file that covers the
component's observable behaviors: list rendering, grid view toggle, upload flow,
image preview modal, file deletion, and error states.

### Step 1: Test file scaffolding

Create `ui/src/pages/StoragePage.test.tsx` with Vitest + React Testing Library,
following the pattern in `FunctionLogsPage.test.tsx`:

- `describe('StoragePage', () => { ... })`
- Mock `globalThis.fetch` for `/api/storage/files` and `/api/storage/upload`
- Wrap component in `<MemoryRouter>`

→ verify: `cd ui && grep -q 'describe.*StoragePage' src/pages/StoragePage.test.tsx`

### Step 2: Render and empty state

Test that the page renders the header "Storage", and shows an empty state (no
file rows) when the API returns an empty array.

→ verify: `cd ui && npx vitest run src/pages/StoragePage.test.tsx -t "empty state"`

### Step 3: File list and grid view

Mock 3 files from the API. Assert:
- All 3 file names appear in list mode (default)
- Toggle to grid mode — verify grid button becomes primary, list button
  secondary
- File sizes are formatted (e.g., "1.5 KB")

→ verify: `cd ui && npx vitest run src/pages/StoragePage.test.tsx -t "list"`

### Step 4: Upload flow

Mock the upload fetch to return 200. Simulate file selection and form submit.
Assert the upload button shows "Uploading..." while the request is in-flight,
then reverts after completion.

→ verify: `cd ui && npx vitest run src/pages/StoragePage.test.tsx -t "upload"`

### Step 5: Image preview modal

Mock files including an image. Click a file name or preview trigger. Assert:
- The preview modal opens with the file name visible
- Close button dismisses the modal

→ verify: `cd ui && npx vitest run src/pages/StoragePage.test.tsx -t "preview"`

### Step 6: Delete with confirmation

Mock `window.confirm` to return true, then mock DELETE endpoint. Assert the file
disappears from the list. Then test the cancel case (`window.confirm` →
`false`).

→ verify: `cd ui && npx vitest run src/pages/StoragePage.test.tsx -t "delete"`

### Step 7: Error states

- Mock upload failure → assert error message "upload failed"
- Mock delete failure → assert error message "delete failed"
- Mock network error on file list fetch → assert error state

→ verify: `cd ui && npx vitest run src/pages/StoragePage.test.tsx -t "error"`

### Step 8: Full suite

→ verify: `cd ui && npm test -- StoragePage -- --coverage`

## Verification Script (Manual)

1. `cd ui && npx vitest run src/pages/StoragePage.test.tsx`
2. Confirm all test suites pass
3. `npm test -- StoragePage -- --coverage` — confirm coverage > 60% lines

## Out of scope
- Drag-and-drop upload (not yet implemented in the page)
- Folder tree (not yet implemented)
- E2E tests

## Risks
- `window.confirm` mock must be restored to avoid leaking between tests
- `fetch` mock must handle both `/api/storage/files` and `/api/storage/upload`
