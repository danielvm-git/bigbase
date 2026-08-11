# e88s02 — Formal conformance exceptions (1.4.8 partial + 2.2.3 security)

**type:** docs
**risk:** P3
**context:** domain
**BCPs:** 2

## Summary

A strict AAA claim needs the two remaining partials recorded as formal, defensible exceptions rather than informal notes: **1.4.8 Visual Presentation** (user-selectable fg/bg colors + text-spacing controls not implemented) and **2.2.3 No Timing** (JWT session expiry claimed under the essential/security exception). Also verify the implemented 1.4.8 bullets hold.

## Context

e87s07 shipped the feasible 1.4.8 bullets (line-height ≥1.5, rem text scaling, no justified text) and documented the rest in the story spec. The session-timeout warning (e87s05) covers 2.2.6; 2.2.3 needs the exception stated. WCAG allows conformance claims with documented exceptions only where the criterion itself permits them (2.2.3: "essential" activities; 1.4.8: no exception, so the claim becomes "AAA with documented partial" — this must be stated honestly).

## Requirements

#### ADDED: Formal conformance exception record
A `specs/CONFORMANCE.md` (or equivalent) stating: implemented AAA criteria with evidence; the 1.4.8 partial (which bullets, why out — OS/browser settings cover user color/spacing needs, admin density constraint, audit trail); the 2.2.3 essential/security exception (JWT expiry is auth security, not content timing; users warned via 2.2.6); and the resulting claim wording ("WCAG 2.2 AA; AAA except 1.4.8 partial, 2.2.3 exception").

## Implementation Steps

1. Verify implemented 1.4.8 bullets on the current build: line-height ≥1.5 on body text, text scales to 200% (rem tokens), no justified text. → verify: `grep -q "line-height: 1.5" ui/src/styles/tokens.css && grep -q "0.75rem" ui/src/styles/tokens.css`
2. Write `specs/CONFORMANCE.md`: evidence table (criteria → implementation → verification), the 1.4.8 partial statement with rationale, the 2.2.3 exception statement, and the exact conformance claim. Reference the e87/e88 threat models + axe scan. → verify: `grep -q "1.4.8" specs/CONFORMANCE.md && grep -q "2.2.3" specs/CONFORMANCE.md`
3. Cross-link: reference CONFORMANCE.md from the e88 epic note + release-plan e88 note. → verify: `grep -c "CONFORMANCE" specs/epics/e88-aaa-closing/epic.yaml`

## Risks

- Overclaiming: the document must NOT claim full AAA if 1.4.8 is partial — the wording is "AA certified; AAA except [documented partials]" (WCAG conformance claims are level-atomic; a partial AAA claim is expressed as a statement, not a conformance level).

## Acceptance Criteria

- [ ] CONFORMANCE.md exists with criteria table + both exceptions + claim wording
- [ ] Implemented 1.4.8 bullets verified on current build
- [ ] Claim wording honest (AA certified; AAA except documented partials)
