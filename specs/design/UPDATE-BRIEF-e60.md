# Update Brief — e60: Route Rename /deploy → /sites

**Prototype file:** `BigBase Console.html`  
**Project:** BigBase Prototype (`ec1480a1`)  
**Effort:** Trivial — label changes only, no layout changes  
**Epic:** e60-route-rename

---

## What to change

### 1. Sidebar nav item label
```
BEFORE: Sites  (links to #/sites — already correct in prototype)
AFTER:  no change needed — prototype already uses "sites" route
```

The prototype's `app.jsx` already routes to `'sites'`, `'sites/new'`, `'sites/:id'`. No routing change needed in the prototype.

### 2. Breadcrumbs

In **Create Site Wizard** (`bb/pages-sites.jsx`), the breadcrumb reads:
```
Sites  /  Create site
```
This is already correct — no change needed.

### 3. Screen inventory note in prototype
There is nothing to change in the HTML. The prototype never used `/deploy` — it was already correct.

---

## Result

No prototype edit required. The prototype pre-empted the rename. This brief is a confirmation only.

**Mark e60 as prototype-aligned before implementation.**
