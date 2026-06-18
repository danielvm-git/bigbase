# BUG-2026-06-18T174500: Semantic Release CI fails — npx cannot find @semantic-release/changelog

## Problem

The Release & Deploy workflow (`release-deploy.yml`) fails on every push to main:

```
Error: Cannot find module '@semantic-release/changelog'
Require stack: /home/runner/work/bigbase/bigbase/noop.js
```

- **Actual**: CI job fails, release not published, deploy skipped
- **Expected**: Semantic release analyzes commits, publishes version + changelog, deploys
- **Frequency**: Every push to main (observed across PRs #28-#34)
- **First seen**: After `.releaserc` was added with `@semantic-release/changelog` plugin

## Root Cause Analysis

The workflow runs `npx semantic-release` without installing the required plugins:

1. `.releaserc` declares plugins: `@semantic-release/changelog`, `@semantic-release/exec`, `@semantic-release/git`
2. `package.json` does NOT list `semantic-release` or its plugins as devDependencies
3. `npm ci` only installs declared dependencies — plugins are in local `node_modules/` (cached) but not guaranteed
4. `npx semantic-release` downloads a fresh copy to `~/.npm/_npx/` — this copy cannot resolve project-local plugins
5. The `requireStack` shows `/home/runner/work/bigbase/bigbase/noop.js` — npx creates a minimal entry point that fails to resolve

**Risk**: Low — affects only CI release automation, not production code. Workaround: semantic-release is manually usable locally with `npm run` scripts.

## TDD Fix Plan

1. **RED**: The release workflow fails on push to main.
   **GREEN**: Add `semantic-release` and its plugins to `devDependencies` in `package.json`, then run `npm install` to update `package-lock.json`. Update CI to use `npx --no-install semantic-release` so it uses the locally installed version.
   **verify**: `npx --no-install semantic-release --dry-run` exits 0 locally

2. **RED**: CHANGELOG generation may be broken if `.releaserc` references plugins not installed.
   **GREEN**: Verify all `.releaserc` plugins are in devDependencies.
   **verify**: `node -e "const c=require('./.releaserc'); c.plugins.filter(p=>typeof p==='string').forEach(p=>require(p))"` exits 0

## Acceptance Criteria

- [ ] `npm install` adds semantic-release + plugins to package-lock.json
- [ ] CI release job passes on next push to main
- [ ] Local `npx --no-install semantic-release --dry-run` succeeds
- [ ] Existing test suites unaffected (Go + UI tests pass)

## Resolution

<!-- filled in by validate-fix -->
