# e59s02: Secrets UI in Admin Panel
## Story ID: e59s02 | Epic: e59 | BCPs: 1 | Status: planned

## 1. Type
**feat** · ui · domain

## 2. Context
With the backend `project_secrets` table and CRUD API implemented in `e59s01`, administrators and developers need a graphical interface to configure project-level secrets. This story implements the Secrets tab inside the `ProjectDetailPage` console view, reusing UI primitives and matching the UX of the existing `SiteEnvVarsTab`.

## 3. Problem / Opportunity
- **CLI Only**: Without a user interface, operators must use direct API requests or CLI commands to manage project secrets, slowing down operations.
- **Poor Discoverability**: Developers cannot easily see which secrets are active in a project, leading to duplication.

## 4. Proposed Solution
1. **Component `ProjectSecretsTab`**:
   Create a new React component at `ui/src/components/ProjectSecretsTab.tsx`.
   - Render a table showing secret keys, values (server-masked `••••••••`), and timestamps.
   - Re-use `AddEditForm` pattern for creating and editing keys.
   - Include drag-and-drop or file import for `.env` files to bulk-load project-level credentials.
2. **Tab Integration**:
   Integrate `ProjectSecretsTab` into `ProjectDetailPage.tsx` (the project details view from `e57s01`) as a dedicated "Secrets" tab.
3. **Data Library**:
   Create `ui/src/lib/secretsData.ts` to manage API communication for project secrets CRUD operations.
4. **Vitest Unit/Integration Tests**:
   Add test files for the new tab component and helper functions.

## 5. Alternatives Considered
- **Dedicated Top-Level Secrets Page**: Overcomplicates navigation. Projects are the natural container for these credentials, so placing management inside `ProjectDetailPage` is contextually relevant.
- **Settings Page tab**: Project secrets belong specifically to a single project, not organization-wide configuration, so putting them in Settings is too broad.

## 6. Who are the users?
- **BigBase Administrators** organizing deployment secrets.
- **Developers** reviewing project credentials.

## 7. Dependencies
- **e57s01 (Project Scoping Admin UI)**: Expects the existence of `ProjectDetailPage.tsx` to insert the new tab.
- **e59s01 (Encrypted Secrets Storage & API)**: Requires the project secrets backend endpoints.

## 8. Assumptions
- The frontend uses React Router to mount pages.
- Standard UI components (Button, Card, Input) from the component library are fully functional.

## 9. Risks
| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| UI Breaks without projects | Medium | High | Ensure mock data or safe checks prevent crashing if no projects exist in the state. |
| Masked secrets overwrite | High | Medium | Follow the SiteEnvVarsTab design where value is left blank during edits; editing requires inputting a new value. |

## 10. Non-goals
- Multi-user access control inside the Secrets tab (all project members/owners can read/write).
- Multi-line secrets editor.

## 11. Migration Plan
No database changes required.
1. Create `ui/src/types/secrets.ts` defining `ProjectSecret`.
2. Create `ui/src/lib/secretsData.ts` implementing `getProjectSecrets`, `createProjectSecret`, `updateProjectSecret`, and `deleteProjectSecret`.
3. Create `ui/src/components/ProjectSecretsTab.tsx` mimicking the structure of `SiteEnvVarsTab.tsx`.
4. Register the new tab in `ProjectDetailPage.tsx`.
5. Add unit and integration tests.

## 12. Wireframes / Diagrams
```
Projects / My App
[ Info ] [ Sites ] [ Deployments ] [ Secrets ]  <-- Selected
                                   -----------------------
                                   [Import .env] [+ Add Secret]
┌────────────────────────────────────────────────────────┐
│ Key             Value            Created      Actions   │
├────────────────────────────────────────────────────────┤
│ DB_URL          ••••••••         Jun 28       [Edit] [x]│
│ API_KEY         ••••••••         Jun 28       [Edit] [x]│
└────────────────────────────────────────────────────────┘
```

## 13. API / Data Model
Matches the types defined in `e59s01`:
```typescript
export interface ProjectSecret {
  id: string
  project_id: number
  key: string
  value?: string
  value_preview: string
  is_build_time: boolean
  is_runtime: boolean
  created_at: string
  updated_at: string
}
```

## 14. Affected Code
| File | Change |
|------|--------|
| `ui/src/types/secrets.ts` | **NEW** — Secret type definition. |
| `ui/src/lib/secretsData.ts` | **NEW** — Fetch helpers for project secrets. |
| `ui/src/components/ProjectSecretsTab.tsx` | **NEW** — Secrets management UI component. |
| `ui/src/pages/ProjectDetailPage.tsx` | Add "Secrets" tab rendering `<ProjectSecretsTab projectId={projectId} />`. |

## 15. Testing Strategy
- **Vitest**:
  - `ProjectSecretsTab.test.tsx` — Test rendering, adding a new secret, deleting a secret, masking verification, and drag-and-drop `.env` file parse.
  - `secretsData.test.ts` — Test fetch requests, response parsing, and error conditions.
- **Vite Build**: Verify `npm run build` is successful.

## 16. Rollback Plan
- Revert changes to `ProjectDetailPage.tsx`.
- Delete `ProjectSecretsTab.tsx`, `secretsData.ts`, and `types/secrets.ts`.
- Rebuild UI.

## 17. Acceptance Criteria
```gherkin
Scenario: Secrets Tab is rendered
  Given a user is on the Project Detail Page for project 10
  When they click the "Secrets" tab
  Then they see a table of secrets and an "Add Secret" button

Scenario: Add a new secret
  Given the Secrets tab is open
  When the user clicks "Add Secret" and inputs key "KEY_A" and value "val-1"
  Then the secret is successfully saved and appears in the table with preview "••••-1"

Scenario: Overwriting value in edit mode
  Given a secret with key "KEY_A" exists
  When the user clicks "Edit", inputs new value "val-2", and saves
  Then the secret value is updated and value preview is updated to "••••-2"

Scenario: Delete secret
  Given a secret exists
  When the user clicks "Delete" and confirms
  Then the secret is deleted and disappears from the table
```

## 18. Implementation Steps
1. Create `ui/src/types/secrets.ts` and `ui/src/lib/secretsData.ts`.
2. Create `ui/src/components/ProjectSecretsTab.tsx`.
3. Add `ProjectSecretsTab` to `ui/src/pages/ProjectDetailPage.tsx` tabs array.
4. Add frontend unit and integration tests.
5. Build the UI project via `npm run build` to verify no compilation/bundling issues.

## 19. Verification Script
1. Run Vitest: `cd ui && npx vitest run src/components/ProjectSecretsTab.test.tsx`
2. Run Vite build: `cd ui && npm run build`
3. Manually run BigBase and verify that the Project Detail page lets you CRUD secrets inside the Secrets tab.

## 20. Out of Scope
- Secret value generation/password generator utility in the form.
- Vault integration.
