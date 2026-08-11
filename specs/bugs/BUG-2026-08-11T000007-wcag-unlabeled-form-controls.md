# Systemic: form controls without accessible names (placeholder-only)

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** SERIOUS
**WCAG:** 3.3.2 Labels or Instructions / 4.1.2 Name, Role, Value — Level A

## Description

The `Input` component only emits `<label htmlFor>` when a `label` prop is passed; many call sites use placeholder-only inputs → no accessible name. Screen readers announce bare "edit text". Affects 15+ files:

- `LoginPage.tsx` — email, password, reset-email
- `SqlEditorPage.tsx` — query textarea
- `DataStudioPage.tsx` — filter, sort
- `StoragePage.tsx` — file upload input
- `FunctionsPage.tsx` — name, runtime, trigger, cron, env, timeout, source
- `MessagingPage.tsx` — all email/sms/push test fields
- `ForgePage.tsx` — repo select, issue fields, status select
- `CiciPage.tsx` — repo select, workflow fields
- `MonitoringPage.tsx` — log search, alert form fields
- `StreamLog.tsx`, `RequestLogs.tsx` — filter inputs/selects
- `SiteDeployKeysTab.tsx` — key name (label not htmlFor-linked)
- `SiteEnvVarsTab.tsx` — key, value (labels not linked)
- `SiteDomainsTab.tsx` — domain input

## Recommended Fix
Pass `label="..."` (or `aria-label="..."`) to every `<Input>`/`<select>`/`<textarea>`; add a `.sr-only` utility class for visually-hidden labels where needed; link existing labels via `htmlFor`/`id`.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
