# guard-git

source: /Users/danielvm/.claude/skills/guard-git/SKILL.md
references: [/Users/danielvm/.claude/skills/guard-git/SKILL.md]
enforced_by: [survey-context, plan-work, verify-work]

Block dangerous git commands (push, force push, reset --hard, clean, branch -D, checkout/restore .) and enforce Conventional Commits & Branch Protection before an AI agent runs them. Installs hook scripts for Claude Code, Cursor, Cursor CLI, and Gemini CLI; documents Google Antigravity Terminal deny lists. Use when the user wants git safety hooks, to block git push or destructive git in agents, or to mirror the same policy across AI coding tools.
