# e51s06: Extended Component Library

**Story ID:** e51s06 | **Epic:** e51 | **BCPs:** 5 | **Status:** planned
**Type:** feat | **Context:** domain
**Depends on:** e51s02 (Core Components) | **Blocks:** e57, e59, e61, e62, e64

## §1 — Summary

Build the second wave of shared components needed by upcoming epics (Project
Scoping UI, Secrets Management, Usage Dashboard, Schema Designer, Multi-User
Platform). Components include: `Tooltip` (hover info), `DropdownMenu` (context
menus and action menus), `Dialog` (confirm dialogs distinct from Modal),
`FileUpload` (drag-and-drop file input), `CopyButton` (one-click clipboard
copy), `TagInput` (multi-value tag picker), and `Alert` (inline notification
banners). All accessible, all tested, all exported from the component index.

## §2 — Motivation

Upcoming epics (e52-e64) will build new admin UI pages for project scoping,
secrets, usage dashboards, schema designer, and multi-user management. These
pages need interactive components that don't exist yet. Building them now in
one cohesive pass ensures consistency and avoids duplicated implementations.

## §3 — Background / Context

- Existing components provide: buttons, cards, modals, badges, inputs, tabs (e51s02 adds Checkbox, Spinner, Switch, Select, Label, Table)
- `Modal` exists but a lighter `Dialog` (confirm/cancel pattern) is needed for delete confirmations
- `Input` component handles text/textarea/select but not file uploads
- No tooltip or dropdown menu exists — pages like SettingsPage would benefit from dropdown menus for "..." action buttons
- No clipboard utility exists — API keys and deploy URLs need copy buttons
- No tag input exists — schema designer and env var editor would use this

## §4 — Zoom-Out Check

- **Module purpose**: Admin console React component library
- **Callers**: All page components under `ui/src/pages/`
- **Contracts**: Components accept standard React patterns, exported from `components/index.ts`, tested with vitest + testing-library

## §5 — Prior Art

- Radix UI: `@radix-ui/react-tooltip`, `@radix-ui/react-dropdown-menu`, `@radix-ui/react-alert-dialog`
- shadcn/ui: Tooltip, DropdownMenu, AlertDialog, Command (command palette)
- Decision: Build our own thin wrappers over native HTML + CSS (avoid Radix dependency; matches existing pattern of zero UI deps beyond React + react-router-dom)

## §6 — Design Decisions

| Decision | Rationale |
|----------|-----------|
| Tooltip: pure CSS + `title` attribute enhancement | No floating UI library; use CSS `::after` pseudo-element for positioning. Complex tooltips use `aria-describedby`. |
| DropdownMenu: custom with `<dialog>` or portal | Avoids popper.js; use portal + CSS `position: fixed` with simple placement logic |
| Dialog: distinct from Modal (no overlay scrim, lighter visual) | Confirm dialogs need different visual weight than full modals |
| FileUpload: native `<input type="file">` with drag-and-drop | Cross-browser without extra deps |
| CopyButton: `navigator.clipboard.writeText()` | Standard Web API, well-supported |
| TagInput: controlled input that tokenizes on Enter/comma | Simple state machine, no external dep |
| Alert: role="alert" with variant colors | Reuses Badge-like variant system |

## §7 — Architecture / Component Design

```
ui/src/components/
  Tooltip.tsx           ← NEW: hover tooltip with positioning (top/bottom/left/right)
  Tooltip.test.tsx      ← NEW
  DropdownMenu.tsx      ← NEW: trigger + floating menu with items
  DropdownMenu.test.tsx ← NEW
  Dialog.tsx            ← NEW: confirm/cancel dialog pattern
  Dialog.test.tsx       ← NEW
  FileUpload.tsx        ← NEW: drag-and-drop file input with preview
  FileUpload.test.tsx   ← NEW
  CopyButton.tsx        ← NEW: button that copies value to clipboard
  CopyButton.test.tsx   ← NEW
  TagInput.tsx          ← NEW: multi-value tag entry
  TagInput.test.tsx     ← NEW
  Alert.tsx             ← NEW: inline notification (success/warning/error/info)
  Alert.test.tsx        ← NEW
```

## §8 — Data Model / Types

```typescript
// Tooltip
interface TooltipProps {
  content: string
  children: ReactNode
  position?: 'top' | 'bottom' | 'left' | 'right'
  delay?: number  // ms before showing
}

// DropdownMenu
interface DropdownMenuProps {
  trigger: ReactNode
  items: {
    label: string
    onClick: () => void
    icon?: IconName
    danger?: boolean
    disabled?: boolean
    divider?: true  // renders a divider instead of an item
  }[]
  align?: 'start' | 'end'
}

// Dialog
interface DialogProps {
  open: boolean
  onClose: () => void
  title: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  onConfirm: () => void
  variant?: 'danger' | 'default'
  loading?: boolean
}

// FileUpload
interface FileUploadProps {
  accept?: string       // e.g., ".json,.yaml"
  multiple?: boolean
  maxSize?: number      // bytes
  onFiles: (files: File[]) => void
  label?: string
  hint?: string
  disabled?: boolean
}

// CopyButton
interface CopyButtonProps {
  value: string
  label?: string        // screen-reader text
  timeout?: number      // ms to show "Copied!" feedback
}

// TagInput
interface TagInputProps {
  value: string[]
  onChange: (tags: string[]) => void
  placeholder?: string
  suggestions?: string[]
  maxTags?: number
  disabled?: boolean
}

// Alert
interface AlertProps {
  variant: 'info' | 'success' | 'warning' | 'error'
  title?: string
  children: ReactNode
  dismissible?: boolean
  onDismiss?: () => void
}
```

## §9 — API / Interface Contract

All components:
- Accept `className` for composition
- Forward relevant HTML attributes
- Export from `components/index.ts`
- Named exports only

## §10 — State Management

- `DropdownMenu`: internal open/close state, focus trap when open
- `Dialog`: controlled via `open`/`onClose` props
- `FileUpload`: internal drag state (dragOver boolean)
- `CopyButton`: internal "copied" feedback timer
- `TagInput`: controlled via `value`/`onChange`
- `Alert`: controlled dismiss via `onDismiss` callback

## §11 — Error Handling

- `FileUpload`: validates file type and size before calling `onFiles`; shows error message for rejected files
- `CopyButton`: catches `clipboard.writeText()` errors and shows "Failed to copy" feedback
- `TagInput`: prevents duplicate tags, respects `maxTags` limit
- `DropdownMenu`: disabled items don't fire `onClick`
- `Dialog`: `loading` prop disables buttons during async confirm

## §12 — Testing Strategy

| Component | Tests |
|-----------|-------|
| Tooltip | renders on hover, shows on focus, hides on Escape, aria-describedby association |
| DropdownMenu | opens on trigger click, closes on Escape/outside click, keyboard navigation (Arrow keys), focus trap, fires onClick, renders dividers |
| Dialog | renders when open, fires onConfirm/onCancel, closes on Escape, focus trap, danger variant has red confirm button |
| FileUpload | renders drop zone, accepts drag, validates type/size, calls onFiles, shows error for invalid files, disabled state |
| CopyButton | copies value to clipboard, shows "Copied!" feedback, handles clipboard errors |
| TagInput | renders tags as badges, adds on Enter/comma, removes on X click/Backspace, prevents duplicates, respects maxTags, shows suggestions |
| Alert | renders variant colors, shows title/content, dismisses on close button click |

## §13 — Performance Considerations

- `Tooltip`: uses `onMouseEnter`/`onMouseLeave` — no continuous event listeners
- `DropdownMenu`: renders portal to `document.body` to avoid z-index issues
- `FileUpload`: validates file size synchronously; no async overhead
- `TagInput`: avoids re-render on every keystroke via internal input state

## §14 — Security Considerations

- `FileUpload`: validates file type server-side is handled by storage component; this is client-side convenience only
- `CopyButton`: respects user gesture requirement for `clipboard.writeText()` (only fires on click)
- `Tooltip`/`DropdownMenu`: content is React children — no HTML injection risk

## §15 — Accessibility

| Component | A11y requirements |
|-----------|------------------|
| Tooltip | `aria-describedby` linking trigger to tooltip content; shows on hover AND focus; Escape dismisses |
| DropdownMenu | `role="menu"`, `aria-label`, keyboard nav (Arrow keys, Enter, Escape), focus trap, `role="menuitem"` on items |
| Dialog | `role="alertdialog"`, `aria-labelledby`, `aria-describedby`, focus trap, Escape closes, focus returns to trigger |
| FileUpload | `role="button"` on drop zone, `aria-label`, keyboard activation (Enter/Space opens file dialog) |
| CopyButton | `aria-label` for screen readers, `aria-live="polite"` for feedback announcement |
| TagInput | `role="listbox"` for suggestions, `aria-label` on remove buttons |
| Alert | `role="alert"` for auto-announcement, `aria-labelledby` |

## §16 — Internationalization (i18n)

All labels, placeholders, and feedback text passed as props — no hardcoded English.

## §17 — Acceptance Criteria (Gherkin)

```gherkin
Scenario: Tooltip shows on hover
  Given a button wrapped in a Tooltip with content="Save changes"
  When the user hovers over the button
  Then the tooltip "Save changes" appears after the configured delay

Scenario: DropdownMenu opens and selects item
  Given a DropdownMenu with items [{label:"Edit",onClick:fn}, {label:"Delete",onClick:fn2}]
  When the user clicks the trigger and clicks "Edit"
  Then fn is called and the menu closes

Scenario: Dialog confirms and cancels
  Given a Dialog with title="Delete site?" and variant="danger"
  When the user clicks "Cancel"
  Then onClose is called
  When the user clicks "Delete"
  Then onConfirm is called

Scenario: FileUpload accepts valid files
  Given a FileUpload with accept=".json"
  When the user drops a valid JSON file
  Then onFiles is called with the file

Scenario: CopyButton copies to clipboard
  Given a CopyButton with value="secret-token-123"
  When the user clicks the button
  Then the clipboard contains "secret-token-123" and "Copied!" is displayed

Scenario: TagInput adds and removes tags
  Given a TagInput with value=["tag1"]
  When the user types "tag2" and presses Enter
  Then onChange is called with ["tag1", "tag2"]
  When the user clicks the X on "tag1"
  Then onChange is called with ["tag2"]
```

## §18 — Verification Script (Step-by-Step)

1. Run new component tests: `cd ui && npx vitest run src/components/Tooltip.test.tsx src/components/DropdownMenu.test.tsx src/components/Dialog.test.tsx src/components/FileUpload.test.tsx src/components/CopyButton.test.tsx src/components/TagInput.test.tsx src/components/Alert.test.tsx`
2. Run all tests: `cd ui && npm test`
3. Type check: `cd ui && npx tsc --noEmit`
4. Build UI: `cd ui && npm run build`
5. Build Go: `cd .. && go build ./...`

## §19 — Out of Scope

- ComboBox / autocomplete (typeahead search)
- Date picker
- Rich text editor
- Drag-and-drop list reordering
- Toast/notification system (already exists: `ToastContext`)
- Command palette (⌘K)
- Infinite scroll
- Virtualized list

## §20 — Risks

| Risk | Mitigation |
|------|-----------|
| `navigator.clipboard` not available in HTTP (only HTTPS/localhost) | Admin console runs on localhost; if deployed over HTTP, show manual copy fallback |
| DropdownMenu portal causes testing complexity | Test with `baseElement` from testing-library to find portal content |
| FileUpload drag-and-drop fragile on mobile | Native file input always available as fallback; drag-and-drop is progressive enhancement |
