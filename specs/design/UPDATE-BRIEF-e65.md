# Update Brief — e65: Preview Environments

**Prototype file:** `BigBase Console.html`  
**Project:** BigBase Prototype (`ec1480a1`)  
**Effort:** Small — add Previews tab to Site Detail  
**Epic:** e65-preview-envs

---

## What to change

Add a **Previews** tab to the Site Detail page, between "Domains" and "Env Vars".

### Updated Site Detail tab bar
```
[Deployments]  [Logs]  [Domains]  [Previews]  [Env Vars]  [Cache]  [Manifest]
```

### Previews tab content

```
Preview Environments                              [+ Create preview]
Automatic preview deployments for open pull requests.

┌──────────────────────────────────────────────────────────────────────────┐
│ PR #42 — feat/dark-mode    [RUNNING]   preview-42.bigbase.local  ↗  [✕] │
│ PR #38 — fix/login-bug     [BUILDING]  preview-38.bigbase.local     [✕] │
│ PR #31 — chore/deps        [STOPPED]   preview-31.bigbase.local     [✕] │
└──────────────────────────────────────────────────────────────────────────┘
```

Each preview row:
- PR number + branch name (monospace for branch)
- Status badge (same `statusVariant` as deployments)
- Preview URL (monospace, accent color, external link icon when RUNNING — no link when not)
- [✕] danger sm button to delete the preview environment

**Create preview modal:**
```
┌──────────────────────────────────────────────┐
│ Create preview environment                ✕  │
│                                              │
│ [Branch name or PR #              ]          │
│                                              │
│ A preview URL will be created at:            │
│ preview-{branch}.bigbase.local               │
│                                              │
│ [Cancel]  [Create preview]                   │
└──────────────────────────────────────────────┘
```

**Empty state (no previews):**
```
No preview environments.
Open a pull request or manually create a preview to see it here.
[+ Create preview]
```

**Settings note (below the list):**
```
Auto-preview settings
☑ Auto-create previews for new pull requests
☑ Auto-delete previews when PR is closed
```
Two checkboxes, `bb-section-label` above them.

---

## Mock data to add (`bb/data.js`)

```js
previews: {
  s1: [
    { pr: 42, branch: 'feat/dark-mode', status: 'running', url: 'preview-42.bigbase.local' },
    { pr: 38, branch: 'fix/login-bug', status: 'building', url: 'preview-38.bigbase.local' },
    { pr: 31, branch: 'chore/deps', status: 'stopped', url: 'preview-31.bigbase.local' },
  ]
}
```
