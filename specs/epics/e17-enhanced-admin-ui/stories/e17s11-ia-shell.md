---
id: e17s11
title: IA shell and nav parity
status: done
wsjf: 3.2
tasks:
  - desc: Rebuild Layout sidebar groups (Overview, Build, Data, Auth, Engage, DevOps)
    verify: "cd ui && npm run build && grep -E 'Build|Engage|Auth' src/Layout.tsx"
  - desc: Label Sites nav to /deploy; document sites ≡ deploy in story context
    verify: "grep -q 'Sites' ui/src/Layout.tsx && grep -q '/deploy' ui/src/Layout.tsx"
  - desc: Move dark-mode toggle to sidebar footer with Appearance label
    verify: "grep -q 'Appearance' ui/src/Layout.tsx"
  - desc: Add Settings nav link to /settings (stub route until e17s17)
    verify: "grep -q '/settings' ui/src/Layout.tsx ui/src/App.tsx"
  - desc: Replace letter sidebar icons with shared Icon component
    verify: "test -f ui/src/components/Icon.tsx && grep -q Icon ui/src/Layout.tsx"
  - desc: Layout smoke test — IA sections render
    verify: "cd ui && npx vitest run src/Layout.test.tsx 2>/dev/null || cd ui && npm run build"

acceptance: |
  Given an authenticated user on any console page
  When the sidebar is visible
  Then section overlines read OVERVIEW, BUILD, DATA, AUTH, ENGAGE, DEVOPS (case as implemented)
  And Build contains Sites (href /deploy) and Functions
  And Engage contains Messaging only
  And DevOps contains Git Repos, CI/CD, Monitoring plus platform extras Forge and Realtime
  And footer shows Appearance (dark toggle) and Settings link

context: |
  Prototype IA: specs/archive/assets/bigbase-prototype/project/Component Spec - Information Architecture.html
  Route alias: prototype `sites` ≡ app `/deploy`, `/deploy/new`, `/deploy/:siteId` (no URL rename).
  Accent theme picker lands in e17s12; Settings page body in e17s17.
---
