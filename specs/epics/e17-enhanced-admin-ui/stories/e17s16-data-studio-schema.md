---
id: e17s16
title: Data Studio schema mode
status: pending
wsjf: 2.5
tasks:
  - desc: Add Data/Schema toggle on DataStudioPage
    verify: "grep -i schema ui/src/pages/DataStudioPage.tsx"
  - desc: Schema view lists columns with Add Edit Delete actions
    verify: "grep -E 'Add|Edit|Delete' ui/src/pages/DataStudioPage.tsx"
  - desc: Query this link navigates to SQL Editor with collection hint
    verify: "grep -q sql ui/src/pages/DataStudioPage.tsx"
  - desc: Build passes
    verify: "cd ui && npm run build"

acceptance: |
  Given a selected collection
  When the user switches to Schema mode
  Then column names and types are listed
  When Add column is clicked
  Then UI prompts for name/type (API wired or toast stub if DDL unsupported)
  When Query this is clicked
  Then user lands on /sql with collection context in query or state

context: |
  Prototype Data Studio Data/Schema toggle and column ops.
  Backend column DDL may be stubbed with toast per previewMode pattern.
---
