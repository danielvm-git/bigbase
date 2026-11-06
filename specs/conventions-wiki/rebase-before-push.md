# Rebase before push

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

### Rebase before push
- Always `git pull --rebase --autostash` before pushing to `main`. The
  big-release bot commits to `origin/main` between your fetch and push
  (3 releases/day at peak), which causes non-fast-forward rejections.
- If the release bot beat you, **rebase** onto the new `main` and push again.
- **Never** use `--force`/`--force-with-lease` to win a race against the bot —
  that discards its release commit. Rebase, don't force.
