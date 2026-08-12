# Defensive Code Categories

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

## Defensive Code Categories
- **Rate limit** — all API endpoints
- **Retry with backoff** — external calls (DB, OAuth, S3)
- **Timeout** — all network operations
- **Graceful degradation** — if Redis/dependency is down, fall back
