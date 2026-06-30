# Update Brief — e61: Secrets Manager UI

**Prototype file:** `BigBase Console.html`  
**Project:** BigBase Prototype (`ec1480a1`)  
**Effort:** Small — add Secrets tab to Project Settings  
**Epic:** e61-secrets  
**Depends on:** UPDATE-BRIEF-e58 applied first (Project Settings screen must exist)

---

## What to change

Add a **Secrets** tab to Project Settings (`/project/:id/settings`), between "Env Vars" and "Danger Zone".

### Updated Project Settings tab bar
```
[General]  [Database]  [Env Vars]  [Secrets]  [Danger Zone]
```

### Secrets tab content

Same visual pattern as Env Vars tab (masked values, eye toggle, edit/delete), but with a scope note at top:

```
Project Secrets                                    [+ Add secret]
Encrypted key-value pairs injected into all sites in this project.
Site-level env vars override these when keys collide.

KEY                  VALUE              Build  Runtime  Actions
DATABASE_URL         ••••••••  [👁]    [✓]    [✓]      [✏] [🗑]
STRIPE_SECRET_KEY    ••••••••  [👁]    [ ]    [✓]      [✏] [🗑]
```

- "Build" and "Runtime" columns: small checkbox badges showing injection scope
- Values masked as `••••••••` with eye toggle (same as Site Env Vars tab)
- Scope note is a muted `<p>` below the header row

**Add secret modal:**
```
┌──────────────────────────────────────────┐
│ Add secret                             ✕ │
│                                          │
│ [Key *                               ]   │
│ [Value *                             ]   │
│                                          │
│ ☑ Inject at build time                   │
│ ☑ Inject at runtime                      │
│                                          │
│ [Cancel]  [Add secret]                   │
└──────────────────────────────────────────┘
```

---

## Mock data to add (`bb/data.js`)

```js
projectSecrets: {
  p1: [
    { key: 'DATABASE_URL', value: 'postgres://prod/mydb', build: true, runtime: true },
    { key: 'STRIPE_SECRET_KEY', value: 'sk_live_xxxx', build: false, runtime: true },
  ]
}
```
