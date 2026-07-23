---
bug_id: BUG-134
status: fixed
severity: critical
scope: security
title: "CRITICAL: Cross-Tenant Deployment Hijack / Custom-Domain Takeover"
github_issue: 134
created_at: 2026-07-13
---

## Summary

Site-ownership is checked only when authenticating via site deploy key (`bb_dep_*`). For JWT/API-key auth, `kernel.SiteIDFromContext` returns `ok=false` and the ownership check is skipped — attacker's `site_id` in the request body is trusted.

## Exploit

Any authenticated user POSTs `/api/deploy` with their own `repo_id` but a victim's `site_id`. After build passes health check, BigBase repoints the victim's verified custom domain to the attacker's deployment and kills the victim's live process — full site/domain takeover + DoS.

## Root Cause

In `components/deploy/gateway.go:52-59`, the site ownership check only executes when `kernel.SiteIDFromContext(r.Context())` returns `ok=true`:

```go
if siteID, ok := kernel.SiteIDFromContext(r.Context()); ok && siteID != "" {
    if req.SiteID != "" && req.SiteID != siteID {
        // reject mismatch
    }
    req.SiteID = siteID
}
```

For JWT/API-key auth:
- `SiteIDFromContext` returns `("", false)` — check is skipped entirely
- Attacker's `req.SiteID` from request body is passed directly to `Trigger()`
- No validation that the site belongs to the caller's org

## Affected Files

- `components/deploy/gateway.go:52-59` — missing ownership check for JWT/API-key auth
- `components/deploy/engine.go:20` — `Trigger()` accepts arbitrary site_id without validation

## Fix

1. Add `org_id` column to `sites` table
2. In `HandleCreate`, when JWT/API-key auth is used:
   - If `site_id` is provided, verify the site exists AND `site_id.org_id == caller.org_id`
   - Reject with 403 if ownership check fails
3. Update site creation to populate `org_id`

## Verify

- [ ] Attacker with valid JWT cannot deploy to another org's site
- [ ] Site deploy key auth still works (existing flow)
- [ ] New sites are created with correct org_id
- [ ] All existing deploy tests pass
