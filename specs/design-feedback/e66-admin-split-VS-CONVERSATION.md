# Admin console split — gap map vs. prototype & roadmap

**Date:** 2026-07-12
**Trigger:** conversation proposing a Repositories-vs-Platform login split, with Platform further split into an instance-admin lens and a project-workspace lens.
**Method:** compared that conversation's decisions against every live design artifact — not just the codebase.

## Sources compared

| Source | What it is | State |
|---|---|---|
| This conversation | Repositories vs Platform doors; Platform = instance-admin lens vs project-workspace lens via a combined switcher; `is_superadmin` flag; solo-today/multi-tenant-later | Wireframes only, not yet written to specs/ until now |
| `ui/src/` (codebase) | Shipped admin UI | Flat single sidebar (Overview/Build/Data/Auth/Engage/DevOps), one `LoginPage`, routes still `/deploy/*` |
| claude.ai/design `ec1480a1` "BigBase Prototype" (`BigBase Console.html`, `bb/shell.jsx`) | **Live, canonical prototype** — `design-sync.config.json` points here | Confirmed pixel-aligned to codebase per `specs/design/GAP-ANALYSIS-v3-codebase.md` (2026-06-30): same flat sidebar, same single login, no project zone, no second door |
| claude.ai/design `502492b2` "BigBase Design System" (`SYSTEM_DESIGN.md`, `UPDATE-BRIEF-v3-design-c.md`) | Token/component reference + an earlier "Design C" IA exploration | Explicitly **not** the prototype target anymore (superseded by `ec1480a1`); its two-zone sidebar was never wired into `bb/shell.jsx` |
| `specs/epics/e57-58-59-60-66` + `specs/design/PROTOTYPE-UPDATE-PLAN.md` | Active roadmap | e58 is next up per `state.yaml`; e66 (closest epic to "instance admin") has **zero real spec** and isn't even listed in the Epic→Prototype Impact Matrix |
| `specs/IMPACT_LATEST.md`, `specs/PLAN-AUDIT.md` | Known architectural debt | Issue #43 (Policy Gate) already flagged as blocking e66, with a recommended action not yet applied |

## Gap map

### 1. No epic owns the Repositories/Platform door split
Nothing in e08 (Forge), e57–e66, or either claude.ai/design project treats git hosting (Git Repos/Forge/CI-CD) as a separate product with its own login. The live prototype's `bb/shell.jsx` keeps them as one "DevOps" sidebar section inside the single console. This is a genuinely new IA decision — not a refinement of anything planned.

### 2. "Project" already means something else
e57/e58 are building a `projects` table — a Neon-style **database** project with branches, owned inside an `org`. The conversation's "project workspace" lens means the **tenant/org** scope (Dashboard/Data/Storage/Messaging/Users/Settings for one customer). Same word, two different resources. If this isn't renamed before e58 ships UI copy, the sidebar will end up with two unrelated things both called "project."

### 3. e66 is the right epic, but it's an unspecced, deferred stub
`e66s01` (Admin User Management Dashboard) and `e66s03` (Role Switching & Workspaces) are the closest existing epic stories to "instance-admin lens" and "combined switcher" — but both are placeholder Gherkin (`> Full spec deferred to plan-work`). `epic.yaml` already notes the real blocker: **Issue #43** (PolicyEnforcer interface in auth/proxy) must land first, and `specs/IMPACT_LATEST.md` recommends deferring the build to Band 3 — a recommendation nobody has acted on yet. e66 is also absent from `PROTOTYPE-UPDATE-PLAN.md`'s epic list entirely, so it isn't even in the prototype rollout sequence.

### 4. No two-door login exists anywhere
`SYSTEM_DESIGN.md` Journey C is single login → dashboard, everywhere, in both claude.ai/design projects and in `ui/src/pages/LoginPage.tsx`. The two-door chooser and per-lens sign-in from this conversation has no prototype, brief, or code precedent.

### 5. Global-vs-scoped disagreement on Data/Storage/Messaging/Users
The Design C exploration (never wired into the live prototype, but still the only written IA doc for project scoping) keeps Data Studio, Storage, Messaging, and Users in a **Global Zone** — not project- or org-scoped. This conversation scoped those same pages to the project-workspace lens. Since e58 (project-scoping UI) is next up per `state.yaml`, this disagreement needs resolving before that epic writes sidebar code, or it'll get built once against Design C's model and rebuilt again for ours.

### 6. Route drift is already tracked, but will collide
`/deploy` → `/sites` is already epic e60, blocked on e57, "ships in Wave 2 alongside e58." Any new routing for a Repositories/Platform split (e.g. `/repos/*`, `/admin/*`, `/app/:org/*`) needs to land after e60, not race it.

## Closure plan, in order

1. **Naming decision (no code, do first).** Keep `org` = tenant/customer (already shipped in `components/auth`), keep `project` = Neon-style DB project (e57/e58). Rename this conversation's "project workspace" lens to **"org workspace"** everywhere to stop the collision. Update the wireframe artifact language to match.
2. **Resolve Issue #43 scope before writing real e66 specs.** Per `specs/IMPACT_LATEST.md`'s own recommendation: add the PolicyGate prerequisite note to `e66/epic.yaml` formally, and decide whether e66s01/s03 need the full PolicyEnforcer or can ship against a minimal `users.is_superadmin` boolean first. Given the solo-user reality from this conversation, the flag-first path is the lower-risk one — but that's a call for `plan-work`, not something to silently assume.
3. **Write real specs for e66s01 and e66s03**, replacing their Gherkin placeholders with this conversation's decisions: `is_superadmin` flag, combined switcher (This server / org rows), solo-today-grows-into-multi-tenant-later.
4. **Open a new epic for the Repositories/Platform door split.** No existing epic covers it. Needs `scope-work` → `slice-tasks`, sequenced after e60 (route rename) and likely alongside e58 (both touch `Layout.tsx` and top-level routing — flag the overlap explicitly when scoping).
5. **Resolve gap 5 (Global Zone vs org-scoped) before e58 implementation starts** — this is the most time-pressured item since e58 is already next in the queue.
6. **Only after 1–5 are decided:** write `specs/design/UPDATE-BRIEF-e<new>-admin-split.md` following the existing protocol, post it to the **live** prototype project (`ec1480a1`), not `502492b2`. That's what actually needs updating — the Design System project is reference material, not the sync target.

## Still open (carried over from the conversation, unaffected by this gap map)

Whether the Repositories door and the Platform door share one `auth` session/account or are genuinely separate identities — independent of everything above, and worth deciding before step 4.
