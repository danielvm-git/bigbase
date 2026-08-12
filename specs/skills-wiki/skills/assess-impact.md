# assess-impact

source: /Users/danielvm/.claude/skills/assess-impact/SKILL.md
references: [/Users/danielvm/.claude/skills/assess-impact/SKILL.md]
enforced_by: [survey-context, plan-work, verify-work]

Analyze the blast radius of a proposed change before any code is written. Maps dependents, affected stories, and test coverage. Produces specs/IMPACT_LATEST.md. Use before plan-work on any non-trivial change, when touching a shared module, or when the user asks "what does this break?".
