# gate-trace

source: /Users/danielvm/.claude/skills/gate-trace/SKILL.md
references: [/Users/danielvm/.claude/skills/gate-trace/SKILL.md]
enforced_by: [survey-context, plan-work, verify-work]

Deterministic traceability quality gate — reads coverage matrix + blind-spot data, applies decision rules with oracle confidence downgrade, emits PASS/CONCERNS/FAIL/WAIVED verdict. Use before release-branch to gate merges on traceability.
