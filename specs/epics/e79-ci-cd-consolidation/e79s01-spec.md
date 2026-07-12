# e79s01 — Consolidate CI/CD Workflows

## Scope
Migrate BigBase's CI/CD workflows to match the consolidated templates in `danielvm-git/.github`, as outlined in Issue #95.

## Current State
| Workflow | Purpose |
|----------|---------|
| `ci.yml` | SQLite + PostgreSQL matrix test, Allure report to GitHub Pages |
| `codeql.yml` | Go + JavaScript/TypeScript security scanning |
| `release-deploy.yml` | Semantic release + SSH deploy to Contabo VPS |
| `pr-review.yaml` | PR-Agent automated code review |

## Target State
| Workflow | Action |
|----------|--------|
| `ci-cd.yml` (NEW) | Replace `ci.yml` with consolidated Go template, preserving PostgreSQL matrix + Allure report |
| `codeql.yml` | Update to new template format |
| `release-deploy.yml` | Keep as-is (special case: SSH deploy to Contabo VPS) |
| `pr-review.yaml` | Keep as-is (not part of consolidation) |

## Key Decisions
- Go version stays at 1.26 (template defaults to 1.22)
- Keep PostgreSQL matrix test strategy (not in template)
- Keep Allure report generation (not in template)
- Keep semantic-release in `release-deploy.yml` (already works)
- Do NOT include deploy job from template — SSH deploy is separate
- Do NOT include verify/semantic-release from template — handled by release-deploy.yml

## Out of Scope
- Changing `release-deploy.yml` SSH deploy logic
- Modifying `pr-review.yaml`
- Updating other repos
