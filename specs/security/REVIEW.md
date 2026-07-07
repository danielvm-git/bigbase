# Security Review — Epic e67: MCP Provisioning Tools

**Date:** 2026-07-07
**Branch/Diff:** `e67-mcp-provisioning-tools` (31 files, 1674 insertions)
**Threat Model:** `specs/security/epics/e67/THREAT_MODEL.md`

## Scope Resolution
This review covers all changes introduced in Epic e67 including:
- MCP provisioning tools: create_repo, create_site, provision_ci_credentials, get_ci_template
- Site-scoped deploy keys (bb_dep_ prefix) in `components/auth/apikeys.go`
- Git CreateRepo refactor in `components/git/git.go`
- Sites CreateSite/insertSite refactor in `components/sites/sites.go`
- Kernel WithSiteID/SiteIDFromContext in `kernel/scope.go`
- CI template knowledge loading in `components/mcp/knowledge.go`
- Wire-up in `main.go`

## Vulnerability Assessment

### 1. SQL Injection
All new/refactored queries use parameterized placeholders (`?`) exclusively:
- `components/auth/apikeys.go` — `ResolveSiteKey`, `createSiteKeyRecord`
- `components/git/git.go` — `CreateRepo` INSERT
- `components/sites/sites.go` — `insertSite` INSERT, git_repo validation SELECT
- `components/deploy/deploy.go` — pre-existing `ensureSiteKeyAuth` (out of scope)

**Verdict:** No findings. Confidence: 10/10.

### 2. Auth Bypass / Privilege Escalation
- `provision_ci_credentials` generates site-scoped keys (`bb_dep_` prefix) via `createSiteKeyRecord`
- Key generation uses `crypto/rand` (same as existing `generateRawAPIKey`) ✅
- Raw key returned once; SHA-256 hash stored via existing `hashAPIKey()` ✅
- `ResolveSiteKey` enforces `revoked = 0` and `site_id IS NOT NULL` ✅
- `org_id` set to 0 for site-scoped keys (correct by design — scoped to site, not org) ✅
- Deploy middleware enforces `SiteIDFromContext` == `req.SiteID` on `bb_dep_` authenticated requests ✅

**Verdict:** No findings. Confidence: 10/10.

### 3. Path Traversal
- `components/git/git.go` `CreateRepo` uses `filepath.Join(g.dir, id+".git")` where `id` is from `generateID()` (crypto-random, hex-encoded). **User-controlled `name` is never used in filesystem path.** ✅
- `components/sites/sites.go` `insertSite` uses `generateID()` for primary key only; no filesystem operations. ✅

**Verdict:** No findings. Confidence: 10/10.

### 4. Secrets Exposure
- `provision_ci_credentials` returns raw `bb_dep_` token in MCP response — this is the **intended behavior** of a credential-provisioning tool. Caller is responsible for secure handling (documented: `gh secret set BIGBASE_DEPLOY_TOKEN`). ✅
- Logs only `key_id` and `site_id`, never raw token. ✅

**Verdict:** No findings (by design). Confidence: 10/10.

### 5. Command Injection
- All MCP tool arguments parsed via `json.Unmarshal` into typed Go values. No shell execution, no `exec.Command` in any new code path. ✅

**Verdict:** No findings. Confidence: 10/10.

### 6. Context Key Collision
- `kernel/scope.go` uses typed `siteIDKeyType` string type for context keys (not raw string) — no collision with `project_id` key or any third-party context values. ✅

**Verdict:** No findings. Confidence: 10/10.

## Conclusion
All security concerns identified in `specs/security/epics/e67/THREAT_MODEL.md` have been addressed with concrete mitigations. No unresolved HIGH or MEDIUM findings remain.

**Verdict:** PASS
