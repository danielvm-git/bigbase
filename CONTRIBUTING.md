# Contributing to BigBase

BigBase is a Go-based Backend-as-a-Service platform with 19 ECC components. Contributions are welcome.

## Before You Start

1. **Read the docs:** [`README.md`](README.md), [`AGENTS.md`](AGENTS.md), [`CONVENTIONS.md`](CONVENTIONS.md).
2. **Run BigBase locally:** `go run . serve --port 8080` and explore the admin UI at http://localhost:8080/admin/.
3. **Understand the ECC pattern:** Components communicate exclusively via the event bus. No direct imports between components. Read `specs/tech-architecture/tech-stack.md`.

## How to Contribute

### Reporting Bugs
- Search [existing issues](https://github.com/danielvm-git/bigbase/issues) first.
- Include: what you did, what you expected, what happened, your OS, Go version, database driver (SQLite or Postgres).
- If the bug is in a specific component, name it.

### Proposing Features
- Open an issue before opening a PR. Align on the approach first.
- New features should fit the ECC pattern — kernel discovers, components register, event bus communicates.

### Pull Request Process

> **External contributors** use the pull-request flow below. **Maintainers**
> follow the solo-git workflow in [`CONVENTIONS.md`](CONVENTIONS.md#git--workflow)
> — short-lived branches/worktrees with direct pushes (or fast-forward merges)
> to `main` once CI is green, always rebasing before push. The two are not in
> conflict: PRs are the contribution path, solo-git is the maintenance path.

1. **Branch from main.** `fix/bug-description` or `feat/feature-description`.
2. **Follow CONVENTIONS.md.** Go conventions, test coverage, no direct component imports.
3. **Run tests:** `go test ./...`
4. **Run lint:** `golangci-lint run ./...`
5. **Run preflight:** `npm run preflight`
6. **Commit:** Conventional Commits format — `feat(component): description` or `fix(component): description`.
7. **Open a PR with `gh pr create`.** Include what you changed, why, and how to verify.

### Code Standards
- Functions: 4–20 lines. Split if longer.
- No direct imports between components — use the event bus.
- Tests: F.I.R.S.T (Fast, Independent, Repeatable, Self-Validating, Timely).
- Coverage gate: 80% minimum.
- No `any` / `interface{}` on public APIs without explicit justification.

## Development Setup
```bash
git clone https://github.com/danielvm-git/bigbase
cd bigbase
go run . serve --port 8080
```

### Prerequisites
- Go 1.26+
- `golangci-lint` (development only)
- SQLite (bundled via `modernc.org/sqlite`) or PostgreSQL for dual-driver testing

## Getting Help
- **Questions?** Open a [GitHub issue](https://github.com/danielvm-git/bigbase/issues).
- **Want to contribute?** Look for issues labeled "good first issue."

---

BigBase is MIT-licensed. By contributing, you agree that your contributions will be licensed under the same terms.
