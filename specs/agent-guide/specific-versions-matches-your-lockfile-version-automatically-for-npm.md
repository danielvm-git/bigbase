# Specific versions (matches YOUR lockfile version automatically for npm)

source: AGENTS.md
references: [AGENTS.md]

# Specific versions (matches YOUR lockfile version automatically for npm)
rg "ZodError"     $(opensrc path zod@3.22.0)
cat               $(opensrc path pypi:flask@3.0.0)/src/flask/app.py
