---
description: Push branch, open PR, merge when preflight and CI pass (bigpowers release-branch)
agent: build
---

Ship this feature using the bigpowers **release-branch** workflow. Load the `release-branch` skill first (`skill({ name: "release-branch" })`) and follow it.

## Preconditions

1. Run `npm run preflight` — must pass (go vet + tests + ui/dist/index.html).
2. Run `/check-stack` if agentic wiring changed.
3. All commits on this branch follow Conventional Commits 1.0.0.

## Steps

1. Final verification on the feature branch:
   - `npm run preflight`
   - `git log main...HEAD --oneline` — no WIP/debug commits
   - `git diff main...HEAD --stat` — no secrets in diff

2. Push the branch:
   - `git push -u origin HEAD`

3. Open PR with Conventional Commits title (feeds semantic-release on merge to `main`):
   - `gh pr create --title "<type>(<scope>): <description>" --body "$(cat <<'EOF'
## Summary
- Implements feature: {{SHIP_SUMMARY}}
- Go binary + embedded React admin UI
- semantic-release on merge to main

## Verify
- [ ] `npm run preflight` passed locally
- [ ] CI green
EOF
)"`

4. After CI is green, merge (squash or merge per project policy):
   - `gh pr merge --auto --squash` or `gh pr merge` when ready
   - User approval required if `permission` has `gh pr merge*` as `ask`

5. `release.yml` on `main` runs semantic-release after merge.

Report: PR URL, CI status, merge result.
