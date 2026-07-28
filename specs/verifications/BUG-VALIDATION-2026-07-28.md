# Closed-Bug Certification Sweep

**Pinned baseline:** `cedb0fc51cca0436c709295c79dd251906536513`
**Branch:** `chore/bug-certification-sweep`
**Date:** 2026-07-28
**Scope:** 100 closed registry bugs (99 `fixed` + 1 `done`) + 46 closed GitHub issues

Uncommitted working-tree changes were **excluded**; every file check reads the
pinned commit via `git cat-file -e <sha>:<path>`. Raw evidence lives in
`specs/verifications/.sweep/`.

---

## 1. Arithmetic (no rows silently dropped)

| | Count |
|---|---|
| Registry entries total | 111 |
| − `wontfix` | 9 |
| − `deferred` | 2 |
| **= closed rows adjudicated** | **100** |

Of those 100, three ids appear twice (`BUG-2026-07-11T032535`,
`BUG-2026-07-11T032549`, `BUG-2026-07-24T091500`), so rows are keyed by
`(id, file)` throughout. Excluded ids are listed in §8.

## 2. Verdict distribution

| Verdict | Count | Meaning |
|---|---|---|
| **REGRESSED** | **1** | The claimed fix is demonstrably absent from HEAD |
| **PARTIAL** | **5** | Fix only partly applied, or applied but not wired in |
| SUSPECT | 0 | (all resolved by Tier-2) |
| UNPROVEN | 7 | Cannot be confirmed from repo evidence |
| CERTIFIED-WEAK | 62 | Fix confirmed present, but proof is circumstantial or leg-limited |
| CERTIFIED | 25 | Named guard exists and passes, or live behavioural proof |

**13 bugs carry live behavioural proof** from HTTP probes against a running instance.

## 3. Test evidence (Tier 0, run once)

| Gate | Result |
|---|---|
| `go build ./...` | PASS |
| `go test ./... -json` (sqlite) | **1226 pass / 0 fail / 1 skip** across 29 packages |
| `go vet ./...` | clean |
| `golangci-lint run ./...` | 2 issues, both inside `ui/node_modules/` (see §6) |
| `ui` vitest, **Node 22** | **549 pass / 0 fail** — authoritative |
| `ui` vitest, Node 26.3.0 | 543 pass / 6 fail — **environment artifact, not regressions** |
| Postgres CI leg | **NOT RUN** — no Docker/Postgres on this host |
| `packages/auth`, `packages/auth-next` vitest | NOT RUN — no `node_modules` installed |

The single Go skip is `TestEnsureNodePackageManager_MissingBun`, an intentional
negative-path test. npm 11.16.0 and bun 1.3.14 were on `PATH`, so the
`components/deploy` mass-skip failure mode did **not** occur.

### The Node 26 near-miss

On the machine's default Node (26.3.0), six UI tests fail with
`TypeError: Cannot read properties of undefined (reading 'getItem'/'clear')` —
Node 26 ships an experimental built-in `localStorage` that conflicts with jsdom.
The same suite is **549/549 green on Node 22**, matching CI (`node-version: '20'`).

Had the sweep trusted the default-Node run, it would have reported **six false
regressions**. This is exactly the failure mode the rubric was built to prevent.
The incompatibility is real, though, and is filed as a new bug.

## 4. Findings that matter

### 4.1 REGRESSED — fix was never applied

**`BUG-2026-07-10T160102`** (medium, sites) — closed claiming the inline sites
migration SQL was extracted into a `sitesMigration` const. It was not.
`components/sites/sites.go:175` still holds the `CREATE TABLE IF NOT EXISTS sites`
statement inline in `Start()`. Only `domainsMigration` (`components/sites/domains.go:24`)
was ever extracted. Independently confirmed: `grep -rn "sitesMigration"` returns nothing.

### 4.2 PARTIAL — a security fix that is wired to nothing

**`BUG-2026-07-10T160002`** (high, auth, CWE-287) — "API key scopes not enforced in
auth middleware". The fix added `ResolveOrgKey`, a `ctxOrgKeyScopes` context key,
and an `OrgKeyScopesFromContext()` accessor.

`OrgKeyScopesFromContext` has **zero callers**. Independently verified: the only
occurrences anywhere are its own definition at `components/auth/middleware.go:166`,
the registry's own prose, and stale `.ctxo` index artifacts. Scopes are resolved,
stashed in the request context, and then never read by any REST handler in
`sites`, `api`, or `deploy`.

The middleware half of the fix is real; the enforcement half does not exist. A
scoped org key still gets unrestricted CRUD on org resources through the core API.
**This bug should not be considered closed.** It also has no regression guard, and
its `BUG-*.md` file is missing, so nothing would have caught this.

### 4.3 Other PARTIAL findings

| Bug | Severity | What is actually missing |
|---|---|---|
| `BUG-2026-06-19T131025` | high | No `.golangci.yml` exists anywhere — the "exclude dependency directories" fix was never implemented. CI avoids the symptom only because it never runs `npm ci` before linting. Confirmed by this sweep's own lint run hitting `ui/node_modules/flatted` |
| `BUG-2026-07-10T160107` | medium | `internalTableRegexes` is built in `init()` but never referenced; `handleSQL` still calls `regexp.Compile` per table per request at `components/api/api.go:542` |
| `BUG-2026-07-10T160112` | medium | Scoped 7 components for coverage; only `monitoring` received tests |
| `BUG-2026-06-02T211731` | medium | Code-level fix present; whether production has the GitHub App provisioned is a manual operator step, not verifiable from source |

### 4.4 The orphan-commit notes were wrong

Two entries carried `note: Fix exists in orphaned git commit … needs recovery`.
Both commits (`8d217c9d`, `c83772f2`) are genuinely unreachable from the baseline
— **but both code fixes are present in HEAD**, reapplied through refactors:

- `BUG-2026-07-10T160001` → `components/auth/auth.go:222` (CWE-601 panic guard), test `TestOAuthPublicURLOrDefaultIgnoresHostHeader` at `auth_test.go:1271`
- `BUG-2026-07-10T160002` → `components/auth/middleware.go:73` (`ResolveOrgKey`) — though see §4.2

The notes were stale metadata, not outstanding work. They have been corrected.

## 5. Behavioural proof (Tier 3)

A scratch instance was booted on an ephemeral SQLite DB and probed over HTTP.
**12 PASS / 0 FAIL / 1 by-design.** The harness is `specs/verifications/.sweep/probe.py`.

| Probe | Result |
|---|---|
| Unauthenticated `/api/auth/me` rejected | 401 |
| Site listing org-scoped | disjoint sets |
| Cross-tenant site fetch by id | 404 |
| **Cross-tenant IDOR on a real seeded site** | owner 200, attacker 404 on GET **and** DELETE |
| cici cross-tenant workflow write | 404 |
| Invalid `bb_dep_` key rejected | 401 |
| CSP header present on HTML routes | present |
| `X-Content-Type-Options: nosniff` | present |
| Login brute-force protection | 401×4 then 429 — lockout engages |
| Internal error details not leaked | no stack trace / SQL / path in body |
| No static directory listing | no index markers |

The first cross-tenant probe compared two empty site lists, which proves nothing,
so a real site row was seeded under one org and re-probed from another. That
stronger form is what the table reports.

### One by-design behaviour worth a second look

Legacy `org_id = 0` site rows are visible to **every** org. This is deliberate —
`components/sites/sites.go:352` reads `WHERE s.org_id = ? OR s.org_id = 0`, with
the comment that `org_id=0` is never a real caller's org. The safety of that rests
entirely on the `Start()` migration reassigning orphans to an admin org, which
runs only at boot and only when an admin user with `default_org_id > 0` exists.
Nothing prevents new `org_id = 0` rows from appearing. Filed as a hardening bug,
not a regression — `BUG-2026-07-24T184443`'s own stated goal is met.

## 6. Systemic registry problems

| Problem | Count | Detail |
|---|---|---|
| Closed bugs with **no `BUG-*.md`** | **21** | 7 have no `file:` field at all; the rest point at nonexistent paths |
| Closed bugs with **no regression guard** | **39** | Nothing would catch a recurrence |
| `.md` files unreferenced by the registry | 16 | Orphaned documentation |
| md/registry status conflicts | 6 | incl. `BUG-130` (critical) reading `status: open` while registry says `fixed` |
| Duplicate ids | 3 | one group mixes a `wontfix` row in with `fixed` rows |
| Guard names pointing at nonexistent tests | ≥3 | e.g. `TestCrossTenantStorageIsolation` (real name: `TestStorageOrgIsolation`); `TestCrossOrgSiteAccessDenied` does not exist |
| Guards naming a **non-test** file | 1 | `components/auth/anonymous.go TestPostMessageFailClosed` |
| Missing audit source | 12 bugs | cite `specs/audit-code-2026-07-10.md`, which exists nowhere in the repo or git history |
| Dependency bugs closed with no target version | 6 | see §7 |

## 7. Dependency bugs cannot be certified as a class

Six of seven `deps` bugs came back `CANNOT_DETERMINE` — not because the
dependencies are vulnerable, but because **no bug entry records the version the
CVE was fixed in**. Each says only "update to the latest version". Current
resolved versions were captured (`golang.org/x/crypto v0.54.0`,
`golang.org/x/net v0.57.0`, `undici 7.28.0`, `vite 8.1.5`, `vitest 4.1.10`), but
with no target to compare against, "fixed" is unfalsifiable.

The verifying agent was explicitly instructed not to invent CVE fix versions from
memory, and correctly declined. The one certifiable case is
`BUG-2026-07-11T021059-01` (esbuild): vite 8.1.5 bundles via rolldown, so esbuild
is not in the dependency tree at all.

**Recommendation:** dependency bug entries must record the patched version at
closure time, or they can never be re-verified.

## 8. Excluded rows (accounted for, not dropped)

9 `wontfix` + 2 `deferred` = 11 entries excluded by the user-agreed scope.
The `deferred` pair should be re-checked separately — deferral is easy to forget.

## 9. GitHub issue reconciliation

46 closed issues:

| Classification | Count | |
|---|---|---|
| LINKED to a registry bug | 14 | 57, 129–134, 136, 138, 141, 143, 155, 161, 162 |
| COMMIT_ONLY | 2 | 176, 177 |
| NOT_A_BUG (feature/chore/docs) | 15 | 40, 42, 44, 45, 58–62, 95, 106, 109, 153, 154, 174 |
| **ORPHANED** | **15** | 65, 99–103, 135, 137, 139, 140, 142, 144, 145, 146, 173 |

The 15 orphaned issues include **security issues closed with no registry entry
and no traceable commit** — #99 (Zip Slip), #101 (SQL from user-controlled data),
#102 (uncontrolled path data), #103 (open redirect), #135/#137 (IDOR on deploy
keys / MCP tools), #139/#140/#142 (forge IDOR, cross-tenant message leak,
WebSocket hijacking), #145/#146 (missing realtime auth, unauthenticated
Prometheus). Some are plausibly covered by work recorded under different bug ids,
but **none can be traced from the artifact trail**, which is the point.

## 10. Per-bug scorecard

Sorted worst verdict first. `Guards` = number of `regression_guards` entries;
`Doc` = whether the `BUG-*.md` exists; `Probes` = passing live HTTP probes.

| Bug | Sev | Verdict | Guards | Doc | Probes | Summary |
|---|---|---|---|---|---|---|
| `BUG-2026-07-10T160102` | medium | REGRESSED | 1 | **MISSING** | – | Sites component split 1600+ lines — extract inline migration to co |
| `BUG-2026-06-19T131025` | high | PARTIAL | 0 | yes | – | Full lint baseline blocks verify-work despite story-scoped checks  |
| `BUG-2026-07-10T160002` | high | PARTIAL | 2 | **MISSING** | – | API key scopes not enforced in auth middleware — full org access ( |
| `BUG-2026-06-02T211731` | medium | PARTIAL | 0 | yes | – | Production VPS never receives GitHub App credentials — /api/github |
| `BUG-2026-07-10T160107` | medium | PARTIAL | 1 | **MISSING** | – | SQL endpoint regex blocking — per-request regex compilation causin |
| `BUG-2026-07-10T160112` | medium | PARTIAL | 1 | **MISSING** | – | Seven components below 60% test coverage |
| `BUG-2026-07-11T021033` | critical | UNPROVEN | 0 | yes | – | golang.org/x/crypto — 12 vulns (7 CRITICAL, 1 MAJOR, 4 NORMAL) |
| `BUG-2026-07-11T021059-03` | critical | UNPROVEN | 0 | yes | – | Vitest UI (npm) — arbitrary file read/execute — CVE-2026-47429 (CV |
| `BUG-2026-07-11T021041-02` | high | UNPROVEN | 0 | yes | – | golang.org/x/crypto — underflow panic — CVE-2026-46597 (CVSS 7.5) |
| `BUG-2026-07-11T021051-01` | high | UNPROVEN | 0 | yes | – | undici (npm) — 7 vulns: WebSocket DoS, TLS bypass, info disclosure |
| `BUG-2026-07-11T021041-01` | medium | UNPROVEN | 0 | yes | – | Go net/html parser DoS — CVE-2026-25680 (CVSS 6.5) |
| `BUG-2026-07-11T021051-02` | medium | UNPROVEN | 0 | yes | – | Vite (npm) — path traversal + fs.deny bypass |
| `BUG-2026-07-11T020149-03` | low | UNPROVEN | 0 | yes | – | Information Disclosure - Suspicious Comments — HTML/JS comments in |
| `BUG-004` | critical | CERTIFIED-WEAK | 8 | yes | – | Python/uvicorn deployments fail — missing cmd.Dir, literal $PORT,  |
| `BUG-2026-07-11T032535` | critical | CERTIFIED-WEAK | 0 | yes | – | 3 git command injection sites (CWE-78) — git clone/fetch uses user |
| `BUG-144` | high | CERTIFIED-WEAK | 1 | **MISSING** | – | Insecure postMessage targetOrigin during OAuth popup — JWT leak to |
| `BUG-2026-06-04T100000` | high | CERTIFIED-WEAK | 0 | yes | – | Reviewer must-fix findings on prototype-fidelity-parity: mock succ |
| `BUG-2026-06-04T114000` | high | CERTIFIED-WEAK | 4 | yes | – | Production site deployments store localhost URL instead of https:/ |
| `BUG-2026-06-04T120500` | high | CERTIFIED-WEAK | 0 | yes | – | Node deploy fails on VPS — npm not in PATH; create-site wizard sho |
| `BUG-2026-06-04T120800` | high | CERTIFIED-WEAK | 0 | yes | – | Node deploy fails — npm EACCES (no HOME) and Node 18 too old for V |
| `BUG-2026-06-04T152200` | high | CERTIFIED-WEAK | 0 | yes | – | Site subdomain HTTPS fails — Caddy cannot obtain *.bigbase.click w |
| `BUG-2026-06-05T172800` | high | CERTIFIED-WEAK | 0 | yes | – | Deploy health check returns 502 and rollback fails with "Text file |
| `BUG-2026-06-19T163327` | high | CERTIFIED-WEAK | 1 | yes | – | Site deletion returns 500 when deployments table missing site_id c |
| `BUG-2026-06-19T173242` | high | CERTIFIED-WEAK | 7 | yes | – | Python deploys fail with PEP 668 + alert checker missing duration_ |
| `BUG-2026-06-20T014500` | high | CERTIFIED-WEAK | 3 | yes | – | Redeployment does not replace static files — old content persists  |
| `BUG-2026-06-21T120000` | high | CERTIFIED-WEAK | 3 | yes | – | Orphaned process-based apps (Python/Go/Node) survive BigBase resta |
| `BUG-2026-06-21T153000` | high | CERTIFIED-WEAK | 0 | yes | – | Process-based apps (Python/Go/Node SSR) not resumed after BigBase  |
| `BUG-2026-06-29T000000` | high | CERTIFIED-WEAK | 0 | yes | – | auth-astro middleware never sets storedToken — getSession() always |
| `BUG-2026-07-06T213500` | high | CERTIFIED-WEAK | 1 | yes | – | Sites list slow and incomplete — N+1 latestDeployment queries hit  |
| `BUG-2026-07-09T001200` | high | CERTIFIED-WEAK | 0 | yes | – | CI fails on flaky TestProxyAuthPolicy — missing waitForServer afte |
| `BUG-2026-07-10T160001` | high | CERTIFIED-WEAK | 1 | **MISSING** | – | OAuth redirect URI uses attacker-controlled Host when publicURL un |
| `BUG-2026-07-10T160004` | high | CERTIFIED-WEAK | 2 | yes | – | Four production cross-component direct imports violate ECC |
| `BUG-2026-07-10T160007` | high | CERTIFIED-WEAK | 1 | **MISSING** | – | 56 duplicated writeJSON/noopLogger/generateID/DBer copies |
| `BUG-2026-07-10T160008` | high | CERTIFIED-WEAK | 2 | yes | – | OAuth callback paths use PublicURLOrDefault Host fallback |
| `BUG-2026-07-10T220000` | high | CERTIFIED-WEAK | 0 | yes | – | Deploy Keys — revoke and copy operations broken; missing prefix in |
| `BUG-2026-07-11T030000` | high | CERTIFIED-WEAK | 0 | yes | – | v2.76.9 CSP blocks browser stylesheets on API error pages; deploy  |
| `BUG-2026-07-11T032549` | high | CERTIFIED-WEAK | 2 | yes | – | Path traversal (CWE-22) — 3 true positives: engine.go, gateway.go, |
| `BUG-2026-07-24T091500` | high | CERTIFIED-WEAK | 0 | yes | – | Node deploy hardcodes npm — pnpm projects fail with pnpm: not foun |
| `BUG-2026-07-24T091500` | high | CERTIFIED-WEAK | 0 | yes | – | Node deploy hardcodes npm — pnpm projects fail with pnpm not found |
| `BUG-2026-07-25-port-allocator-no-liveness-check` | high | CERTIFIED-WEAK | 2 | yes | – | Port allocator hands out ports still held by orphaned processes |
| `BUG-20260628T050000` | high | CERTIFIED-WEAK | 1 | yes | – | Anonymous tokens can write/delete records — missing role check in  |
| `BUG-2026-06-01T193841` | medium | CERTIFIED-WEAK | 0 | yes | – | GitHub and Sites components not wired into backend registration, p |
| `BUG-2026-06-02T163718` | medium | CERTIFIED-WEAK | 0 | yes | – | Root URL shows agent waiting page; cockpit and projectDir not used |
| `BUG-2026-06-03T120000` | medium | CERTIFIED-WEAK | 0 | yes | – | GitHub install callback and webhook behind auth middleware — retur |
| `BUG-2026-06-03T122114` | medium | CERTIFIED-WEAK | 0 | yes | – | Create site hides Connect GitHub when connected but /api/github/re |
| `BUG-2026-06-18T174500` | medium | CERTIFIED-WEAK | 0 | yes | – | Semantic Release CI fails — npx cannot resolve @semantic-release/c |
| `BUG-2026-06-23T184500` | medium | CERTIFIED-WEAK | 0 | yes | – | BigBase app logs not forwarded to New Relic — nrslog integration m |
| `BUG-2026-06-28T154700` | medium | CERTIFIED-WEAK | 1 | yes | – | Bypassable programmatic Options.Secret validation in auth.New |
| `BUG-2026-06-30T223500` | medium | CERTIFIED-WEAK | 1 | yes | – | EventsPage missing from sidebar navigation — route exists but no n |
| `BUG-2026-07-07T223200` | medium | CERTIFIED-WEAK | 2 | yes | – | MCP references list_sites but tool missing; no site key list/revok |
| `BUG-2026-07-10T160101` | medium | CERTIFIED-WEAK | 1 | **MISSING** | – | Deploy god object 1924 lines — needs decomposition per ADR 0005 |
| `BUG-2026-07-10T160105` | medium | CERTIFIED-WEAK | 1 | **MISSING** | – | Error string matching instead of sentinels — fragile string compar |
| `BUG-2026-07-10T160108` | medium | CERTIFIED-WEAK | 1 | yes | – | Refresh token rotation logic not fully verified (CWE-613) |
| `BUG-2026-07-10T160110` | medium | CERTIFIED-WEAK | 2 | yes | – | OAuth/token cookies use Secure: r.TLS != nil behind TLS proxy (CWE |
| `BUG-2026-07-10T160202` | medium | CERTIFIED-WEAK | 1 | yes | – | auth.go 1756 lines — split handlers by concern |
| `BUG-2026-07-11T021059-01` | medium | CERTIFIED-WEAK | 0 | yes | – | esbuild (npm) — dev server request smuggling |
| `BUG-2026-07-11T032535` | medium | CERTIFIED-WEAK | 0 | yes | – | CodeQL SQL injection alerts #8-#15 — passthrough wrappers; tainted |
| `BUG-2026-07-12T171000` | medium | CERTIFIED-WEAK | 2 | yes | – | Python deploys fail when uv is not installed on build server |
| `BUG-2026-07-25T004837` | medium | CERTIFIED-WEAK | 0 | yes | – | TestTriggerRunsThroughSupervisor flakes under CI load (2s eventual |
| `BUG-2026-06-01T182800` | low | CERTIFIED-WEAK | 0 | yes | – | GitHub App config flags not registered or parsed in serve command |
| `BUG-2026-06-02T000000` | low | CERTIFIED-WEAK | 0 | yes | – | GitHub App CLI flags not registered in serve command |
| `BUG-2026-06-22T030000` | low | CERTIFIED-WEAK | 0 | yes | – | Code quality issues in e40s03 manifest implementation — error leak |
| `BUG-2026-06-22T031000` | low | CERTIFIED-WEAK | 0 | yes | – | CI failures: TestSiteManifestGetAndSave assumes git default branch |
| `BUG-2026-06-27T131500` | low | CERTIFIED-WEAK | 1 | yes | – | rate-limit-enabled bool flag uses manual os.Getenv instead of Flag |
| `BUG-2026-07-10T160103` | low | CERTIFIED-WEAK | 1 | **MISSING** | – | Forge component split 536 lines — split into entity files by conce |
| `BUG-2026-07-10T160104` | low | CERTIFIED-WEAK | 1 | **MISSING** | – | ConfigSchema dead interface — 18 components implement ConfigSchema |
| `BUG-2026-07-10T160113` | low | CERTIFIED-WEAK | 1 | **MISSING** | – | Hooks dead interface — 18 components implement Hooks() returning n |
| `BUG-2026-07-10T160201` | low | CERTIFIED-WEAK | 1 | **MISSING** | – | AppType enum OCP — hardcoded constants without validation or enume |
| `BUG-2026-07-10T160203` | low | CERTIFIED-WEAK | 1 | **MISSING** | – | Inline migrations extract — extract inline SQL DDL to named consta |
| `BUG-2026-07-10T160204` | low | CERTIFIED-WEAK | 1 | **MISSING** | – | Export state machine — stateMachine type and newStateMachine const |
| `BUG-2026-07-10T160205` | low | CERTIFIED-WEAK | 1 | **MISSING** | – | Missing Component compile checks — no var _ kernel.Component = (*T |
| `BUG-2026-07-10T160207` | low | CERTIFIED-WEAK | 1 | **MISSING** | – | Manifest path traversal (CWE-22) — user-controlled manifestPath es |
| `BUG-2026-07-10T160208` | low | CERTIFIED-WEAK | 1 | yes | – | Empty CORS allowlist passes through — comment says closed (CWE-942 |
| `BUG-2026-07-10T160209` | low | CERTIFIED-WEAK | 1 | **MISSING** | – | ctxo indexes packages/* only — 208 Go files excluded from dependen |
| `BUG-2026-07-10T192800` | low | CERTIFIED-WEAK | 0 | yes | – | TestDeployCustomSiteName cleanup failure on CI — .git dir not empt |
| `BUG-130` | critical | CERTIFIED | 2 | yes | 1 | RCE via cici workflows — IDOR to unsandboxed exec.CommandContext ( |
| `BUG-2026-07-24-deploy-404-site-not-found` | critical | CERTIFIED | 2 | yes | 1 | HTTP 404 site not found on bb_dep_ deploy after #163 — orphan key/ |
| `BUG-2026-07-24-deploy-key-organization-required` | critical | CERTIFIED | 1 | yes | 1 | bb_dep_ site deploy keys get 403 organization required on /api/sit |
| `BUG-2026-07-24-static-directory-listing` | critical | CERTIFIED | 2 | yes | 1 | Static hosts serve git checkout FileServer directory listings — so |
| `BUG-2026-07-24T184443` | critical | CERTIFIED | 0 | yes | 1 | Sites UI shows only 1 site — BUG-136's org_id scoping fix has no f |
| `BUG-131` | high | CERTIFIED | 1 | yes | – | IDOR on all CRUD handlers — functions table had no org_id column ( |
| `BUG-133` | high | CERTIFIED | 1 | yes | 1 | IDOR on storage file handlers — no org_id scoping (CWE-639) |
| `BUG-135` | high | CERTIFIED | 1 | yes | 3 | IDOR on site deploy keys — no ownership verification (CWE-639) |
| `BUG-136` | high | CERTIFIED | 1 | yes | 3 | IDOR on all site endpoints — no org_id multi-tenant isolation (CWE |
| `BUG-140` | high | CERTIFIED | 1 | yes | – | Cross-tenant message leak — messages table had no org_id (CWE-639) |
| `BUG-2026-06-04T112700` | high | CERTIFIED | 2 | yes | – | Sites/Create site UI missing prototype CSS — mashed text, broken s |
| `BUG-2026-07-10T160003` | high | CERTIFIED | 2 | yes | 1 | Password login has no rate limiting or lockout (CWE-307) |
| `BUG-2026-07-10T160005` | high | CERTIFIED | 1 | **MISSING** | 1 | MCP error details leaked at 13+ sites (CWE-200) — raw error messag |
| `BUG-2026-06-04T120000` | medium | CERTIFIED | 6 | yes | – | Dashboard diverged from prototype — CPU always 0%, no Memory tile, |
| `BUG-2026-06-20T125952` | medium | CERTIFIED | 2 | yes | – | MCP SSE connection fails with 405 — GET /mcp blocked by POST-only  |
| `BUG-2026-06-26T200000` | medium | CERTIFIED | 4 | yes | – | e44s02: Missing test coverage for rollback UI functions — SiteDeta |
| `BUG-2026-07-10T160106` | medium | CERTIFIED | 1 | **MISSING** | – | NewMCPServer god function 632 lines — monolithic tool registration |
| `BUG-2026-07-11T020132` | medium | CERTIFIED | 0 | yes | 1 | CSP: Failure to Define Directive with No Fallback — missing explic |
| `BUG-2026-07-11T032549` | medium | CERTIFIED | 2 | yes | – | Weak cryptographic hashing — raw SHA-256 for API key storage (CWE- |
| `BUG-2026-07-11T032549` | normal | CERTIFIED | 2 | yes | – | Cookie 'Secure' attribute not set on 3 http.SetCookie calls (CWE-6 |
| `BUG-2026-07-11T032549` | normal | CERTIFIED | 2 | yes | – | Open URL redirect (CWE-601) — isSPAOriginAllowed uses strings.HasP |
| `BUG-2026-06-12T150000` | low | CERTIFIED | 1 | yes | 1 | CSP blocks home page styles and Google Fonts — restricted to defau |
| `BUG-2026-07-10T160109` | low | CERTIFIED | 2 | **MISSING** | – | Custom DBer interfaces duplicated across mcp, backup, proxy — shou |
| `BUG-2026-07-10T160206` | low | CERTIFIED | 1 | yes | – | WebSocket accepts all origins — TODO in production |
| `BUG-2026-07-11T020149-05` | low | CERTIFIED | 0 | yes | 1 | Re-examine Cache-control Directives — suboptimal caching headers |
---

## 11. How this sweep guarded itself against a false "all green"

| Failure mode | Mitigation | Did it fire? |
|---|---|---|
| Skipped tests counted as passes | `SKIP` tracked as its own state; forces `UNPROVEN` | Yes — 5 bugs |
| Guard name drift fuzzy-matched to the wrong test | hard 0.6 confidence floor; fuzzy hits capped at `CERTIFIED-WEAK` and flagged | Yes — 1 (`BUG-133`) |
| Stub `echo ok` package tests treated as evidence | 5 `packages/auth-*` scripts denylisted → `STUB_NOOP` | Available, not triggered |
| Deploy tests mass-skipping when npm absent | preflight recorded npm 11.16.0 / bun 1.3.14 present | Did not occur |
| Postgres leg not run but scored as if it were | tracked per-leg; DB-sensitive scopes capped at `CERTIFIED-WEAK` | Yes — 18 bugs |
| `commit_message` exact-matched against `git log` | fuzzy token match only; a miss is a recorded input | Confirmed drift exists |
| Duplicate ids collapsing rows | keyed by `(id, file)`; row-count assertion | Yes — 3 ids |
| Environment-specific test failures read as regressions | cross-checked against CI's Node version | **Yes — caught 6 false regressions** |

**Determinism:** `score.py` was run twice on identical inputs and produced
byte-identical output.

## 12. Recommended next actions

1. **Reopen `BUG-2026-07-10T160002`** — the scope-enforcement fix is dead code. Highest priority; it is a live authorization gap.
2. **Reopen `BUG-2026-07-10T160102`** — the claimed migration extraction was never made.
3. Add `OrgKeyScopesFromContext` enforcement to the REST handlers, with a regression test.
4. Backfill the 21 missing `BUG-*.md` files, or null their `file:` fields so the gap is visible.
5. Add regression guards to the 39 closed bugs that have none, prioritising the security-scoped ones.
6. Record patched versions on dependency bugs at closure time.
7. Add a `.golangci.yml` excluding `ui/node_modules` (`BUG-2026-06-19T131025`).
8. Triage the 15 orphaned closed GitHub issues — several are security issues with no artifact trail.
9. Pin CI and local development to the same Node major, or fix the Node 26 localStorage incompatibility.

---

## 13. Write-backs applied by this sweep

### Reopened (status `fixed` → `open`)

| Bug | Why |
|---|---|
| `BUG-2026-07-10T160002` (high, auth) | Enforcement half of the fix does not exist — `OrgKeyScopesFromContext` has zero callers |
| `BUG-2026-07-10T160102` (medium, sites) | Claimed migration extraction was never applied |

### Registry metadata repaired

- Removed the stale `note: Fix exists in orphaned git commit 8d217c9d — needs recovery` from `BUG-2026-07-10T160001` and replaced it with a `validated:` block recording that the fix **is** in HEAD at `components/auth/auth.go:222`.
- Same for `BUG-2026-07-10T160002` (commit `c83772f2`), with the caveat in §4.2.
- `BUG-133`: guard `TestCrossTenantStorageIsolation` → `TestStorageOrgIsolation` (the real name).
- `BUG-136`: guard `TestCrossOrgSiteAccessDenied` (nonexistent) → `TestRequireSiteOwnership_StillDeniesRealCrossOrgSite` + `TestListSites_StillIsolatesRealCrossOrgSites`.
- `BUG-144`: guard named a non-test file (`components/auth/anonymous.go`) → the four real tests in `anonymous_test.go`.
- `BUG-2026-07-10T160002`: `file:` set to `null` (the referenced doc does not exist); `audit_source` annotated as missing.

### Bug docs reconciled

- `BUG-130-rce-cici-workflows.md`: frontmatter `open` → `fixed` (critical bug that read as open while both source and a live probe confirm the fix).
- `BUG-2026-07-24T091500-deploy-pnpm-not-found.md`: frontmatter `open` → `fixed`.

### New bugs filed

| Id | Sev | Summary |
|---|---|---|
| `BUG-2026-07-28T000001` | high | Org API key scopes resolved into context but never enforced (CWE-287) |
| `BUG-2026-07-28T000002` | medium | Legacy `org_id=0` site rows visible to every organization |
| `BUG-2026-07-28T000003` | medium | UI suite fails on Node 26 (experimental built-in localStorage) |
| `BUG-2026-07-28T000004` | low | No `.golangci.yml`; lint scans vendored Go inside `ui/node_modules` |
| `BUG-2026-07-28T000005` | medium | Registry integrity: 21 missing bug docs, 39 unguarded closed bugs, missing audit source |

Registry after write-back: **116 entries** — 97 `fixed`, 7 `open`, 9 `wontfix`, 2 `deferred`, 1 `done`.

### Proposed GitHub issues (NOT opened — awaiting approval)

1. Enforce org API key scopes in REST handlers (`BUG-2026-07-28T000001`) — highest priority.
2. Reopen/track the never-applied sites migration extraction (`BUG-2026-07-10T160102`).
3. Triage the 15 orphaned closed security issues listed in §9.
4. Add `.golangci.yml` excluding `ui/node_modules`.
5. Pin local Node to the CI major, or stub localStorage in the vitest setup.

### Post-write-back verification

- `go build ./...` — PASS
- `go test ./... -count=1` (sqlite) — 27/27 packages ok, 0 failures
- `git status` — only the intended files changed. Note that `specs/bugs/registry.yaml`, `components/deploy/deploy_test.go`, `specs/state.yaml`, the `.mcp.json` files and two `BUG-2026-07-24-*.md` docs were **already modified** in the working tree by a concurrent session before this sweep began; those changes are not this sweep's and were left untouched.
