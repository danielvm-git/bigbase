# request-review

source: /Users/danielvm/.claude/skills/request-review/SKILL.md
references: [/Users/danielvm/.claude/skills/request-review/SKILL.md]
enforced_by: [survey-context, plan-work, verify-work]

Dispatch a fresh reviewer agent with a clean context to critique the code after audit-code passes. The reviewer has no shared state with the coding agent and gives a genuine second opinion. Use after audit-code passes, before committing, or when user wants an independent code review.
