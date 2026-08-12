# e89s05 — Admin UI for Project Secrets

**type:** feat
**risk:** P1
**context:** domain
**BCPs:** 3

## 1. Type

Admin UI feature.

## 2. Context

The current Site environment editor is flat and site-only. Native secrets require
Project, Environment, Folder, and masked-value navigation.

## 3. Summary

Add a Project secret-management view using existing BigBase UI primitives and
REST contracts.

## 4. Problem

Operators cannot manage reusable Project secrets without duplicating values across
Sites or using undocumented APIs.

## 5. Users

Organization administrators and Project operators.

## 6. Solution

Provide Project → Environment → Folder navigation, masked tables, explicit reveal,
create/update/delete, version history, and safe `.env` import.

## 7. Alternatives

- Extend `SiteEnvVarsTab` into a second flat table: rejected because it hides scope.
- Add export in the first release: rejected because masked list values cannot produce a safe export.

## 8. Dependencies

e89s02, e89s04, existing React/Vite tokens and Admin UI routing.

## 9. Assumptions

Reveal is an explicit action and requires the same read-value policy as REST.

## 10. Risks

Browser state and error messages must not retain values longer than required or
render server-provided values as HTML.

## 11. Migration Plan

Keep the existing Site environment tab. Add a separate Project Secrets surface.

## 12. Data Model

Use metadata types that omit plaintext by default. A value-read response is a
separate type and is never reused for list rows.

## 13. API

Use e89s04 endpoints. Import sends individual validated writes and reports each
failed key without returning the submitted values.

## 14. Affected Code

`ui/src/pages`, `ui/src/components`, `ui/src/lib`, `ui/src/types`, route tests,
accessibility tests, and Playwright E2E.

## 15. Testing Strategy

Vitest component/helper tests, typecheck/build, accessibility tests, and browser
E2E for masking, reveal, permissions, import, and destructive confirmation.

## 16. Rollback Plan

Remove the Project Secrets route/tab. Existing Site environment UI remains intact.

## 17. Acceptance Criteria

```gherkin
Scenario: [SC-e89s05-P1-01] Project secret list is masked
  Given the operator opens a Project Environment Folder
  When secrets load
  Then keys and masked previews appear without plaintext values

Scenario: [SC-e89s05-P1-02] Reveal is explicit and authorized
  Given the operator has read-value permission
  When they choose Reveal for one Secret
  Then only that Secret value is shown and the read is auditable

Scenario: [SC-e89s05-P1-04] Edit requires a replacement value
  Given an existing Secret is selected for editing
  When the form opens
  Then the value input is empty rather than prefilled with plaintext

Scenario: [SC-e89s05-P1-05] Import reports partial failure safely
  Given an import file contains valid and invalid keys
  When import completes
  Then valid keys are saved, invalid keys are reported by name, and values are not displayed in errors
Scenario: [SC-e89s05-P1-03] Unauthorized UI actions stay value-free
  Given the operator lacks read-value or mutation permission
  When Reveal or a mutation fails
  Then the UI shows a value-free 401/403 error and retains no submitted secret

Scenario: [SC-e89s05-P1-06] Project Secrets UI is accessible
  Given the operator navigates the `/secrets` route
  When keyboard navigation and the accessibility scan run
  Then labels, focus, confirmation controls, and axe checks pass
```

## Requirements
+
#### ADDED: Project secret administration UI
The Admin UI MUST manage Project Environment Folder secrets with masked values, explicit authorized reveal, immutable version history, safe import, and value-free errors.

## 18. Implementation Steps

1. Add Project Secrets data types and fetch helpers with separate metadata/value types → verify: `cd ui && npx vitest run src/lib/secretsData.test.ts`
2. Add masked Project Secret table, Folder navigation, and mutation forms → verify: `cd ui && npx vitest run src/components/ProjectSecretsTab.test.tsx`
3. Add explicit reveal, version history, import handling, and safe errors → verify: `cd ui && npx vitest run src/components/ProjectSecretsTab.test.tsx -t 'mask|reveal|import|version'`
4. Integrate `/secrets` navigation, accessibility route coverage, browser E2E, typecheck, and production build → verify: `cd ui && npx tsc --noEmit && npm run build && cd .. && npx playwright test tests/e2e/project-secrets-ui.spec.ts --config tests/e2e/playwright.config.ts && npx playwright test tests/e2e/axe-scan.spec.ts --config tests/e2e/playwright.config.ts --grep 'Project Secrets' && echo 'no new security findings in affected paths'`

## 19. Verification Script

1. Open a Project Environment in the Admin UI.
2. Confirm values are masked.
3. Create and update a Secret.
4. Reveal it with and without permission.
5. Import a mixed-validity `.env` file.
6. Confirm no error message displays submitted values.

## 20. Out of scope

Secret export, password generation, approvals, sharing, and advanced Infisical UI features.
## 21. Zoom-Out Check

- **Purpose:** the Admin UI renders Project secret navigation and invokes REST helpers; it does not own authorization or persistence.
- **Callers:** authenticated React Router users, fetch helpers, component tests, Playwright browser sessions, and existing design-system primitives.
- **Contracts:** `/secrets` route, separate metadata/value types, `/value` as the only reveal path, value-free 401/403 rendering, no plaintext in list state or import errors, and accessible keyboard/focus/axe behavior.
- **Reason for Depth:** separate metadata/value types prevent a reveal response from being reused by list state or mutation rendering.
