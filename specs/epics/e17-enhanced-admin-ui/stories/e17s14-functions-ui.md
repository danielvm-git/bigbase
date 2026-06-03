---
id: e17s14
title: Functions list and detail
status: done
wsjf: 3.0
tasks:
  - desc: Refactor FunctionsPage to card grid (name, runtime, trigger, created, CTA)
    verify: "grep -q function-card ui/src/pages/FunctionsPage.tsx ui/src/index.css"
  - desc: Add FunctionDetailPage at /functions/:id with tabs code triggers variables logs
    verify: "test -f ui/src/pages/FunctionDetailPage.tsx && grep -q 'functions/:id' ui/src/App.tsx"
  - desc: Redirect or embed /functions/:id/logs into detail logs tab
    verify: "grep -q functions ui/src/App.tsx"
  - desc: Functions page tests or build
    verify: "cd ui && npm run build"

acceptance: |
  Given functions exist in the API
  When the user opens /functions
  Then each function appears as a card with runtime and trigger
  When the user opens /functions/:id
  Then tabbed detail shows code, triggers, variables, and logs
  And /functions/:id/logs still resolves (redirect or same tab)

context: |
  Prototype: BigBase Console.html functions list and functions/:id detail.
  Reuse FunctionLogsPage stream UI inside logs tab where possible.
---
