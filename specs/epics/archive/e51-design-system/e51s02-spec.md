# e51s02: Core Component Library

**Story ID:** e51s02 | **Epic:** e51 | **BCPs:** 4 | **Status:** planned
**Type:** feat | **Context:** domain
**Depends on:** e51s01 (design tokens TS types) | **Blocks:** e51s03, e51s04

## §1 — Summary

Convert CSS-only utility classes in `index.css` into properly typed, accessible
React components. The existing codebase has ~35 React components and extensive
CSS classes (`.checkbox-label`, `.spinner`, tables, `.loading`, `.empty-state`,
etc.) but many CSS patterns lack component wrappers — pages compose raw HTML
with CSS class strings. This story creates the missing primitives:
`Checkbox`, `Spinner`, `Switch` (toggle), `Select` (already partially covered
by `Input as="select"` — add standalone variant), `Label`, and `Table`.

## §2 — Motivation

CSS classes without component wrappers mean:
- No prop type checking (misspelled class names = silent bugs)
- No autocomplete
- Inconsistent patterns across pages
- Harder to enforce accessibility (aria attributes on raw `<table>` easily forgotten)
- Cannot add behavior (e.g., Switch controlled/uncontrolled state)

## §3 — Background / Context

- Existing components use pattern: `components/ComponentName.tsx` + `ComponentName.test.tsx`
- Existing tests use vitest + `@testing-library/react`
- All components accept `className` for composition
- Existing `Input` component handles `as="input" | "textarea" | "select"` — the `select` variant exists but isn't a standalone `Select`
- Existing `Badge` component handles status variants
- CSS classes to componentize: `.checkbox-label`, `.spinner` + `@keyframes spin`, `table` + `th` + `td`, `.loading`

## §4 — Zoom-Out Check

- **Module purpose**: Admin console React component library
- **Callers**: All page components under `ui/src/pages/`, Layout, other components
- **Contracts**: Components accept standard React HTMLAttributes + variant props; no breaking changes to existing components

## §5 — Prior Art

- Radix UI primitives pattern (unstyled, accessible, composable) — we follow a lighter version
- Existing `Button`, `Card`, `Modal`, `Badge`, `Input` components set the pattern

## §6 — Design Decisions

| Decision | Rationale |
|----------|-----------|
| Native `<input type="checkbox">` for Checkbox | Zero deps, full browser accessibility |
| Native `<table>` wrapper for Table | Avoid heavy data-grid lib; admin tables are simple |
| CSS `@keyframes` spinner as component | Reusable loading indicator; no SVG dependency |
| Switch as `<button role="switch">` | Better keyboard support than checkbox-styled-as-switch |

## §7 — Architecture / Component Design

```
ui/src/components/
  Checkbox.tsx          ← NEW: controlled/uncontrolled checkbox with label
  Checkbox.test.tsx     ← NEW
  Spinner.tsx           ← NEW: size variants (sm, md, lg)
  Spinner.test.tsx      ← NEW
  Switch.tsx            ← NEW: toggle switch (role="switch")
  Switch.test.tsx       ← NEW
  Select.tsx            ← NEW: standalone Select with options prop
  Select.test.tsx       ← NEW
  Label.tsx             ← NEW: form label with optional required indicator
  Label.test.tsx        ← NEW
  Table.tsx             ← NEW: Table, TableHead, TableBody, TableRow, TableCell
  Table.test.tsx        ← NEW
```

## §8 — Data Model / Types

```typescript
// Checkbox
interface CheckboxProps {
  label: string
  checked?: boolean
  defaultChecked?: boolean
  onChange?: (checked: boolean) => void
  disabled?: boolean
  className?: string
}

// Switch (toggle)
interface SwitchProps {
  label: string
  checked?: boolean
  defaultChecked?: boolean
  onChange?: (checked: boolean) => void
  disabled?: boolean
  size?: 'sm' | 'md'
}

// Select
interface SelectProps {
  label?: string
  options: { value: string; label: string }[]
  value?: string
  defaultValue?: string
  onChange?: (value: string) => void
  placeholder?: string
  error?: string
  hint?: string
  disabled?: boolean
}

// Spinner
interface SpinnerProps {
  size?: 'sm' | 'md' | 'lg'
  label?: string  // screen-reader text
}

// Table
interface TableProps { children: ReactNode; className?: string }
// + TableHead, TableBody, TableRow, TableCell (with header/align variants)
```

## §9 — API / Interface Contract

All new components follow the existing conventions:
- Accept `className`
- Spread remaining native HTML attributes
- Export from `ui/src/components/index.ts`
- Named exports only (no default exports)

## §10 — State Management

Checkbox and Switch support both controlled (`checked` + `onChange`) and
uncontrolled (`defaultChecked`) modes — standard React form pattern.

## §11 — Error Handling

- `Select` displays `error` prop as `.input-error-text` below the field
- All form components forward `disabled` state with `opacity: 0.5; cursor: not-allowed`

## §12 — Testing Strategy

| Component | Tests |
|-----------|-------|
| Checkbox | renders label, handles click toggles checked, disabled state, controlled + uncontrolled |
| Switch | renders label, toggles aria-checked, keyboard activation (Space/Enter), disabled |
| Select | renders options, onChange fires, error display, disabled |
| Spinner | renders with size classes, has aria-label for screen readers |
| Label | renders text, shows required indicator, associates with input via htmlFor |
| Table | renders semantic structure, accessibility roles correct |

## §13 — Performance Considerations

All components are pure presentational with no effects or external data — zero overhead.

## §14 — Security Considerations

No XSS surface — all content rendered via React children.

## §15 — Accessibility

| Component | A11y requirements |
|-----------|------------------|
| Checkbox | `<input type="checkbox">` with associated `<label>`, focus-visible ring |
| Switch | `role="switch"`, `aria-checked`, keyboard toggle (Enter/Space) |
| Select | Native `<select>` is fully accessible by default |
| Spinner | `role="status"`, `aria-label` for screen readers |
| Table | Use `<thead>`, `<tbody>`, `<th scope="col">` for header cells |

## §16 — Internationalization (i18n)

All user-facing text passed as props — no hardcoded strings beyond default labels.

## §17 — Acceptance Criteria (Gherkin)

```gherkin
Scenario: Checkbox renders and toggles
  Given a Checkbox with label "Accept terms"
  When the user clicks the label
  Then the checkbox becomes checked and onChange fires with true

Scenario: Switch toggles with keyboard
  Given a Switch with label "Dark mode"
  When the user presses Space on the Switch
  Then aria-checked toggles and onChange fires

Scenario: Select displays options
  Given a Select with options [{value:"a",label:"Option A"}, {value:"b",label:"Option B"}]
  When rendered
  Then both options appear in the dropdown

Scenario: Spinner is accessible
  Given a Spinner with label "Loading data"
  When rendered
  Then it has role="status" and aria-label="Loading data"
```

## §18 — Verification Script (Step-by-Step)

1. Run new component tests: `cd ui && npx vitest run src/components/Checkbox.test.tsx src/components/Spinner.test.tsx src/components/Switch.test.tsx src/components/Select.test.tsx src/components/Label.test.tsx src/components/Table.test.tsx`
2. Run all tests: `cd ui && npm test`
3. Verify type check: `cd ui && npx tsc --noEmit`
4. Build UI: `cd ui && npm run build`
5. Build Go binary: `cd .. && go build ./...`

## §19 — Out of Scope

- ComboBox / autocomplete (e51s06)
- Data grid / virtualized table
- File upload component
- Form library integration (react-hook-form, formik)

## §20 — Risks

| Risk | Mitigation |
|------|-----------|
| Table component too opinionated | Keep it thin — just semantic structure + CSS classes |
| Switch vs Checkbox confusion | Switch is for immediate effects (settings toggles); Checkbox for form submission |
| Breaking existing CSS classes | New components use new class names where needed; existing classes preserved for backward compat |
