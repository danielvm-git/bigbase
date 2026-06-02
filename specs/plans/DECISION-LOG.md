# Decision Log

Lightweight decisions (full ADRs in `specs/adr/`).

| Date | Decision | Rationale | Alternatives |
|------|----------|-----------|--------------|
| 2026-06-01 | Infrastructure-first execution order for Epics 017-023 | Multi-DB unlocks production-readiness; testing woven throughout | Strict dependency-only ordering |
| 2026-06-01 | 7 vertical-slice epics with no overlap | Each epic targets a distinct concern | Monolithic v2 release |
| 2026-06-01 | Reorder to UI-first — Epic 017 = Enhanced Admin UI | Independent, fully spec'd, immediate visible value | Multi-DB first |
| 2026-06-01 | Console UI design docs under `specs/epics/e17-enhanced-admin-ui/` | SYSTEM_DESIGN, COMPONENT_INVENTORY, IMPLEMENTATION_GUIDE | Inline in RELEASE-PLAN only |
| 2026-06-01 | Deleted conflicting `RELEASE_PLAN.md` | Assumed greenfield; v1.0 already shipped | Keep duplicate plan |
| 2026-06-02 | YAML-first specs layout (bigpowers) | Agent skills use state.yaml + epic shards | Keep flat markdown SoT |
