# e87s06 — Help text + error prevention (3.3.5/3.3.6)

**type:** feat
**risk:** P1
**context:** domain
**BCPs:** 4

## Summary

AAA form criteria: 3.3.5 (help available for inputs — context-sensitive or link), 3.3.6 (error prevention for ALL user inputs — reversible, checked, or confirmed). The `Input` component has a `hint` prop but it's underused; destructive actions confirm via Modal, but data-entry forms (deploy, messaging, env vars) have no undo/review step.

## Requirements

#### ADDED: Context-sensitive help for inputs (3.3.5)
Every form input that needs explanation has a `hint` (visible or via `aria-describedby` — already wired in T000015) or a help link; help is reachable from the field.

#### ADDED: Error prevention for all submissions (3.3.6)
Irreversible or complex submissions (deploy, messaging sends, env-var edits, storage deletes) are reversible (undo) or confirmed (review step) before execution. At minimum: destructive sends/deletes confirm; multi-step forms (CreateSite wizard) allow review before final submit.

## Implementation Steps

1. Audit every `<Input>`/`<Select>` usage for missing `hint`: produce a list of fields where the label alone is ambiguous (deploy env vars, messaging templates, monitoring alert form, function runtime options). → verify: `grep -rn "placeholder=" ui/src/pages | wc -l` (record field list in spec)
2. Add `hint` (or help link) to the audited fields; hints render via the existing `input-hint` pattern with `aria-describedby` (T000015 already wires the ID). → verify: `cd ui && npx vitest run src/components/Input.test.tsx src/components/Select.test.tsx`
3. Storage delete + site delete + messaging send: add a confirm/review step (reuse `Dialog` with a summary of what will happen; destructive variants already have `danger` styling). → verify: `cd ui && npx vitest run src/pages/StoragePage.test.tsx src/pages/MessagingPage.test.tsx`
4. CreateSite wizard: add a review step before final deploy (summary of site config) OR an immediate undo surface after deploy. → verify: `cd ui && npx vitest run src/pages/CreateSitePage.test.tsx`
5. Undo where confirmation is not appropriate (env-var key/value edits in EnvVarEditor): add a "Revert" action per row while editing. → verify: `cd ui && npx vitest run src/components/EnvVarEditor.test.tsx`

## Risks

- Confirmation fatigue: too many dialogs degrade UX. Prefer reversible (undo) over confirm for row-level edits; reserve confirm for true irreversibility (deletes, sends).
- 3.3.6 is "all inputs" — full coverage is large; scope to the highest-risk forms first and record partial coverage honestly.

## Acceptance Criteria

- [ ] Help (hint/aria-describedby) on all ambiguous fields
- [ ] Deletes and sends confirm or support undo
- [ ] Wizard forms have a review step
- [ ] EnvVarEditor row-level revert
