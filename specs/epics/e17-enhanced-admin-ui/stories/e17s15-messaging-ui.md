---
id: e17s15
title: Messaging templates
status: done
wsjf: 2.6
tasks:
  - desc: Template list view with type, date, status, Create template CTA
    verify: "grep -qi template ui/src/pages/MessagingPage.tsx"
  - desc: MessagingDetailPage at /messaging/:id with editor and preview
    verify: "test -f ui/src/pages/MessagingDetailPage.tsx && grep -q 'messaging/:id' ui/src/App.tsx"
  - desc: Preserve send-test or outbound log behind secondary action if API differs
    verify: "cd ui && npm run build"
  - desc: Route registration and build
    verify: "cd ui && npm run build && grep messaging ui/src/App.tsx"

acceptance: |
  Given messaging templates (API or preview mock)
  When the user opens /messaging
  Then a table or card list shows templates with status
  When the user opens /messaging/:id
  Then subject, body, variables, and preview panel are editable
  And user can navigate back to the list

context: |
  Prototype models transactional templates; current MessagingPage is outbound send + log.
  Use previewMode or mock data until template APIs exist.
---
