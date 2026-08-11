# e87s04 — Abbreviations, unusual words, reading level (3.1.3/3.1.4/3.1.5)

**type:** feat
**risk:** P2
**context:** domain
**BCPs:** 3

## Summary

AAA language criteria: 3.1.3 (unusual words have a mechanism for definitions), 3.1.4 (abbreviations expandable/explainable), 3.1.5 (content reads below lower-secondary level, or a mechanism like a glossary/plain-language version). The admin console shows unexpanded jargon: "Goroutines", "Heap MB" (Dashboard), "BaaS", "MCP", "SSE", "Repo", "Cici", "ACME". Add `<abbr>` expansions + a glossary/definition mechanism.

## Context

Jargon inventory (from pages): Dashboard "Goroutines", "Heap MB", "Uptime"; Monitoring "SSE", "avg latency"; MCP "Model Context Protocol"; docs/landing "BaaS"; sites "ACME SSL", "repo". The `Tooltip` component (aria-describedby) is the existing mechanism for definition-on-demand. Reading level 3.1.5: technical UI text is necessarily above lower-secondary — the acceptable mechanism is an expandable glossary (or documenting the exception).

## Requirements

#### ADDED: Abbreviations expandable (3.1.4)
Every abbreviation/acronym in UI text is wrapped in `<abbr title="expansion">` or linked to a definition. Jargon terms get a definition mechanism (tooltip or glossary link).

#### ADDED: Unusual words definable (3.1.3) + reading-level mechanism (3.1.5)
Jargon terms ("Goroutines", "Cici", "Forge") link to a definitions mechanism; a `/glossary` surface or in-place tooltips satisfy 3.1.3/3.1.5's "mechanism available".

## Implementation Steps

1. Inventory all acronyms/jargon across `ui/src/pages/*` + `ui/src/components/*` (grep for the terms above); build the canonical expansion list. → verify: `grep -rn "Goroutines\|MCP\|SSE\|BaaS\|ACME" ui/src/pages ui/src/components | wc -l` (list recorded in spec)
2. Wrap abbreviations in `<abbr title>` where they appear (Dashboard stats, Monitoring labels, site cards). → verify: `grep -c "<abbr" ui/src/pages/DashboardPage.tsx ui/src/pages/MonitoringPage.tsx`
3. Add a reusable `Abbr` component (or glossary tooltip) wired through `Tooltip` for the high-jargon terms; document the pattern. → verify: `cd ui && npx vitest run src/components/Tooltip.test.tsx`
4. Add a lightweight Glossary section (README-style page or footer link) listing terms + expansions, satisfying the 3.1.5 mechanism. → verify: glossary artifact exists + linked from the app footer

## Risks

- Over-annotating adds visual noise — use `<abbr>` with `title` (no visible change) for 3.1.4; reserve tooltips/glossary for 3.1.3/3.1.5.
- "Goroutines" is a runtime detail — consider renaming to "Routines/Go workers" with expansion, or documenting.

## Acceptance Criteria

- [ ] Every acronym has an expansion (abbr or glossary)
- [ ] Jargon terms reachable from a definition mechanism
- [ ] Glossary artifact linked from the UI
