# Threat Model — e51: UI Design System & Component Library

**Risk Level: LOW**
**Date: 2026-06-28**

## Surface Area

Pure frontend TypeScript/React component library. No backend routes, no database access, no auth logic. All components render in the browser, no server-side rendering.

## Threat Categories

### T1 — XSS via unsanitized content (Medium risk → mitigated)
**Components:** `CodeBlock`, `JsonTree`, `Dialog`
**Attack:** User-controlled strings rendered as HTML via `dangerouslySetInnerHTML` or `innerHTML`.
**Mitigation:** Never use `dangerouslySetInnerHTML`. All text content rendered via React text nodes only. `CodeBlock` displays code as plain text in `<pre><code>`. `JsonTree` renders JSON values as escaped strings.

### T2 — Clipboard hijacking via CopyButton (Low risk → mitigated)
**Component:** `CopyButton`
**Attack:** Clipboard write intercepts sensitive data or overwrites clipboard with malicious content.
**Mitigation:** `navigator.clipboard.writeText()` is used only with the prop value explicitly passed by the parent. No dynamic value resolution. Browser permission model applies.

### T3 — CSS injection via token values (Low risk → informational)
**Module:** `tokens.ts`
**Attack:** If token values were user-supplied and injected into `style` attributes, CSS injection could alter layout.
**Mitigation:** All token values in `tokens.ts` are compile-time constants mirroring `tokens.css`. No runtime user input feeds into token values.

### T4 — File upload abuse (Low risk → mitigated)
**Component:** `FileUpload`
**Attack:** Malicious file type bypasses client-side type validation.
**Mitigation:** Client-side type/size validation is defense-in-depth only. Server must enforce file type and size independently. Component validates MIME type and extension; never executes uploaded content.

### T5 — Focus trap escape in Dialog/DropdownMenu (Low risk → mitigated)
**Components:** `Dialog`, `DropdownMenu`
**Attack:** Keyboard users can navigate outside modal, accessing content they shouldn't.
**Mitigation:** Implement focus trap using `inert` attribute or manual focus cycling. Test with keyboard navigation in axe audit (e51s05).

### T6 — Accessibility as attack surface (Informational)
`aria-*` attribute injection via props: all `aria-*` props are typed strings from TypeScript callers, not user input. No escalation risk.

## Residual Risk

All identified risks are LOW and fully mitigated by implementation choices. No architectural changes required. No `grill-me` session needed (risk score: 3/10).

## Mitigations to Enforce in Code Review

1. Zero `dangerouslySetInnerHTML` in any new component.
2. `CopyButton` must not accept a callback that returns dynamic values from fetch.
3. `FileUpload` must document that server-side validation is mandatory.
4. Focus traps in `Dialog` and `DropdownMenu` must be tested in e51s05 axe suite.
