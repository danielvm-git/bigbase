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

## Field audit (e87s06t1)

Audited every `<Input>`/`<Select>` usage on the owned form pages (StoragePage, MessagingPage, DeployPage, CreateSitePage, FunctionsPage). Fields whose label alone is ambiguous got a `hint` (rendered via the existing `input-hint` pattern and wired with `aria-describedby` by T000015); `name`/`id` was added where missing so the hint ID resolves.

| Page | Field | Hint added | Mechanism |
|---|---|---|---|
| StoragePage | Upload file | "Files are stored on this instance and stay available until you delete them." | `aria-describedby="upload-hint"` on raw file input |
| MessagingPage | To (email) | "Recipient email address, e.g. name@example.com" | `hint` + `name="emailTo"` |
| MessagingPage | To (SMS) | "Recipient phone number in E.164 format, e.g. +15551234567" | `hint` + `name="smsTo"` |
| MessagingPage | Device token (push) | "Push token registered by the device (APNs or FCM)" | `hint` + `name="pushToken"` |
| CreateSitePage | Site name | "Used in the site URL and list, e.g. my-app." | `hint` + `name="site-name"` |
| CreateSitePage | Production branch | "Commits pushed to this branch trigger production deploys." | `hint` + `name="production-branch"` |
| CreateSitePage | Root directory | "Subdirectory of the repo that contains the app, e.g. ./ or packages/web." | `hint` + `name="root-directory"` |
| FunctionsPage | Runtime | "Runtime for the function. JavaScript is the only supported runtime." | `hint` + `name="runtime"` (select) |
| FunctionsPage | Trigger | "How the function is invoked: HTTP endpoint, schedule (cron), or event." | `hint` + `name="trigger"` (select) |
| FunctionsPage | Cron schedule | "Cron expression in UTC, e.g. */5 * * * *." | `hint` + `name="cron-schedule"` |
| FunctionsPage | Env JSON | 'Environment variables as a JSON object, e.g. {"KEY":"value"}.' | `hint` + `name="env-json"` |
| FunctionsPage | Timeout (seconds) | "Maximum execution time in seconds before the function is stopped." | `hint` + `name="fn-timeout"` |
| FunctionsPage | Source code | "The function handler; must export a default function." | `hint` + `name="fn-source"` |

Not ambiguous (no hint needed): storage Download/Refresh buttons, DeployPage search filter (has `aria-label` "Search sites"), messaging Subject/Body/Messages (self-explanatory), CreateSitePage repo pickers (self-describing rows), function Name.

Deferred (out of contract): MonitoringPage alert form — owned by w4; recorded as deferred.

Error prevention implemented:
- StoragePage file delete → `Dialog` (danger) with file name + size summary.
- DeployPage site delete → `Dialog` (danger) with site name + consequences summary.
- MessagingPage email/SMS/push sends → `Dialog` (danger) with recipient + subject/message summary before the POST.
- CreateSitePage wizard → 4-step flow (Source → Configure → Review → Deploy); Review step summarises site config (name, source, branch, root, URL) before final deploy.
- EnvVarEditor → per-row `Revert` action shown only while the row is dirty (value/key changed or row added); reverts restore the original key/value, added rows are removed. Reversible instead of confirm, per the confirmation-fatigue risk note.
- FunctionsPage delete keeps its existing native `confirm()` (already a confirmation step; not in the Dialog scope of t3).
