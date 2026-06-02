---
description: Verify agentic stack tools are installed and wired for BigBase (Go + React)
agent: build-error-resolver
---

Run a full agentic stack health check for BigBase. For each item, run the shell command and report PASS / FAIL / MISSING:

1. Runtime: `go version` (expect >=1.26.3) and `node --version` (expect v24.x)
2. `opencode --version`
3. `test -f opencode.json`
4. `test -f AGENTS.md`
5. `test -f .opencode/commands/check-stack.md`
6. `npm ls bigpowers` (from project root) 2>/dev/null || `npx bigpowers --version`
7. `npx context-mode --version`
8. `npx sqz-mcp --version` 2>/dev/null || `npx sqz --version` 2>/dev/null
9. `test -f .opencode/plugins/rtk.ts` || `test -f ~/.config/opencode/plugins/rtk.ts`
10. `npx opensrc --version`
11. `npx opensrc list` — confirm present: `danielvm-git/bigpowers`, `npm:react`, `npm:vite`, `github:appwrite/appwrite`
12. `find .opencode/skills -name SKILL.md 2>/dev/null | wc -l` — expect > 0 after `npm run sync:opencode-skills`
13. `test -d specs`
14. `gh auth status` 2>/dev/null || echo "MISSING: gh not authenticated"
15. `test -f .github/workflows/ci.yml`
16. `jq -e '.plugin | index("context-mode")' opencode.json`
17. `jq -e '.mcp.sqz and .mcp.context7' opencode.json`
18. `jq -e '.mcp.ctxo' opencode.json`
19. `jq -e '.lsp.gopls and .lsp.typescript' opencode.json`
20. `npm run preflight` — only if package.json defines scripts.preflight
21. `gopls version`; `test -f ui/dist/index.html`; `go vet ./...`

Summarise: | Tool | Status | Version/Detail |. One-line fix per FAIL/MISSING.
