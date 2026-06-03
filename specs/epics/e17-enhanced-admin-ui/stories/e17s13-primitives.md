---
id: e17s13
title: Primitive and state parity
status: done
implemented_in: dff889f
wsjf: 2.8
tasks:
  - desc: Add ghost variant to Button.tsx and .btn-ghost styles
    verify: "grep -q ghost ui/src/components/Button.tsx && grep -q btn-ghost ui/src/index.css"
  - desc: Add Modal component with focus trap and export from components index
    verify: "test -f ui/src/components/Modal.tsx && grep -q Modal ui/src/components/index.ts"
  - desc: Add Breadcrumb component and export
    verify: "test -f ui/src/components/Breadcrumb.tsx && grep -q Breadcrumb ui/src/components/index.ts"
  - desc: Focus-visible ring on .btn and .sidebar-nav a per States spec
    verify: "grep -q focus-visible ui/src/index.css"
  - desc: tokens.css has at least 30 var(-- references
    verify: "grep -c 'var(--' ui/src/styles/tokens.css | awk '{if($1>=30) print \"PASS\"; else print \"FAIL\"}'"
  - desc: Component tests pass
    verify: "cd ui && npm test"

acceptance: |
  Given a page using Button ghost variant
  When hovered and focused via keyboard
  Then styles match secondary/ghost tokens from prototype States spec
  Given Modal open
  When user presses Escape
  Then modal closes and focus returns

context: |
  Inventory gaps from COMPONENT_INVENTORY.md (Modal, Breadcrumb, ghost button).
  DropdownMenu deferred unless a screen in e17s14–s17 requires it.
---
