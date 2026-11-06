# Workflow: Solo Git

**type:** workflow
**mode:** solo-git
**context:** BigBase Development Workflow

## Overview

BigBase uses the **solo-git** development workflow designed for high-velocity solo developers operating with full TDD discipline and automated gate verification.

## Principles

1. **Short-Lived Worktrees / Branches:** All work is done on isolated short-lived feature branches or worktrees initialized via `kickoff-branch`.
2. **Protected `main`:** Direct accidental push to `main` is blocked by pre-push hooks and guard-git rules.
3. **Verification Before Land:** Preflight (`npm run preflight`), tests, and lint must pass before landing on `main`.
4. **Fast-Forward Landing:** Code lands on `main` via `scripts/land-branch.sh` without PR ceremony.
5. **Rebase Before Push:** Always `git pull --rebase --autostash` before pushing to `main` to handle automated release bot commits.
6. **Conventional Commits:** All commit messages strictly follow the Conventional Commits specification. No AI attribution (`Co-authored-by`) allowed.

## Command Sequence

```bash
# 1. Start feature
git checkout -b feat/<epic>-<story>

# 2. Develop via TDD
# Write red test -> implement green -> refactor

# 3. Preflight verification
npm run preflight

# 4. Land branch to main
bash scripts/land-branch.sh
```
