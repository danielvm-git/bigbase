type: impact-assessment
context: domain
epic: e89
story: e89s05
mode: lightweight

# Impact Assessment: e89s05

## Target

Add the `/secrets` Admin UI for Project → Environment → Folder navigation, masked
secret management, explicit reveal, version history, and safe `.env` import.

## Module purpose, callers, contracts

- Purpose: UI pages/components render and mutate the REST contract without owning
  authorization or plaintext persistence.
- Callers: authenticated Admin UI users through React Router, fetch helpers, and
  Playwright browser sessions.
- Contracts: separate metadata/value TypeScript types; `/value` is the only reveal
  endpoint; 401/403 errors are value-free; list state never contains plaintext;
  import reports key names only; `/secrets` is keyboard-accessible and axe-clean.

## Impact and risks

Medium/high. The UI is isolated from Go source but crosses a security-sensitive REST
boundary and browser state can retain values. Route registration touches App/Layout,
page titles, navigation, and axe inventory.

## Coverage

Scenarios: SC-e89s05-P1-01 through SC-e89s05-P1-06. Component tests run with direct
fetch mocks; browser tests use the existing root Playwright server lifecycle.

## Recommended action

Start in parallel with s07 after s04's REST response shapes and `/value` route are
frozen. Keep the existing Site environment UI unchanged.
