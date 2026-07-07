## Target

Epic e67 — MCP provisioning tools across `components/mcp`, `components/git`, `components/sites`, `components/auth`, `components/deploy`, `kernel`, and `main.go`.

## Dependents

- `components/mcp`: MCP clients call tools over stdio/HTTP; existing `deploy_site`, `list_repos`, status, logs, and knowledge tools must keep their current behavior.
- `components/git`: HTTP `/api/git/repos` and admin/API callers depend on `git_repos` creation semantics; e67 adds a public `CreateRepo` method over the same behavior.
- `components/sites`: HTTP `/api/sites` and admin UI depend on site creation and deploy callback behavior; e67 adds a public `CreateSite` method and name defaulting from `git_repos`.
- `components/auth`: Protected routes depend on JWT and `bb_` API key middleware; e67 adds `bb_dep_` site keys and must preserve org-scoped API keys.
- `components/deploy`: `/api/deploy` callers depend on `HandleCreate` accepting normal JWT/API-key deploys; e67 adds site-key authorization enforcement.
- `kernel`: shared component contracts and neutral context helpers; e67 adds `CtxSiteID`/`SiteIDFromContext` to avoid direct component imports.
- `main.go`: composition root wires MCP options and protected route middleware.

## Affected Stories

- `e67s01`: MCP `create_repo`
- `e67s02`: MCP `create_site`
- `e67s03`: MCP `provision_ci_credentials`
- `e67s04`: MCP `get_ci_template`
- `e57`: prerequisite multi-tenant DB/auth/sites hard gate; e67 remains blocked until e57 completes.

## Test Coverage

- `components/mcp/mcp_test.go`: in-memory MCP session tests for tool registration and responses.
- `components/git/git_test.go`: repo creation and duplicate-name behavior.
- `components/sites/sites_test.go`: site creation, deploy callback, domain/custom-domain surrounding behavior.
- `components/auth/apikeys_test.go` and `components/auth/auth_test.go`: API key creation, hash lookup, middleware context behavior.
- `components/deploy/deploy_test.go`: deploy HTTP handler behavior, site_id handling, deploy lifecycle paths.
- Gap to close during e67s03: explicit cross-site rejection test for `bb_dep_` site key A attempting deploy to site B.

## Risk: Medium

Most e67 work is narrow MCP/interface wiring, but e67s03 touches credential authentication plus deploy authorization, making the epic medium overall with P0 tasks inside the hard-gate story.

## Recommended Action

Proceed with `plan-work` after e57 hard gate lands. Build order: e67s04 → e67s01 → e67s02 → e67s03. Treat e67s03 as the hard-gate story and require auth, deploy, and full-suite verification before marking it complete.
