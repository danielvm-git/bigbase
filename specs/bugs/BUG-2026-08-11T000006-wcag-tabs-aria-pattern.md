# Tabs component missing ARIA tab pattern and keyboard navigation

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** SERIOUS
**WCAG:** 4.1.2 Name, Role, Value / 2.1.1 Keyboard — Level A

## Description

`Tabs.tsx` renders plain `<button>`s with no `role="tablist"`/`role="tab"`/`aria-selected`, and consuming panel components render in bare `<div>` roots with no `role="tabpanel"`, no `id`↔`aria-controls` linkage, and no Arrow-key tab navigation. The entire WAI-ARIA tablist pattern is absent. Affects every page using tabs: SiteDetailPage, MessagingPage, CiciPage, MonitoringPage, ForgePage, DataStudioPage.

## Affected Files
- `ui/src/components/Tabs.tsx`
- `ui/src/components/SiteDeployKeysTab.tsx`, `SiteCacheTab.tsx`, `SiteDomainsTab.tsx`, `SiteEnvVarsTab.tsx`
- `ui/src/pages/SiteDetailPage.tsx` (tab panel composition)

## Recommended Fix
Add `role="tablist"` to the container, `role="tab"` + `aria-selected` + `id` + `aria-controls` to each tab, `role="tabpanel"` + `aria-labelledby` to each panel, and Arrow-Left/Right/Home/End keyboard activation with roving tabindex.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
