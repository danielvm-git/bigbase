# Security

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

## Security
- No secrets in code. Use env vars.
- `httpOnly` + `Secure` + `SameSite=Strict` cookies
- Parameterized queries only (no SQL concatenation)
- Validate all input with schemas
- Generic error messages to clients, full details in logs
