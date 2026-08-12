# e89s07 — Secure MCP Secret-Management Tools

**type:** feat
**risk:** P0
**context:** domain
**BCPs:** 3

## 1. Type

Machine-access adapter.

## 2. Context

BigBase already exposes Site env tools through MCP, but generic organization-key
scopes are not enough to authorize arbitrary target Sites or Project secrets.

## 3. Summary

Add scoped MCP access to native secrets and repair target binding for existing Site
environment tools.

## 4. Problem

MCP authentication resolves organization identity, but the adapter forwards a
caller-supplied Site ID without checking ownership.

## 5. Users

CI/CD agents, deployment automation, and operators using MCP.

## 6. Solution

Introduce narrow secret scopes, resolve target ownership before every operation,
return masked metadata by default, and use the same SecretManager policy as REST.

## 7. Alternatives

- Reuse `mcp:provision` for all secret actions: rejected because it is too broad.
- Trust the caller's Site ID: rejected as an IDOR risk.

## 8. Dependencies

e89s01, e89s04, existing MCP bearer auth, organization key resolution, and Site keys.

## 9. Assumptions

MCP value reads are explicit and separately scoped. Site deploy keys remain Site-
scoped and cannot read Project secrets unless an explicit policy grants it.

## 10. Risks

Tool handlers may accidentally stringify underlying errors or leak values through
logs. All tool responses use typed safe result structures.

## 11. Migration Plan

Keep existing Site tools but route them through the ownership-aware SecretManager
adapter. Add native Project tools after REST policy behavior is stable.

## 12. Data Model

No new tables. Add scope/action mapping for MCP credentials and audit actor metadata.

## 13. API

Add:

```text
list_project_secrets
get_project_secret
read_project_secret_value
set_project_secret
delete_project_secret
```

`get_project_secret` is metadata-only and never returns plaintext. The separate
`read_project_secret_value` action requires `mcp:secrets:read`, is audited, and is
available through both HTTP and stdio. Mutations require `mcp:secrets:write` in
addition to the existing generic provisioning scope.

## 14. Affected Code

`components/mcp/auth.go`, `components/mcp/mcp.go`, `adapters.go`, `components/auth`,
`components/sites`, SecretManager adapters, integration tests.

## 15. Testing Strategy

Real MCP HTTP sessions with org keys, cross-org denial, Site-key binding,
read/write scope matrix, masked results, generic errors, and audit events.

## 16. Rollback Plan

Disable native Project tools while retaining ownership-aware Site tools. Do not
restore the old unbound adapter.

## 17. Acceptance Criteria

```gherkin
Scenario: [SC-e89s07-P0-01] MCP read is organization-scoped
  Given an org key for organization A
  When it lists a Project owned by organization B
  Then the tool denies the request without exposing Project metadata

Scenario: [SC-e89s07-P0-02] Read and write scopes differ across transports
  Given an MCP key has secrets:read but not secrets:write
  When it lists through HTTP and stdio and then updates a Secret
  Then both reads succeed and both writes are denied

Scenario: [SC-e89s07-P0-03] Site deploy key cannot cross Sites
  Given a deploy key bound to Site A
  When it calls a Site environment tool for Site B
  Then the tool denies the request
Scenario: [SC-e89s07-P0-04] Existing Site tools enforce target ownership
  Given a Site deploy key is authenticated for Site A
  When an existing Site environment tool receives Site B
  Then the tool denies the request without revealing Site B

Scenario: [SC-e89s07-P1-06] MCP responses remain bounded and masked
  Given a Project secret list or mutation succeeds
  When the tool returns its result and audit metadata
  Then values are masked, arguments are bounded, and actor metadata is safe

Scenario: [SC-e89s07-P0-05] Tool errors are safe
  Given the database returns an internal error
  When an MCP tool fails
  Then the client receives a generic error and no SQL, key, or plaintext value
```

## Requirements
+
#### ADDED: Scoped MCP secret management
MCP MUST bind target organization, Project, Environment, Folder, and Site identities to authenticated credentials, use narrow read/write scopes, return masked metadata by default, and emit safe generic errors.

## 18. Implementation Steps

1. Add narrow MCP secret scopes and target authorization helper (the existing `github.com/modelcontextprotocol/go-sdk/mcp` `[OK]` `AddTool` API accepts a typed handler receiving `context.Context`, `*CallToolRequest`, and input, then returns `*CallToolResult`, output, and error; use that seam for per-tool authorization) → verify: `go test ./components/mcp ./components/auth -run 'Test.*Secret.*Scope|Test.*Org|Test.*Site.*Binding' -count=1`
2. Route existing Site tools through ownership-aware adapter → verify: `go test ./components/mcp -run 'TestEnvVar.*Isolation|TestEnvVar.*Auth' -count=1`
3. Register native Project secret list/get/read-value/set/delete tools with masked and explicit-value contracts → verify: `go test ./components/mcp -run TestProjectSecret -count=1`
4. Add real MCP HTTP and stdio integration with safe-error assertions in `components/mcp` → verify: `go test ./components/mcp -run 'TestMCP.*Secret|Test.*Secret.*MCP|Test.*Stdio.*Secret' -count=1 && echo 'no new security findings in affected paths'`

## 19. Verification Script

1. Create two organizations and one MCP key per organization.
2. Attempt Project and Site reads across organizations.
3. Test read-only and write-scoped keys.
4. Test Site deploy-key target mismatch.
5. Force an internal error and inspect the tool response/logs.

## 20. Out of scope

CLI implementation, SDK generation, Kubernetes delivery, external secret syncs,
and dynamic provider credentials.
## 21. Zoom-Out Check

- **Purpose:** MCP adapts HTTP/stdio tool calls to authenticated SecretManager/Site adapters; Auth resolves principals/scopes; `adapters.go` bridges composition-root implementations.
- **Callers:** MCP HTTP clients, stdio agents, CI/CD automation, organization keys, and Site deploy keys; `main.go` performs coordinator-owned wiring.
- **Contracts:** authenticated org/Site principal context, per-tool `mcp:secrets:read` and `mcp:secrets:write`, Site-key binding, untrusted caller target IDs, metadata-only default results, explicit value reads, bounded arguments, and generic value-free errors across both transports.
- **Reason for Depth:** a shared principal model is required to enforce scope and target binding consistently across MCP HTTP and stdio transports.
