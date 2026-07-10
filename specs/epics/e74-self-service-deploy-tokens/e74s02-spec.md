# e74s02 — Deploy Keys Tab on Site Detail Page

## Summary

Add a "Deploy Keys" tab to the existing `SiteDetailPage` (`/deploy/:siteId`). Repo owners can generate, copy, list, and revoke `bb_dep_` deploy tokens directly from the browser — no admin or CLI needed.

## Changes

### New React component: `SiteDeployKeysTab.tsx`

Placed alongside the existing tab components at `ui/src/components/`.

### Tab registration

Add a "Deploy Keys" tab to the `SiteDetailPage` tab bar — order it after "Domains" and before "Cache".

### List view

Fetches `GET /api/sites/{siteId}/deploy-keys` and renders a table:

```
┌──────────────────────────────────────────────────┐
│  Deploy Keys                                      │
│                                                   │
│  Name          │ Key (prefix)   │ Last Used │     │
│ ───────────────────────────────────────────────── │
│  grimoire-ci   │ bb_dep_a1b2…  │ 10 min ago │ [🗑]│
│  site-bot      │ bb_dep_d4e5…  │ never      │ [🗑]│
│                                                   │
│  [ + Generate New Deploy Key ]                    │
└──────────────────────────────────────────────────┘
```

### Generate modal

On clicking "+ Generate New Deploy Key":

1. Show a modal with optional "Name" field (defaults to repo name)
2. `POST /api/sites/{siteId}/deploy-keys` with the name
3. On success, display the raw token in a monospace `<code>` block:
   - Copy button (`navigator.clipboard.writeText`)
   - Warning: "This key will not be shown again. Copy it now."
4. On modal close: clear the raw token from React state
5. On component unmount: clear the raw token from React state

### Revoke

On clicking the trash icon:
1. Confirmation dialog: "Revoke deploy key 'grimoire-ci'? This cannot be undone."
2. `DELETE /api/sites/{siteId}/deploy-keys/{keyID}`
3. Remove from list on success

### States

- **Loading**: Skeleton rows
- **Empty**: "No deploy keys. Generate one to deploy from CI/CD."
- **Error**: Toast with error message (generic, no internal details)
- **Rate limited**: "Too many requests. Try again later."

## Verify

```
npx vitest run ui/src/components/SiteDeployKeysTab.test.tsx
npx tsc --noEmit
npm run build --prefix ui
```
