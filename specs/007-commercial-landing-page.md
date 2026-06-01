# 007 — Appwrite-Style Commercial Landing Page

**type:** feature
**status:** planned
**target:** `components/proxy/proxy.go` — `homeTemplate`

## Structure

```
┌─────────────────────────────────────────────────────────┐
│  STICKY NAV: [B] BigBase · Features · Admin · GitHub ★  │
├─────────────────────────────────────────────────────────┤
│  HERO                                                    │
│  "The open-source BaaS that runs anywhere"               │
│  Single-binary platform with Auth, DB, Storage, Funcs…  │
│  [Launch Admin] [View on GitHub →]                      │
│  ┌─ CSS Dashboard Mockup ─────────────────────────┐     │
│  │  stats cards, activity feed, bar charts         │     │
│  └─────────────────────────────────────────────────┘     │
├─────────────────────────────────────────────────────────┤
│  FEATURES GRID (4×2)                                     │
│  Auth · Database · Storage · Functions · Messaging       │
│  · Deploy · Realtime · Git Repos                         │
├─────────────────────────────────────────────────────────┤
│  COMPONENT STATUS TABLE (live data from kernel)           │
├─────────────────────────────────────────────────────────┤
│  WHY BIGBASE (3×2 differentiators)                       │
│  Single Binary · No Lock-In · ECC Architecture           │
│  Admin UI · Flexible DB · Built-in Auth                  │
├─────────────────────────────────────────────────────────┤
│  CTA: "Ready to build something great?"                  │
│  [Launch Admin →]                                        │
├─────────────────────────────────────────────────────────┤
│  FOOTER (3 columns): Product · Resources · Community     │
└─────────────────────────────────────────────────────────┘
```

## Design Decisions

| Decision | Choice |
|----------|--------|
| Component status table | Keep below features grid |
| GitHub star count | Fetch live from GitHub API |
| Hero visual | CSS-drawn dashboard mockup |
| Accent color | BigBase indigo `#4F46E5` |
| Font | Inter |
| Dark mode | `prefers-color-scheme: dark` |

## Key Implementation Details

- Single file change: `components/proxy/proxy.go` — replace `homeTemplate` string
- Template rendered by Go `html/template` at `GET /`
- GitHub stars fetched from `api.github.com` with 5-min cache
- CSS dashboard mockup is pure CSS, no JS
