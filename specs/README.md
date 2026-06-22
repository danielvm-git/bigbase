# BigBase Specs (bigpowers YAML layout)

YAML files are the **source of truth** for agents. Legacy markdown lives under `specs/archive/` (read-only).

## Metadata (`type:` / `context:`)

Every planning YAML file starts with provenance fields (see `plan-work` skill):

| Field | Meaning |
|-------|---------|
| `type:` | Artifact kind (`session-state`, `release-plan`, `epic-shard`, `requirements-scope`, …) |
| `context:` | `domain` (product/requirements) or `infra` (tooling, session, CI) |

Epic shards use `type: epic-shard` and `context: domain`. Dev dependency slopcheck tags live in [`dependencies.yaml`](dependencies.yaml).

## Quick index

| If you want to… | Read |
|-----------------|------|
| Session state and next skill | [`state.yaml`](state.yaml) |
| Epic index (WSJF, no status) | [`release-plan.yaml`](release-plan.yaml) |
| Story/epic status | [`execution-status.yaml`](execution-status.yaml) |
| In/out of scope | [`requirements/SCOPE_LATEST.yaml`](requirements/SCOPE_LATEST.yaml) |
| Domain glossary | [`requirements/GLOSSARY_LATEST.yaml`](requirements/GLOSSARY_LATEST.yaml) |
| Architecture and stack | [`tech-architecture/tech-stack.md`](tech-architecture/tech-stack.md) |
| Active epic tasks | [`epics/`](epics/) shard for `active_epic_id` in `state.yaml` |
| ADRs | [`adr/`](adr/) |
| Bug investigations | [`bugs/`](bugs/) + [`bugs/registry.yaml`](bugs/registry.yaml) |

## Epic ID mapping

| Legacy | YAML |
|--------|------|
| Epic 017 / slice `017-A` | `e17` / `e17s01` |
| Epic 018 / `018-A` | `e18` / `e18s01` |
| … | … |

Design docs for Epic 017: [`epics/e17-enhanced-admin-ui/`](epics/e17-enhanced-admin-ui/).

## Visual dashboard (ephemeral)

The [visual-dashboard](.opencode/skills/visual-dashboard/SKILL.md) skill stores **session-only** artifacts under `.bigpowers/dashboard/<pid>-<timestamp>/` (HTML screens, server state). That tree is gitignored and is not source of truth — the cockpit reads YAML from `specs/` via `read-specs-status.cjs`. Safe to delete old session folders when the server is stopped.

After `start-server.sh --project-dir $(pwd)`, open the printed **`cockpit_url`** (or root `/`, which redirects there). Optional agent HTML still lives in `content/*.html`.

## Commands

```bash
bash scripts/validate-specs-yaml.sh
npm run test:specs          # same validator via node:test wrapper
npm run test:dashboard      # visual-dashboard route + parser tests
bash scripts/sync-status-from-epics.sh   # refresh execution-status.yaml from epics
```

## v1.0 slice history

Implemented component specs: [`archive/slices/`](archive/slices/) (`001-cli-*.md` … `015-deploy-github-journey.md`).
