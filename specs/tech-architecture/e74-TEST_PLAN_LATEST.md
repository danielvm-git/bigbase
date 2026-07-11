# Test Design: e74 — Self-Service Deploy Tokens (UI + REST)

> **Scope:** Testing strategy and scenario design for site deploy key (`bb_dep_*`) CRUD endpoints and UI management tab.
> **Epic ID:** e74
> **Risk Level:** P0 (contains authentication token generation and credential exposure risks)

---

## 1. Risk Matrix & Scenarios

### P0 — Critical (Core security, authorization & release-blocking)

| Scenario ID | Behavior Description | Risk | Test Level | Target File/Module |
|-------------|----------------------|------|------------|--------------------|
| `SC-e74s01-P0-01` | **Generate Key logic** — `CreateSiteKey` produces a valid `bb_dep_*` token and saves its SHA-256 hash. | P0 | Unit/Integration | `components/auth/sitekeys_test.go` |
| `SC-e74s01-P0-02` | **Revoke Key logic** — `RevokeSiteKey` flags key as revoked; subsequent resolution calls fail. | P0 | Unit/Integration | `components/auth/sitekeys_test.go` |
| `SC-e74s01-P0-03` | **Key Resolution & Context** — Middleware correctly resolves active `bb_dep_*` key to its `site_id` context. | P0 | Integration | `components/auth/sitekeys_test.go` |
| `SC-e74s01-P0-04` | **POST Handler authorization** — Endpoint `/api/sites/{id}/deploy-keys` rejects anonymous/invalid authentication. | P0 | Integration | `components/auth/sitekey_handlers_test.go` |
| `SC-e74s02-P0-01` | **UI Generate Flow** — Click "+ Generate New Deploy Key", submit form, see raw token displayed once. | P0 | UI Unit | `ui/src/components/SiteDeployKeysTab.test.tsx` |
| `SC-e74s02-P0-02` | **UI Revoke Flow** — Click trash icon, confirm in dialog, trigger DELETE request, key is removed from UI list. | P0 | UI Unit | `ui/src/components/SiteDeployKeysTab.test.tsx` |
| `SC-e74s02-P0-03` | **E2E Deploy Key Lifecycle** — Full browser E2E flow: register user, navigate to detail page, open deploy keys tab, generate key, verify raw token is displayed and copyable, close modal, click revoke on list, confirm revocation. | P0 | E2E | `tests/e2e/deploy-keys-ui.spec.ts` |

### P1 — High (Common user flows, regression-prone & input validation)

| Scenario ID | Behavior Description | Risk | Test Level | Target File/Module |
|-------------|----------------------|------|------------|--------------------|
| `SC-e74s01-P1-01` | **Metadata redaction** — `GET /api/sites/{id}/deploy-keys` list endpoint returns metadata only (no raw tokens). | P1 | Integration | `components/auth/sitekey_handlers_test.go` |
| `SC-e74s01-P1-02` | **Site validation** — Key generation fails with 404/not found if targeted site ID does not exist in the database. | P1 | Integration | `components/auth/sitekey_handlers_test.go` |
| `SC-e74s01-P1-03` | **Scope constraints** — Reject deploy key generation requests containing unknown/unauthorized scopes. | P1 | Integration | `components/auth/sitekey_handlers_test.go` |
| `SC-e74s01-P1-04` | **Rate limiting** — Limit deploy key creation requests per site/user to 10 per hour. | P1 | Integration | `components/auth/sitekey_handlers_test.go` |
| `SC-e74s02-P1-01` | **Clipboard copy** — Click CopyButton, verify system clipboard contains the raw token. | P1 | UI Unit | `ui/src/components/SiteDeployKeysTab.test.tsx` |
| `SC-e74s02-P1-02` | **Tab placement** — "Deploy Keys" tab is registered and rendered after "Domains" and before "Cache". | P1 | UI Unit | `ui/src/pages/SiteDetailPage.test.tsx` |

### P2 — Medium (Secondary features & clean state)

| Scenario ID | Behavior Description | Risk | Test Level | Target File/Module |
|-------------|----------------------|------|------------|--------------------|
| `SC-e74s01-P2-01` | **Audit logging** — Verify `auth.site_key_created` and `auth.site_key_revoked` audit events are dispatched on action. | P2 | Integration | `components/auth/sitekey_handlers_test.go` |
| `SC-e74s01-P2-02` | **Log redaction** — Verify the raw generated token is never written to backend stdout/stderr server logs. | P2 | Integration | `components/auth/sitekey_handlers_test.go` |
| `SC-e74s02-P2-01` | **Ephemeral React state** — Clear raw token from React state immediately on modal close to prevent memory leaks. | P2 | UI Unit | `ui/src/components/SiteDeployKeysTab.test.tsx` |
| `SC-e74s02-P2-02` | **State clearance on unmount** — Clear raw token from React state on component unmount to prevent leaks. | P2 | UI Unit | `ui/src/components/SiteDeployKeysTab.test.tsx` |

### P3 — Low (Edge cases, visual states)

| Scenario ID | Behavior Description | Risk | Test Level | Target File/Module |
|-------------|----------------------|------|------------|--------------------|
| `SC-e74s02-P3-01` | **Loading state** — Show SkeletonCard rows during API fetch of list view. | P3 | UI Unit | `ui/src/components/SiteDeployKeysTab.test.tsx` |
| `SC-e74s02-P3-02` | **Empty state** — Display empty state layout with "No deploy keys" message if list is empty. | P3 | UI Unit | `ui/src/components/SiteDeployKeysTab.test.tsx` |
| `SC-e74s02-P3-03` | **Error notification** — Show error toast warning if the list fetch or revocation API fails. | P3 | UI Unit | `ui/src/components/SiteDeployKeysTab.test.tsx` |
| `SC-e74s02-P3-04` | **Rate limit UI response** — Render specific rate-limit warning if API returns 429. | P3 | UI Unit | `ui/src/components/SiteDeployKeysTab.test.tsx` |

---

## 2. Fixture Architecture & Isolation

### Unit & UI Integration Tests (`ui/src/components/SiteDeployKeysTab.test.tsx`)
- **React Testing Library & Vitest**: Mock the server endpoints with structured handlers to simulate success, failure, loading, and rate limiting states.
- **State Verification**: Inspect state changes by spy functions (e.g., verifying `navigator.clipboard.writeText` is invoked with correct value, and checking that the raw token state is null after unmount).

### E2E Test Suite (`tests/e2e/deploy-keys-ui.spec.ts`)
- **Playwright Desktop Browser**: Uses a real Chromium context to walk through the actual UI panels.
- **Pre-seeded Site Fixture**:
  ```typescript
  import { test, expect } from '@playwright/test';
  
  test.describe('Deploy Keys UI Flow', () => {
    let siteId: string;
    let token: string;
    
    test.beforeAll(async ({ request }) => {
      // 1. Register and login to get auth token
      const reg = await request.post('/api/auth/register', {
        data: { email: `e2e-keys-${Date.now()}@test.com`, password: 'TestPass123!' }
      });
      const body = await reg.json();
      token = body.token;
      
      // 2. Provision test git repository
      const repo = await request.post('/api/git/repos', {
        headers: { Authorization: `Bearer ${token}` },
        data: { name: `e2e-repo-${Date.now()}` }
      });
      const repoData = await repo.json();
      
      // 3. Create test site
      const site = await request.post('/api/sites', {
        headers: { Authorization: `Bearer ${token}` },
        data: {
          name: `e2e-site-${Date.now()}`,
          git_repo_id: repoData.id,
          production_branch: 'main',
          root_path: './'
        }
      });
      const siteData = await site.json();
      siteId = siteData.id;
    });
  });
  ```
- **Database isolation**: Uses the fresh SQLite DB file (`/tmp/bigbase-e2e.db`) instantiated by the Playwright web server command. Test cleanup handles removal of records to prevent database pollution.

---

## 3. NFR Verification

| NFR Type | Requirement | Verification Command |
|----------|-------------|----------------------|
| **Perf** | API response latency for generation/revocation is < 150ms under typical loads. | `go test ./components/auth/... -bench=BenchmarkSiteKey` |
| **Security** | Zero raw keys persist in server memory or database logs after revocation. | `sqlite3 /tmp/bigbase-e2e.db "SELECT count(*) FROM org_api_keys WHERE site_id = 'site-1' AND revoked = 0;"` |
| **Reliability** | Clipboard copying function successfully copies the token and triggers standard browser notification hooks. | E2E browser clipboard permissions mock verification. |

---

## 4. Out of Scope

- **Mobile browsers and Safari/WebKit-specific E2E tests**: Confined to Desktop Chrome/Chromium environment.
- **Third-party integrations (e.g. automatically writing deploy keys to GitHub secrets)**: Left as custom scripts; not covered by native frontend E2E flows.
- **Visual styling pixel regression**: Design alignment verification is covered by developer reviews rather than automated visual diff screenshots.
