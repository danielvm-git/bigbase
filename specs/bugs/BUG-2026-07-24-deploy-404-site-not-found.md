---
bug_id: BUG-2026-07-24-deploy-404-site-not-found
status: fixed
severity: critical
priority: critical
scope: sites,deploy,auth,mcp
title: Deploy HTTP 404 site not found after #163 — opaque identity mismatch + missing framework/runtime hard stops
github_issues: [155, 161, 163]
consumer_issues:
  - danielvm-git/big-exames#3
  - danielvm-git/grimoire#5
  - danielvm-git/big-library#4
related:
  - BUG-2026-07-24-deploy-key-organization-required
  - BUG-2026-07-24T184443
  - BUG-2026-07-24T091500
security_impact: MEDIUM
---

# BUG-2026-07-24-deploy-404-site-not-found: Never-bounce deploy identity + stack matrix

## Problem

After #163 / v2.79.10 fixed `403 organization required` for `bb_dep_*` keys, consumer deploys still bounce. Live second failure:

- [big-exames run 30133487221](https://github.com/danielvm-git/big-exames/actions/runs/30133487221) → **HTTP 404** `{"error":"site not found"}`
- Same staircase on grimoire#5 / big-library#4
- Opaque error: cannot tell missing site vs key↔SITE_ID mismatch
- Deploy first-classes only `node|go|python|static` — no Astro/SvelteKit mode detection; pnpm historically fell through poorly; no `AppPHP`

**Expected:** sealed credential bundle → preflight → ownership with distinct codes → `git_repo_id` bind → correct runtime/framework detection → host assert. Failures return `error` + `code` + `hint`.

**Security impact: MEDIUM** — opaque 404 for mismatch is intentional (no cross-tenant leak), but operators need a distinct `code` for DX without leaking peer site existence. No org_id injection for `bb_dep_*`. Exploit path: none beyond existing IDOR surface already gated by site-key match.

## Root Cause Analysis (diagnose-root)

### Reproduce
1. Prod v2.79.10; exames CI secrets set 2026-07-24 ~21:12–21:29Z
2. POST `/api/sites/{SITE_ID}/deploy` with `Authorization: Bearer bb_dep_*`
3. Response: HTTP 404 `site not found` (run 30133487221)

### Isolate
- `requireSiteOwnership` site-key branch returns the same JSON for (a) missing site row and (b) key site ≠ URL site
- Prod SQLite (2026-07-24):

| site | id | git_repo_id | repo name |
|------|-----|-------------|-----------|
| exames | `82e8cdd6…` | `508da18e…` | danielvm-git-big-exames |
| grimoire | `c81e929d…` | `c01f332d…` | danielvm-git-grimoire |
| library | `21fc7f88…` | `bed3cea0…` | danielvm-git-big-library |

- **No duplicate site names** for exames/grimoire/library
- **Zero `org_api_keys` rows** for exames site `82e8cdd6…`
- Orphan active key `#13` `bb_dep_613dad` → site_id `87049ff6…` (**site row deleted**)
- Timing: key #13 created 21:11:56Z ≈ BIGBASE_SITE_ID secret update 21:12:10Z → secrets point at orphaned identity

### Hypothesize (ranked)
1. **Key↔SITE_ID / orphan site** — token bound to deleted site; URL SITE_ID is canonical or orphan → 404. **Falsify:** key.site_id ∈ sites.id for exames. **Result: confirmed** (no exames keys; orphan key #13).
2. **#155 wrong-repo** — redeploy clones wrong `git_repo_id`. **Falsify:** exames.git_repo_id = danielvm-git-big-exames. **Result: falsified** for this failure (binding correct).
3. **Duplicate exames from org_id recovery** — second site shadows first. **Falsify:** `GROUP BY name HAVING count>1`. **Result: falsified today** (still ship unique-name guard).

### Verify
Single identity root cause for the live 404: **orphaned deploy key / SITE_ID bundle not bound to canonical exames site**. Platform gaps that will cause the next bounce: unstructured errors, no unique name, redeploy URL-id fallback, missing PHP/framework-mode detection.

Risk: **High** (fleet blocked; recurring consumer staircase).

## TDD Fix Plan

1. **RED**: site-key mismatch returns `code=key_site_mismatch` + hint; missing site returns `code=site_not_found` + hint; no org_id injection
   **GREEN**: structured JSON from `requireSiteOwnership` site-key branch
   **verify**: `rtk go test ./components/sites/ -run 'SiteKeyOwnership|StructuredError' -count=1`

2. **RED**: second `insertSite` with same name in org → 409 / error
   **GREEN**: unique name check (org-scoped) before INSERT
   **verify**: `rtk go test ./components/sites/ -run 'UniqueSiteName|DuplicateName' -count=1`

3. **RED**: redeploy with URL site id always triggers deploy with that site's `git_repo_id` (never URL-as-repo fallback when site missing after ownership)
   **GREEN**: resolve site row once; pass canonical site id + `git_repo_id`
   **verify**: `rtk go test ./components/sites/ -run 'Redeploy.*Repo|RepoBinding' -count=1`

4. **RED**: `DetectAppType` → php for `composer.json`; pnpm lock → no npm fallback; go/python missing-tool structured codes
   **GREEN**: `AppPHP` + detection + tool-missing errors
   **verify**: `rtk go test ./components/deploy/ -run 'DetectAppType|NodePackageManager|AppPHP|ToolMissing' -count=1`

5. **RED**: Astro static/server/hybrid + SvelteKit static/node/ambiguous (+ npm|pnpm fixtures, root_path)
   **GREEN**: `DetectFrameworkMode` maps to static|node|ambiguous; `app_type_mismatch` when explicit disagrees
   **verify**: `rtk go test ./components/deploy/ -run 'FrameworkMode|Astro|SvelteKit' -count=1`

6. **RED**: MCP `validate_ci_credentials` + templates for php/astro/sveltekit modes
   **GREEN**: tool + ci-templates.json rows
   **verify**: `rtk go test ./components/mcp/ -run 'ValidateCI|GetCITemplate|Provision' -count=1`

7. **Org action**: bigbase-deploy@v1 maps codes; accepts php + framework app_types; pin consumers
8. **Fleet**: fresh keys for exames/grimoire/library → secrets → deploy → host assert
9. **validate-fix** + **release-branch**

## Acceptance Criteria

- [ ] Identity codes `site_not_found` / `key_site_mismatch` with hints; no org_id injection for bb_dep_
- [ ] Unique site name per org
- [ ] Redeploy bound to `sites.git_repo_id`
- [ ] `AppPHP` + Node pnpm never falls back to npm + Go/Python tool codes
- [ ] Astro + SvelteKit mode detection tests (static/SSR/hybrid/ambiguous) with npm|pnpm + root_path
- [ ] MCP validate + templates; bigbase-deploy preflight
- [ ] Fleet smoke green; consumer #3/#4/#5 closed with evidence

## Resolution

Fixed in fix/deploy-404-never-bounce:

- Structured `site_not_found` / `key_site_mismatch` (+ hints) in requireSiteOwnership; no org_id injection for bb_dep_
- Unique site name per org (`site_name_taken`)
- Redeploy binds to sites.id + sites.git_repo_id (no URL-as-repo fallback)
- AppPHP + pnpm lockfile priority + tool_missing codes; Astro/SvelteKit framework mode detection
- Framework-static with package.json runs Node build before serve; root_path honored
- MCP `validate_ci_credentials` + CI templates (php, astro/sveltekit aliases)
- Org action bigbase-deploy@v1 preflight/code mapping (danielvm-git/.github#20); exames pinned to @v1
- Fleet: fresh bb_dep_ for exames/grimoire/library; secrets rotated; identity deploy 201; exames root_path=web

Prod note: full built-app host content for framework-static requires this binary on VPS after release, then redeploy.

## Prod diagnosis evidence (2026-07-24)

- exames/grimoire/library: one row each; git_repo names match consumer repos (#155 falsified for this 404)
- No exames-bound `org_api_keys`; orphan key #13 → deleted site `87049ff6…`
- Fleet realign required before closing consumer issues
