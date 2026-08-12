# deploy

source: /Users/danielvm/.claude/skills/deploy/SKILL.md
references: [/Users/danielvm/.claude/skills/deploy/SKILL.md]
enforced_by: [survey-context, plan-work, verify-work]

Build → verify artifact → deploy → wait → smoke deployment pipeline. Platform-agnostic (MCP or CLI), with configurable timeout, retry with exponential backoff, and integrated health-check. The deploy half of CI/CD: run after build to push to production.
