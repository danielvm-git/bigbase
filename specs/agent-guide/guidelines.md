# Guidelines

source: AGENTS.md
references: [AGENTS.md]

### Guidelines
- For high-risk symbols, call `get_blast_radius` before editing when ctxo is available.
- Check `get_why_context` for high-risk or historically unstable symbols.
- Prefer `search_symbols` for indexed symbol lookup, but use LSP or grep when the index is stale or the target is plain text/configuration.
- Use `find_importers` for dependency tracing when available; verify critical callsites with LSP or repository search.
