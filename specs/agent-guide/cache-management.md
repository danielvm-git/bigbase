# Cache management

source: AGENTS.md
references: [AGENTS.md]

# Cache management
opensrc list                # what's cached
opensrc list --json         # machine-readable
opensrc remove zod          # evict one package
opensrc clean --npm         # wipe npm cache
opensrc clean --pypi        # wipe PyPI cache
opensrc clean               # wipe everything
```
