# BUG-2026-06-04T100000: Reviewer must-fix findings on prototype-fidelity-parity

## Problem

**Actual:** A post-implementation code review (using the `request-review` skill with a fresh-context agent) found 3 must-fix findings in the 4-batch prototype-vs-codebase parity work on branch `prototype-fidelity-parity`.

**Expected:** All code shipped should be CONVENTIONS.md-compliant, ship a11y as specified in the plan, and match the plan's "Where used" column exactly.

## Findings

### must-fix #1: SettingsPage mock success leaks `/api/auth` to users (CONVENTIONS.md §Security violation)

**File:** `ui/src/pages/SettingsPage.tsx:67` (now at line 70-77 after fix).

**Actual:** The `ChangePasswordForm` `handleSubmit` set the form's `status` state to:

> "Password updated (mock — wire to /api/auth in production)"

This is two bugs in one:
1. Fakes a successful password change to the end user, with no persistence and no real submit handler.
2. Leaks the internal `/api/auth` HTTP path into the UI, which a) violates CONVENTIONS.md §Security ("Generic error messages to clients, full details in logs") and b) advertises the future internal API surface.

**Expected:** Stub forms must not signal success. Internal API paths must never reach user-visible text.

### must-fix #2: ThemePicker loses focus on close (a11y regression)

**File:** `ui/src/components/ThemePicker.tsx` (multiple close paths).

**Actual:** The plan (B1.5 in `specs/plans/PROTOTYPE-FIDELITY-PARITY.md`) explicitly required "focus returns to trigger" on every close path. The shipped implementation called `setOpen(false)` on option click, Esc, and outside click — but the focused element (the option button) was unmounted with the popover, so focus fell to `<body>`. Keyboard users had to tab from the top of the document to find the trigger again.

**Expected:** After any close action, the trigger button should regain focus.

**Why the 8 original ThemePicker tests didn't catch this:** they only asserted structural facts (aria-haspopup, aria-expanded, role=option count, selection state) but not focus management.

### must-fix #3: 5 of 15 planned icon names were silently substituted

**File:** `ui/src/components/Icon.tsx` (B4.3).

**Actual:** The plan (B4.3) listed 15 new icon names: `check, chevron-down, x, plus, search, trash-2, pencil, refresh-cw, arrow-right, external-link, download, upload, copy, more-horizontal, github`. The shipped implementation added 15 names, but 5 of them were silent substitutions: `arrow-left, chevron-right, eye, eye-off, circle` replaced the 5 planned names `download, upload, copy, more-horizontal, github`. The PROTOTYPE-VS-CODEBASE.md matrix then marked **G19 closed ✅** — which was inaccurate.

**Expected:** Either add the planned names (preferred) or amend the plan and matrix to record the substitution. The matrix's "closed" status must reflect reality.

## Root Cause Analysis

**Phase 1 — Reproduce:** Confirmed by reading the reviewer's full report (15 findings, 3 must-fix + 7 should-fix + 5 consider).

**Phase 2 — Isolate:**

1. The mock-success pattern came from a deliberate but over-clever decision: "show a success state so the UI feels alive." This violated CONVENTIONS.md §Security without anyone flagging it in self-review. The `audit-code` skill that ran earlier did not have an explicit "no internal-path strings in user-visible text" check.
2. The focus-loss bug was a genuine a11y oversight. The plan called it out; the implementation did not deliver. The 8 ThemePicker tests were structural, not behavioural. No test existed that opened the picker and asserted `document.activeElement === trigger` after a close.
3. The icon substitution was a planning discipline gap: when I picked 5 substitute names, I did not go back and update the plan. The matrix's G19 row stayed "open" until I claimed it closed, and I closed it without re-reading my own implementation against the plan's "Where used" column.

**Phase 3 — Verify:** Each must-fix was applied and verified by a new test (mock-leak guard, focus-return assertions, exhaustive-icon-name render).

## Resolution

**Fixed:** 2026-06-04
**Root cause confirmed:** Self-review (audit-code) checked hygiene and patterns but did not include explicit checks for (a) internal-path string leakage to user text, (b) focus management in popovers, or (c) plan-vs-implementation name set reconciliation.
**Fix applied:**
1. SettingsPage `ChangePasswordForm` now uses a neutral "Demo mode — password change is not persisted" message and a `console.info` log. The internal `/api/auth` path appears only in a code comment, never in user-visible text.
2. `ThemePicker` gained a `triggerRef`; focus is returned on every close path (option click, Esc, outside click). 3 new a11y tests added.
3. `Icon` now exports all 15 planned names (`download, upload, copy, more-horizontal, github` added). Test exhaustively renders 31 names (16 original + 15 new).

**Hardening added:**
1. New SettingsPage test: after submitting the password form, asserts `document.body.textContent` does not contain `/api/` or `.go\b`. Prevents recurrence of the leak.
2. Three new ThemePicker a11y tests: assert `document.activeElement === trigger` after option-click, Esc, and outside-click. Would have caught the bug originally.
3. Icon test now asserts the count is **exactly 31** and exhaustively renders every name in the union (16 original + 15 new), not just the new ones. Catches both regressions (missing name) and over-additions (drift from the plan).
4. WorkspaceNameForm refactored to a child component with `key={ws?.name}` so it re-mounts when the prop changes. This eliminates the "useState + useEffect to sync from prop" anti-pattern that triggered a new lint rule, and makes the data-flow obvious: the parent owns the data, the child owns the form.

**Evidence:** 222/222 UI tests pass. `npm run preflight` (Go vet + Go tests + UI build + Go test all packages) is green. Three review-fix commits on `prototype-fidelity-parity` (0063008, e243aae, 3a2c05d) plus the validate-fix hardening commit (e8f... — see `rtk git log main..HEAD`).

**Commits:**
- `fix(ui): address reviewer must-fix findings` (0063008)
- `fix(ui): address reviewer should-fix findings` (e243aae)
- `docs(epic-017): link Forge/Realtime design specs to epic shard` (3a2c05d)
- `refactor(ui): address validate-fix lint hardening` (see `rtk git log main..HEAD`)

## Lessons

1. **`audit-code` should include an "internal-path leakage" check.** Searching the diff for `/api/` and `.go\b` substrings in any rendered text (not just code) would have caught the SettingsPage bug.
2. **Popovers need a focus-return test by default.** Any component that conditionally renders UI on top of a trigger needs an `activeElement === trigger` assertion. This is a one-line test that catches a class of bugs.
3. **Plan vs. implementation reconciliation is a discrete step.** When the plan names 15 items, the diff should add exactly 15 (or, if fewer, the plan must be amended in the same commit). A "match" check between plan and diff is a small but high-leverage addition to the develop-tdd workflow.
