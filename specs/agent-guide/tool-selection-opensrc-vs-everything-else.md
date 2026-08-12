# Tool Selection: opensrc vs EVERYTHING else

source: AGENTS.md
references: [AGENTS.md]

### Tool Selection: opensrc vs EVERYTHING else

| Your intent | RIGHT tool | WRONG tool (and why) |
|-------------|-----------|----------------------|
| How does `zod.parse()` handle this edge case? | `opensrc path zod` + `rg` | `web_search` — blog posts and docs won't show the actual code path |
| What does React's `useEffect` actually schedule internally? | `opensrc path react` + `rg` | `fetch_content` of react.dev — docs describe behavior, not implementation |
| Why does Prisma generate this SQL for my query? | `opensrc path prisma` + `rg` | `context7` MCP — gives API signatures, not query-generation internals |
| What does serde's `Deserialize` derive macro expand to? | `opensrc path crates:serde` + `rg` | `bts_find` — only searches our codebase, not the dependency |
| How does Express route matching work under the hood? | `opensrc path express` + `rg` | `read` — only reads files in our project, not node_modules |
| Study how Tailscale's DERP relay handles NAT traversal | `opensrc path tailscale/tailscale` + `rg` | `bts_map` — maps our repo structure, not theirs |
| Clone a repo to read its source locally | `opensrc path owner/repo` | `bash` + `git clone` — opensrc is shallow, cached, version-aware, and 10x faster |
| What does Flask's `@app.route` decorator actually register? | `opensrc path pypi:flask` + `rg` | `fetch_content` of flask.palletsprojects.com — docs show usage, not internals |
| Find how a Go library implements an interface | `opensrc path <pkg>` + `rg` | `web_search` for godoc — godoc shows signatures, not implementation |
| Check a dependency's package.json scripts or metadata | `cat $(opensrc path <pkg>)/package.json` | `read` on `node_modules/<pkg>/package.json` — opensrc gives the full repo context |
| Read a library's CHANGELOG to understand a breaking change | `fetch_content <repo-url>/blob/main/CHANGELOG.md` | `opensrc` — fetching a whole clone for one markdown file is overkill |
| Find API usage examples or community patterns | `web_search` or `fetch_content` | `opensrc` — source won't show how real users compose the API |
| Check if a library is actively maintained | `web_search` or `fetch_content <repo-url>` | `opensrc` — you want GitHub activity, not source files |
| Get the official docs for a function signature | `context7` MCP or `web_search` | `opensrc` — docs are faster than source for signatures |
