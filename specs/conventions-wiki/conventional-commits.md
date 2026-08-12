# Conventional Commits

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

### Conventional Commits
- All commits must follow the Conventional Commits format:
  - `feat:`, `fix:`, `chore:`, `refactor:`, `docs:`, `test:`, etc.
  - Ex: `feat(auth): add project scoping to JWT`
- Breaking changes must include a `BREAKING CHANGE:` footer.
- The system uses `semantic-release` to automate versioning based on these commit prefixes.
