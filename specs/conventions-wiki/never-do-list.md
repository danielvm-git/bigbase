# Never-Do List

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

## Never-Do List
- **Never** commit secrets, private keys, or API tokens.
- **Never** force-push (`--force` or `--force-with-lease`) to `main`.
- **Never** include `Co-authored-by` or any AI attribution in git commits.
- **Never** bypass access controls, permission checks, or security gates.
- **Never** directly import between components without going through kernel event bus.
- **Never** mutate inputs or assert on timing sleeps in tests (poll or synchronize channels).
