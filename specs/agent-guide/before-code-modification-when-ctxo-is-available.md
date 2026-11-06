# Before Code Modification (when ctxo is available)

source: AGENTS.md
references: [AGENTS.md]

### Before Code Modification (when ctxo is available)
1. Call `get_blast_radius` for the symbol you are about to change — understand what breaks
2. Call `get_why_context` for the same symbol — check for revert history or anti-patterns
3. Only then read and edit source files
