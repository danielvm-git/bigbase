# Release Plan

## Story 7.0: Appwrite-Style Commercial Landing Page

**type:** feat
**context:** infra | ui
**Context:** Replace the existing bare-minimum component-table landing page at `/` with a full commercial landing page inspired by Appwrite.io. The page includes a sticky nav with live GitHub star count, hero section, product features grid, live component status table, differentiators, CTA, and footer. The entire page is a Go `html/template` string in `components/proxy/proxy.go`.

### Steps

1. **Add GitHub stars fetch with caching to Proxy** → verify: `go test ./components/proxy/ -run TestGitHubStars -v`
   - Add `starsCache` fields to `Proxy` struct
   - Add `fetchGitHubStars()` method with 5-min TTL cache
   - Handle API errors gracefully (show link text without count)
   - Write test: cache hit, cache miss, error handling

2. **Add template data fields for stars and port** → verify: `go test ./components/proxy/ -run TestHomeTemplateData -v`
   - Extend template data struct with `GitHubStars` and `Port`
   - Pass data from `handleHome`
   - Write test: verify data is populated

3. **Replace homeTemplate with commercial landing page (hero + nav)** → verify: `go test ./components/proxy/ -run TestHomePageContent -v`
   - Write the full HTML template with inline CSS
   - Nav: sticky, backdrop-blur, logo, anchor links, GitHub link, "Launch Admin" CTA
   - Hero: headline, subtitle, two CTAs, CSS dashboard mockup
   - Test: HTTP 200, content-type text/html, hero text present

4. **Add features grid + component status table** → verify: `go test ./components/proxy/ -run TestHomePageFeatures -v`
   - 8 feature cards in responsive grid
   - Component status table (existing functionality) rendered below features
   - Test: feature card headings present, component names present in table

5. **Add differentiators, CTA, and footer** → verify: `go test ./components/proxy/ -run TestHomePageSections -v`
   - 6 differentiator cards (3x2 grid)
   - CTA section with "Launch Admin" button
   - Footer with 3 columns, copyright, links
   - Test: differentiator text, footer links present

6. **Add dark mode and responsive CSS** → verify: `go run . serve --port 9999 & curl -s http://localhost:9999/ | grep -i "dark\|@media" > /dev/null`
   - Dark mode via `prefers-color-scheme: dark`
   - Responsive breakpoints at 768px and 480px
   - Visual verification: user opens browser and checks

### Verification Script

1. `go build -o bigbase . && go run . serve --port 9999`
2. Open `http://localhost:9999/` — see sticky nav with BigBase logo and "Launch Admin" button
3. Scroll down — see hero with headline, subtitle, and dashboard mockup
4. Scroll further — see 8 feature cards in 2 rows of 4
5. Below features — see component status table with live data from kernel
6. Below table — see 6 differentiator cards (3x2)
7. Below differentiators — see CTA section with "Launch Admin" button
8. Bottom — see footer with 3 columns
9. Click "Launch Admin" — verify it navigates to `/admin/`
10. `curl http://localhost:9999/health` — verify health endpoint still works
11. Open DevTools → toggle dark mode — verify colors switch

## Out of scope

- Actual admin UI functionality — that's already built in the SPA
- Product detail pages — all anchor links scroll to sections on the same page
- JavaScript interactivity — the landing page is pure CSS/HTML

## Risks

- GitHub API rate limiting (unauthenticated: 60 req/hr) — mitigated by 5-min cache and graceful fallback
- The long template string in Go source may be hard to maintain — acceptable for now, could extract to embedded file later
