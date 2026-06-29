# e61s01: Encrypted Secrets Storage & API
## Story ID: e61s01 | Epic: e61 | BCPs: 2 | Status: planned

## 1. Type
**feat** · domain

## 2. Context
BigBase supports site-scoped environment variables (`site_env_vars`), which are encrypted at rest and injected into build/runtime processes. However, as organizations grow and group sites under **projects** (introduced in `e57`), managing identical secrets across multiple sites becomes repetitive and error-prone. This story introduces project-level encrypted secrets storage, a secure REST API for managing them, and logic to inject them into deployments.

## 3. Problem / Opportunity
- **No Shared Configs**: Operators must manually replicate common keys (e.g. database URLs, shared API tokens) across every site in a project.
- **High Risk of Divergence**: Updating a shared credential requires modifying multiple site env vars, causing potential drift and breakage.
- **Lack of Project-Level Secrets**: No centralized project-level store exists to manage project-scoped credentials securely.

## 4. Proposed Solution
1. **Database Schema**:
   Add a `project_secrets` table to store encrypted project-level environment variables.
2. **REST API**:
   Add endpoints in the `Deploy` component to perform CRUD on project secrets, scoped by `{projectId}`:
   - `GET /api/projects/{projectId}/secrets` — List secrets (values masked).
   - `POST /api/projects/{projectId}/secrets` — Create a secret.
   - `PUT /api/projects/{projectId}/secrets/{key}` — Update secret value.
   - `DELETE /api/projects/{projectId}/secrets/{key}` — Delete a secret.
3. **Authorization**:
   Ensure requests are authenticated via JWT or API Key. Check that the project's `org_id` matches the user's `org_id` from context.
4. **Build & Runtime Injection**:
   Modify site build and start flows to fetch project secrets, decrypt them, and inject them alongside site env vars. Site-scoped env vars override project-scoped secrets if key collision occurs.

## 5. Alternatives Considered
- **Stashing Secrets in Orgs**: Org-level secrets are too broad and leak credentials across unrelated projects. Project-scoping provides the correct boundary of least privilege.
- **Unencrypted Storage**: Storing sensitive secrets in plaintext violates security baselines. Reusing the AES encryption key (via `envcrypto`) provides strong security with no new dependencies.

## 6. Who are the users?
- **Developers** deploying multiple sites in a project.
- **Operators** configuring shared environment settings.

## 7. Dependencies
- **e57s02 (Projects Table CRUD)**: The `project_secrets` table has a foreign key referencing `projects`.
- **components/internal/envcrypto**: Reused for AES-GCM encryption/decryption of secret values.

## 8. Assumptions
- The `envcrypto` encryption key is configured via `--env-encryption-key` or `BIGBASE_ENV_ENCRYPTION_KEY`.
- The database schema supports foreign key constraints (SQLite/PostgreSQL).

## 9. Risks
| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Key collision | Low | Medium | Define merge precedence: site env vars override project secrets. |
| DB schema lock | Low | High | Use idempotent `CREATE TABLE IF NOT EXISTS` migration during startup. |
| Secret leakage in logs | Medium | High | Mask values in list API responses. Ensure values are never written to server or build logs. |

## 10. Non-goals
- Multi-environment secret overrides (e.g. staging vs prod secrets).
- Integrations with external secret managers (HashiCorp Vault, AWS Secrets Manager).

## 11. Migration Plan
Run an idempotent migration in `Deploy.Start()`:
```sql
CREATE TABLE IF NOT EXISTS project_secrets (
    id TEXT PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value_encrypted TEXT NOT NULL DEFAULT '',
    is_build_time INTEGER NOT NULL DEFAULT 0,
    is_runtime INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(project_id, key)
);
CREATE INDEX IF NOT EXISTS idx_project_secrets_project_id ON project_secrets(project_id);
```

## 12. Wireframes / Diagrams
```
+------------------+
|  projects table  |
+--------+---------+
         | 1
         |
         | *
+--------+---------+      +-------------------+
| project_secrets  |      |   site_env_vars   |
+--------+---------+      +---------+---------+
         |                          |
         +------------+-------------+
                      | merge (site overrides project)
                      ▼
            Build/Runtime Process
```

## 13. API / Data Model
### JSON Payload structures
- **ProjectSecret**:
  ```json
  {
    "id": "abc-123",
    "project_id": 42,
    "key": "DATABASE_URL",
    "value_preview": "••••••••",
    "is_build_time": false,
    "is_runtime": true,
    "created_at": "2026-06-28T12:00:00Z",
    "updated_at": "2026-06-28T12:00:00Z"
  }
  ```

### API Endpoints
- `GET /api/projects/{projectId}/secrets` -> returns `{"data": [ProjectSecret, ...]}` (Values masked in previews)
- `POST /api/projects/{projectId}/secrets` -> accepts `{"key": "K", "value": "V", "is_build_time": false, "is_runtime": true}` -> returns `ProjectSecret` with plaintext value
- `PUT /api/projects/{projectId}/secrets/{key}` -> accepts `{"value": "V2", "is_build_time": false, "is_runtime": true}` -> returns updated `ProjectSecret`
- `DELETE /api/projects/{projectId}/secrets/{key}` -> returns `204 No Content`

## 14. Affected Code
| File | Change |
|------|--------|
| `components/deploy/deploy.go` | Run table migration in `Start()`. Register `/api/projects/` routes. |
| `components/deploy/secrets.go` | **NEW** file containing Project Secrets CRUD logic, HTTP route handlers, and DB querying helpers. |
| `components/deploy/deploy.go` | Retrieve and merge project secrets into site build and start cmd environments. |
| `components/deploy/deploy_test.go` | Add integration tests verifying secret injection and precedence. |
| `main.go` | Route `/api/projects/` traffic to `depComp.Handler()`. |

## 15. Testing Strategy
- **Unit/Integration Tests**:
  - `TestProjectSecretsCRUD` — Verify creation, updating, deletion, and retrieval of secrets.
  - `TestProjectSecretsAuth` — Verify that cross-org requests to secrets are blocked (403).
  - `TestDeploySecretsInjection` — Build and run a mock app, verify project-level secrets are injected, and site-level overrides work correctly.

## 16. Rollback Plan
- Revert route registration in `main.go` and `deploy.go`.
- Drop `project_secrets` table via manual migration if needed (forward-only migration is standard).

## 17. Acceptance Criteria
```gherkin
Scenario: Create project secret
  Given an authenticated user who owns org 1 with project 10
  When POST /api/projects/10/secrets with {"key": "API_KEY", "value": "secret-123"}
  Then response is 201 with secret metadata and masked preview "••••-123"

Scenario: Block cross-org access to secrets
  Given a user authenticated to org 2
  When GET /api/projects/10/secrets
  Then response is 403 Forbidden

Scenario: Project secrets injected to site deployments
  Given project 10 has secret API_KEY=secret-123
  And site A belongs to project 10
  When site A is deployed
  Then the running process has env var API_KEY=secret-123

Scenario: Site env var overrides project secret
  Given project 10 has secret DB_URL=postgres://prod
  And site A has site env var DB_URL=postgres://site-specific
  When site A is deployed
  Then the running process has env var DB_URL=postgres://site-specific
```

## 18. Implementation Steps
1. Create `project_secrets` migration in `deploy.go` `Start()`.
2. Create `components/deploy/secrets.go` and implement `ProjectSecret` CRUD handlers.
3. Wire `/api/projects/` route in `deploy.go` `Handler()` and `main.go`.
4. Implement `FetchProjectSecrets()` and integrate it into `buildApp` and `startApp` process execution in `deploy.go`.
5. Add unit and integration tests under `components/deploy/`.

## 19. Verification Script
1. Run all deploy tests: `go test -v ./components/deploy/ -run TestProjectSecrets`
2. Start local server: `go run . serve --port 9999 --db :memory:`
3. Verify via curl requests creating, listing, and deleting secrets.

## 20. Out of Scope
- Secret value search/filtering.
- Exposing decrypted secrets in listing responses.
