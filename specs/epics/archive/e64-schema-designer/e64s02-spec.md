### Story e64s02: Schema Designer UI — Implementation Steps

**type:** feat
**context:** domain
**Context**: This story implements the frontend React component for the Schema Designer. It allows administrators to view all custom collections, inspect their schemas, and interactively add or drop columns. It relies on the backend endpoints built in `e64s01`.

## Steps

1. Create `SchemaDesigner.tsx` page component in `ui/src/pages/` to fetch and list collections from `GET /api/schema`. → verify: `npm run test -- SchemaDesigner.test.tsx`
2. Create a "Collection View" panel to display the columns of the selected collection in a data table. → verify: `npm run test -- SchemaDesigner.test.tsx`
3. Add a "New Column" modal/dialog that POSTs to `/api/schema/{collection}/columns` and refreshes the schema. → verify: `npm run test -- SchemaDesigner.test.tsx`
4. Add a "Drop Column" action with a destructive confirmation dialog that DELETEs to `/api/schema/{collection}/columns/{column}`. → verify: `npm run test -- SchemaDesigner.test.tsx`
5. Wire the Schema Designer route to the Admin Dashboard sidebar navigation. → verify: `npm run test -- Layout.test.tsx`

## Verification Script (Step-by-Step)

1. Start the server using `go run .` and the UI using `npm run dev` in `ui/`.
2. Navigate to the Admin Dashboard in the browser.
3. Click on "Schema Designer" in the sidebar.
4. Verify the list of existing collections is displayed.
5. Click on a collection and verify its columns (e.g., `id`, `org_id`, `data`) are listed.
6. Click "Add Column", enter `price` and `INTEGER`, and submit. Verify the column appears in the list.
7. Click "Drop" next to `price`, confirm the warning dialog, and verify the column is removed.

## Out of scope

- Visual relationship/foreign-key builder.
- Visual data browser (that belongs to a Content Manager feature).
- Renaming columns (backend doesn't support it yet).

## Risks

- Destructive actions (dropping columns) without proper warnings. A double-confirmation modal must be used.
