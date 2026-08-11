# e87s04 — Abbreviations, unusual words, reading level (3.1.3/3.1.4/3.1.5)

**type:** feat
**risk:** P2
**context:** domain
**BCPs:** 3

## Summary

AAA language criteria: 3.1.3 (unusual words have a mechanism for definitions), 3.1.4 (abbreviations expandable/explainable), 3.1.5 (content reads below lower-secondary level, or a mechanism like a glossary/plain-language version). The admin console shows unexpanded jargon: "Goroutines", "Heap MB" (Dashboard), "BaaS", "MCP", "SSE", "Repo", "Cici", "ACME". Add `<abbr>` expansions + a glossary/definition mechanism.

## Context

Jargon inventory (from pages): Dashboard "Goroutines", "Heap MB", "Uptime"; Monitoring "SSE", "avg latency"; MCP "Model Context Protocol"; docs/landing "BaaS"; sites "ACME SSL", "repo". The `Tooltip` component (aria-describedby) is the existing mechanism for definition-on-demand. Reading level 3.1.5: technical UI text is necessarily above lower-secondary — the acceptable mechanism is an expandable glossary (or documenting the exception).

## Canonical expansion inventory (t1)

Recorded 2026-08-11 from `grep -rn 'Goroutines\|MCP\|SSE\|BaaS\|ACME' ui/src/pages ui/src/components` (8 matches, all code identifiers/comments in MonitoringPage) plus a full term sweep of `ui/src/pages/*` and `ui/src/components/*`:

| Term | Expansion | Where it appears as visible text | Mechanism |
| --- | --- | --- | --- |
| Goroutines | Concurrent lightweight tasks managed by the Go runtime | MonitoringPage stat label | `<abbr title>` + tooltip definition (Abbr) |
| Heap MB | Heap memory in megabytes | MonitoringPage stat label | `<abbr title>` |
| Avg Latency | Average latency | MonitoringPage stat label + requests table header | `<abbr title>` |
| Uptime | Time elapsed since the server last started | MonitoringPage stat label, SystemStatusPanel sub-line | `<abbr title>` + tooltip definition |
| CPU | Central processing unit | MonitoringPage host tab, SystemStatusPanel metric label | `<abbr title>` |
| RAM | Random-access memory | MonitoringPage host tab (BarGauge label prop — outside contract, glossary only) | glossary |
| process heap | The memory region where the server process makes dynamic allocations | SystemStatusPanel dim label | tooltip definition (Abbr) |
| SQL | Structured Query Language | DashboardPage jump link "Write a SQL query" | `<abbr title>` |
| Git Repos | Git repositories | DashboardPage stat card label | `<abbr title>` |
| SSE | Server-Sent Events | not user-visible (code identifiers/comments only) | glossary |
| MCP | Model Context Protocol | not user-visible in UI | glossary |
| BaaS | Backend-as-a-Service | docs/landing only, not UI | glossary |
| ACME | Automated Certificate Management Environment | sample site data only, not static UI text | glossary |
| repo | repository | data-driven (site/repo names); DashboardPage "Git Repos" annotated | `<abbr title>` + glossary |

SiteCard contains no literal acronyms (ACME/repo are data values, not static text) — left unannotated; terms covered by glossary.

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
