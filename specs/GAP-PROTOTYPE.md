# Prototype ↔ codebase gap matrix

Maps [bigbase-prototype](archive/assets/bigbase-prototype/) handoff assets to epic **e17** stories (prototype parity extension).

| Prototype asset | Story | Notes |
|-----------------|-------|-------|
| `project/Component Spec - Information Architecture.html` | e17s11 | Sidebar groups, route labels, cross-links (e17s18) |
| `project/Component Spec - States.html` | e17s12, e17s13 | Accent themes, component states, ghost button |
| `project/Component Spec - Responsive.html` | e17s18 | 1024 / 768 / 375 breakpoints |
| `project/BigBase Console.html` | e17s14–e17s17 | Screen-level UX |
| `archive/prompts/PROMPT_01_UPDATE_DESIGN_SYSTEM.md` | e17s12–e17s18 | Themes, secondary screens, a11y |
| `project/screenshots/01-01-dashboard.png` | e17s05, e17s09, e17s18 | Dashboard hub |
| `project/screenshots/02-01-dashboard.png` | e17s05, e17s09 | Alternate dashboard state |
| `project/screenshots/01-05-final.png`, `02-05-final.png`, `03-05-final.png` | e17s11 | Console shell / nav |
| `project/screenshots/01-06-fndetail.png`, `02-06-fndetail.png` | e17s14 | Function detail |
| `project/screenshots/01-spec-states.png`, `02-spec-states.png` | e17s13 | Component states |
| `project/screenshots/03-navcheck.png` | e17s11 | IA verification |
| `project/screenshots/spec-resp.png` | e17s18 | Responsive spec |

## Route aliases (no URL rename)

| Prototype key | App route |
|---------------|-----------|
| `dashboard` | `/` |
| `sites`, `sites/new`, `sites/:id` | `/deploy`, `/deploy/new`, `/deploy/:siteId` |
| `functions`, `functions/:id` | `/functions`, `/functions/:id` (e17s14) |
| `data` | `/data` |
| `sql` | `/sql` |
| `storage` | `/storage` |
| `users` | `/users` |
| `messaging`, `messaging/:id` | `/messaging`, `/messaging/:id` (e17s15) |
| `repos` | `/repos` |
| `cici` | `/cici` |
| `monitoring` | `/monitoring` |
| `settings` | `/settings` (e17s17) |

## Out of prototype IA (platform extras)

- `/forge`, `/realtime` — kept under DevOps in e17s11 until product removes them.

## WSJF (story shards)

Stories e17s11–e18 include per-story `wsjf` in frontmatter for optional Mode B reorder.
