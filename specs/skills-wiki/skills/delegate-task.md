# delegate-task

source: /Users/danielvm/.claude/skills/delegate-task/SKILL.md
references: [/Users/danielvm/.claude/skills/delegate-task/SKILL.md]
enforced_by: [survey-context, plan-work, verify-work]

Delegate one complex task to a single subagent, review its work in two stages before merging back. Sequential — one agent at a time, with oversight. Use when a task is complex and requires careful review before the result is accepted. Distinct from dispatch-agents (no parallelism here; reviewer sees full diff before proceeding).
