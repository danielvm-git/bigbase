# Entity-Component-Construct (ECC)

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

### Entity-Component-Construct (ECC)
- **Entity** = The running BigBase server
- **Component** = Independent submodule with its own lifecycle (auth, db, proxy...)
- **Construct** = The config that decides which components run together
