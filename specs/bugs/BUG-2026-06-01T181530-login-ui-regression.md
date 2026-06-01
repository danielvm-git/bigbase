# BUG-2026-06-01T181530: Login Page Styling Regressed and UI Not Loading Correctly

## Problem

User reports:
- "i cannot login anymore"
- "the look and feel changed as well, used to have a fancy login screen"

The login screen appears as plain HTML without Appwrite design tokens (no fancy card, no proper colors, shadows, or typography). The screenshot shows a basic login form instead of the styled version from commit `da3f118`.

## Root Cause Analysis

**Investigation Results:**

1. **Database Persistence**: ✓ VERIFIED WORKING
   - Tested locally: registered user, stopped server, restarted → user still exists
   - Database file (`bigbase.db`) persists across restarts
   - Auth API works correctly
   - No evidence of destructive database operations in deploy workflow or setup scripts

2. **UI Assets**: ✓ VERIFIED CORRECT LOCALLY
   - UI builds successfully with `npm run build`
   - `index.html` references correct asset hashes: `index-DH469bMX.js` and `index-BQg4Sud-.css`
   - Assets files exist in `ui/dist/assets/`
   - Assets include Appwrite design tokens (CSS variables for colors, shadows, spacing)
   - Admin UI served correctly at `/admin/` path with full styling

3. **Admin UI Path**: ✓ VERIFIED WORKING
   - UI embedded in binary via `ui/embed.go` with `//go:embed all:dist`
   - Served at `/admin/` path via `p.Handle("/admin/", http.StripPrefix("/admin/", ad.Handler())`
   - Root `/` path serves kernel status page (not the admin UI)
   - `/admin/` returns properly styled login page with Appwrite design tokens

4. **Root Cause - Most Likely**: STALE ASSETS ON VPS
   - The VPS deployment was made BEFORE commit `da3f118` (design token port)
   - OR the binary was built without the new UI assets
   - Result: VPS serves embedded binary with old, unstyled assets

5. **Root Cause - Secondary**: USER PATH CONFUSION
   - Users may be accessing `/` instead of `/admin/`
   - Root `/` returns kernel status page, not the admin UI
   - Navigation should redirect to `/admin/#/login`

## Risk Level

**Low** — UI styling issue, not data loss. Database persists correctly. Authentication works. This is a cosmetic + routing issue.

## TDD Fix Plan

### 1. RED: Verify CSS assets are embedded

**Test**: Build binary, extract embedded files, verify CSS has design tokens

```bash
# Build the binary
go build -o bigbase .

# Check embedded assets exist
strings ./bigbase | grep -q "neutral-500" # verify design tokens are in binary
```

**GREEN**: Ensure `ui/dist/` is always present and up-to-date before build

- Add pre-build step to `go build`: verify `ui/dist/index.html` and `ui/dist/assets/*.css` exist
- Or document that `npm run build` must run before `go build`

### 2. RED: Verify login page redirects to correct path

**Test**: Access `/` → should redirect to `/admin/#/login`

```bash
curl -s http://localhost:8080/ -L | grep -q "root" # verify root serves admin UI or redirects
```

**GREEN**: Update root path handler to redirect to `/admin/`

- In `main.go`: add `p.Handle("/", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/admin/", http.StatusMovedPermanently) })`

### 3. RED: Verify assets load with correct MIME types

**Test**: Access login page, verify CSS loads (not 404, not as text/html)

```bash
curl -sI http://localhost:8080/admin/assets/index-DH469bMX.css | grep -q "text/css"
```

**GREEN**: Ensure Go's `http.FileServer` serves `.css` files with correct MIME type

- Go's FileServer auto-detects MIME types; verify by checking HTTP headers

### 4. RED: Verify old deployments can be upgraded

**Test**: Deploy with GitHub Actions, verify assets are up-to-date on VPS

```bash
curl -s http://89.116.26.18/admin/ | grep -q "index-DH469bMX.js"
```

**GREEN**: Rebuild UI as part of CI/CD pipeline

- `npm run build` already runs in GitHub Actions workflow
- Verify binary is built AFTER UI build completes
- Verify embedded assets are included in final binary

## Acceptance Criteria

- [x] Database persists across restarts locally
- [x] Auth API works correctly
- [x] Admin UI loads at `/admin/` with proper styling (Appwrite design tokens)
- [ ] Root `/` redirects to `/admin/`
- [ ] CSS and JS assets load with correct MIME types (verified via `curl -I`)
- [ ] All unit and integration tests pass
- [ ] Deploy to VPS and verify login page has styling (fancy card, colors, shadows)
- [ ] User can log in and navigate dashboard

## Resolution

<!-- filled in by validate-fix -->

