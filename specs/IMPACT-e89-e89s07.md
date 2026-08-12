type: impact-assessment
context: domain
epic: e89
story: e89s07
mode: lightweight

# Impact Assessment: e89s07

## Target

Add scoped MCP Project-secret tools and repair existing Site environment target
binding, with separate read/write scopes and safe generic errors.

## Module purpose, callers, contracts

- Purpose: `components/mcp` translates HTTP/stdio tool calls into authenticated
  adapters; `components/auth` resolves principals and scopes; `adapters.go` bridges
  composition-root implementations.
- Callers: MCP HTTP clients, stdio agents, CI/CD automation, Site deploy keys, and
  organization API keys.
- Contracts: authenticated organization/Site principal context; per-tool
  `mcp:secrets:read` and `mcp:secrets:write`; Site keys remain Site-bound; caller target
  IDs are not trusted identity; metadata is masked by default; errors never stringify
  SQL, keys, or plaintext; HTTP and stdio behavior remain equivalent.

## Impact and risks

Critical. Existing MCP auth has coarse tier checks and high fan-out context callers.
Changing `WithOrgAuth`/tool enforcement can regress existing read-tier behavior, so
both transports and the prior `list_site_keys` read classification require tests.
Composition-root `main.go` wiring remains coordinator-owned.

## Coverage

Scenarios: SC-e89s07-P0-01 through SC-e89s07-P0-05 and SC-e89s07-P1-06.

## Recommended action

Start in parallel with s05 after s04 and the MCP principal/scope contract freeze.
Consume the same SecretManager policy as REST; do not add a separate authorization
implementation.
