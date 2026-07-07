# e67s04: MCP get_ci_template Tool

**Story ID:** e67s04 | **Epic:** e67 — MCP Provisioning Tools | **BCPs:** 1 | **Status:** planned

## 1. Type & Context

**type:** feat
**context:** domain
**maturity:** 3 — Countable

## 2. Story Statement

**As an** AI coding agent using the BigBase MCP server,
**I want** to retrieve the canonical CI/CD workflow YAML for deploying to BigBase,
**so that** I can write it directly to `.github/workflows/deploy.yml` without guessing the pipeline architecture or manually translating deploy API calls.

## 3. Context

The MCP component already loads knowledge data from embedded JSON files (`knowledge/services.json`, `knowledge/frameworks.json`, `knowledge/examples/code-examples.json`) via `//go:embed` directives in `knowledge.go`. This story follows the same pattern: add a new embedded knowledge file for CI/CD templates and a tool to retrieve them.

### Zoom-Out Summary

- **Module purpose:** `components/mcp` — MCP server with knowledge tools.
- **Pattern:** `//go:embed knowledge/...` → typed Go struct → `mcpsdk.AddTool`.
- **No new Go interfaces or component wiring needed.** Pure knowledge data.

## 4. Domain Model

New knowledge file: `components/mcp/knowledge/ci-templates.json`

```json
{
  "templates": [
    {
      "id": "github-actions-deploy",
      "name": "GitHub Actions Deploy",
      "app_types": ["node", "go", "python", "static"],
      "description": "Deploy to BigBase on every push to main",
      "content": "name: Deploy to BigBase\non:\n  push:\n    branches: [main]\n..."
    }
  ]
}
```

## 5. Contract / Interface

MCP tool signature:
```
get_ci_template(app_type?, platform?)
  → YAML content string (ready to paste into .github/workflows/deploy.yml)

get_ci_template()
  → list of available templates (no args = catalog)
```

## 6. Implementation Strategy

1. Create `components/mcp/knowledge/ci-templates.json` with the canonical GitHub Actions workflow for each `app_type`.
2. Add `//go:embed knowledge/ci-templates.json` directive to `knowledge.go`.
3. Define `ciTemplateEntry` struct and `ciTemplatesData` container (matching existing pattern in `knowledge.go`).
4. Add `loadCITemplates()` loader function.
5. Register `get_ci_template` tool in `NewMCPServer()`:
   - No args → list available templates.
   - `app_type` provided → return template with variable placeholders (`${SITE_ID}`, `${BIGBASE_SERVER}`, `${TOKEN}`).
6. Templates use `${{ secrets.BIGBASE_SITE_ID }}`, `${{ secrets.BIGBASE_DEPLOY_TOKEN }}`, `${{ secrets.BIGBASE_SERVER }}` as placeholders — the agent can substitute or tell the user to set them via `gh secret set`.

### Template content

The canonical workflow YAML:

```yaml
name: Deploy to BigBase
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Deploy to BigBase
        run: |
          curl -sfX POST ${{ secrets.BIGBASE_SERVER }}/api/deploy \
            -H "Authorization: Bearer ${{ secrets.BIGBASE_DEPLOY_TOKEN }}" \
            -H "Content-Type: application/json" \
            -d '{
              "site_id": "${{ secrets.BIGBASE_SITE_ID }}",
              "branch": "main"
            }'
```

## 7. Data Flow

```
AI agent: get_ci_template({"app_type": "static"})
  → knowledge.go: loadCITemplates()
  → filter templates by app_type
  → return formatted YAML string
  → AI agent: write to .github/workflows/deploy.yml

AI agent: get_ci_template()
  → return available template catalog
```

## 8. Error Handling

| Condition | Tool response |
|-----------|---------------|
| Unknown `app_type` | "No template for 'foo'. Available types: node, go, python, static" |
| Templates fail to load | "Error loading CI templates: <err>" |

## 9. Testing Strategy

- **Unit — by app_type:** Request `static` → YAML contains `BIGBASE_SITE_ID` and `BIGBASE_DEPLOY_TOKEN`.
- **Unit — no args:** Returns catalog listing available app_types.
- **Unit — unknown type:** Returns "no template for ...".
- **Verify YAML is valid:** Parse the YAML content in the test (not strictly required but good hygiene).

## 10. Migration / Rollback

No schema changes. Rollback = remove CI template JSON file, `//go:embed` directive, Go struct, loader, tool registration.

## 11. Documentation

Update `specs/tech-architecture/tech-stack.md` MCP section to list `get_ci_template`.

## 12. Dependencies

- No new Go dependencies (stdlib only).
- `gopkg.in/yaml.v3` already in `go.mod` but not needed — templates are stored as plain text strings with variable placeholders.

## 13. Observability

```go
c.logger.Info("mcp tool", "tool", "get_ci_template", "app_type", appType)
```

## 14. Security

**Security level:** none — templates are public knowledge data.

## 15. Acceptance Criteria

```gherkin
Scenario: get_ci_template returns YAML for an app type
  When an AI agent calls get_ci_template with {"app_type": "static"}
  Then the response is valid YAML
  And it contains "BIGBASE_SITE_ID" and "BIGBASE_DEPLOY_TOKEN" placeholders
  And it contains a "curl" command targeting /api/deploy

Scenario: get_ci_template without args lists available templates
  When an AI agent calls get_ci_template with no arguments
  Then the response lists available app_types

Scenario: get_ci_template returns helpful message for unknown type
  When an AI agent calls get_ci_template with {"app_type": "unknown"}
  Then the response says "No template for 'unknown'"
```

## 16. Out of Scope

- Per-framework template variants (SvelteKit build step vs. Go build step). Each app_type gets one canonical template; framework-specific build logic is documented elsewhere.
- Dynamic template generation with actual site_id/token substitution (agent handles placeholder replacement).
- Multi-platform templates (GitLab CI, CircleCI, etc.) — GitHub Actions only for v1.

## 17. Requirements

#### ADDED: MCP `get_ci_template` knowledge tool
No args → catalog of available templates. `app_type` arg → YAML string with `${{ secrets.BIGBASE_SITE_ID }}`, `${{ secrets.BIGBASE_DEPLOY_TOKEN }}`, `${{ secrets.BIGBASE_SERVER }}` placeholders targeting `POST /api/deploy`.

#### ADDED: Embedded `knowledge/ci-templates.json`
Follows existing `//go:embed` pattern in `knowledge.go` (same as `services.json`, `frameworks.json`). One template per app_type: `node`, `go`, `python`, `static`.

## 18. Risks

- **Placeholder drift:** If deploy API shape changes, templates must be updated in JSON — no runtime generation.
- **No auth required:** Templates are public knowledge; secrets are GitHub Actions `${{ secrets.* }}` references only.

## 19. Verification Script

1. `python3 -c 'import json; json.load(open("components/mcp/knowledge/ci-templates.json"))'` — valid JSON
2. `go test ./components/mcp/ -run TestGetCITemplate -v -count=1` — catalog, lookup, unknown type
3. `go test ./... -count=1` — full suite green
