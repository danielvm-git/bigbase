---
id: e17s18
title: Responsive cross-links and accessibility
status: done
implemented_in: dff889f
wsjf: 2.0
tasks:
  - desc: Add max-width 1024px and 375px media rules per Responsive spec
    verify: "grep -E '1024|375' ui/src/index.css"
  - desc: Dashboard stat cards link to deploy functions repos users areas
    verify: "grep -E 'navigate|Link|to=' ui/src/pages/DashboardPage.tsx | head -5"
  - desc: Data Studio Query this and Functions logs View all to monitoring
    verify: "grep -i monitoring ui/src/pages/DataStudioPage.tsx ui/src/pages/FunctionDetailPage.tsx ui/src/pages/FunctionsPage.tsx 2>/dev/null; cd ui && npm run build"
  - desc: Sidebar toggle and icon-only controls have aria-label
    verify: "grep aria-label ui/src/Layout.tsx ui/src/components/Icon.tsx 2>/dev/null; grep aria-label ui/src/Layout.tsx"
  - desc: Full UI test suite and preflight UI
    verify: "cd ui && npm test && cd .. && npm run preflight:ui"

acceptance: |
  Given viewport width 1024px or below
  When viewing dashboard or tables
  Then padding and grids follow Responsive spec tiers
  Given dashboard stat cards
  When clicked
  Then user navigates to the matching product area
  Given keyboard user
  When tabbing sidebar links
  Then focus ring is visible on each interactive control

context: |
  Cross-links from IA spec section 5.3 (8 documented jumps).
  Depends on e17s11–s17 screens existing for link targets.
---
