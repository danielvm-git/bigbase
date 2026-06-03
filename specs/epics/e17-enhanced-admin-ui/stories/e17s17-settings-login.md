---
id: e17s17
title: Settings and login polish
status: done
wsjf: 2.2
tasks:
  - desc: Add SettingsPage with Account Workspace Billing tabs (stubs OK)
    verify: "test -f ui/src/pages/SettingsPage.tsx"
  - desc: Register /settings route in App.tsx
    verify: "grep settings ui/src/App.tsx"
  - desc: Login forgot-password UI and field validation states
    verify: "grep -i forgot ui/src/pages/LoginPage.tsx"
  - desc: Build and login-related tests if present
    verify: "cd ui && npm run build"

acceptance: |
  Given an authenticated user
  When they open /settings
  Then Account, Workspace, and Billing tabs render with stub content
  Given the login page
  When the user clicks Forgot password
  Then a reset flow UI appears (API stub acceptable)
  And invalid fields show inline error styling

context: |
  Prototype Settings footer route; Login centered card with Google and reset flow.
  Workspace members distinct from Auth Users per IA cross-link note.
---
