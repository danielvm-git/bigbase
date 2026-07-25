# Impact Assessment — Open Issues vs Multi-Language Deploy

> **Context:** BigBase runs 5 app types (Node, Go, Python, PHP, Static) on a single VPS.
> All share: `pickPort` → `startApp` → proxy routing → env injection pipeline.
> A bug in any shared path affects ALL sites across ALL languages simultaneously.

---

## Issue #173 — Port Allocator (Postmortem Record)

### Target
`components/deploy/utils.go::pickPort` + `portIsFree`

### Dependents (7 confirmed)

| Symbol | Edge | Why it matters |
|--------|------|----------------|
| `engine.go::Trigger` | calls pickPort | **Every deployment** for every language goes through here |
| `gateway.go::HandleCreate` | calls Trigger | MCP `deploy_site` tool |
| `gateway.go::HandleDeploy` | calls Trigger | Redeploy endpoint |
| `samples.go::handleSamples` | calls Trigger | Sample site provisioning |
| `main.go::startProxy` | calls Trigger | Process recovery on restart |
| `orchestrator.go` | calls startApp | Resume interrupted deploys |
| `rollback.go` | calls startApp | Rollback for any language |

### Blast Radius
- **Fan-in:** 1 direct caller (Trigger), but Trigger is called by **4 different entry points**
- **Fan-out:** 0 (pure utility function)
- **Cross-cluster:** Deploy → Proxy (3 edges) — port assignment drives proxy routing

### Multi-Language Impact
| Language | Affected? | How |
|----------|-----------|-----|
| Node | ✅ YES | `npm start` / `node app.js` on assigned port |
| Go | ✅ YES | `./app` binary binds to assigned port |
| Python | ✅ YES | `uvicorn` / `python -m http.server` on assigned port |
| PHP | ✅ YES | `php -S 0.0.0.0:<port>` on assigned port |
| Static | ✅ YES | Caddy/file server on assigned port |

**Impact severity: 🔴 CRITICAL — already fixed**
- Before fix: orphaned processes from ANY language could silently serve wrong site content
- The "wrong site content served" symptom affected `python.bigbase.click`, `go.bigbase.click`, etc.
- The fix (OS-level `net.Listen` probe) protects all 5 runtimes

### Risk: ~~High~~ → **Resolved**
Fix is live (PR #171). The postmortem record should be **closed as completed**.

---

## Issue #174 — PHP Deploy Contract Docs Gap

### Target
Documentation surface: `.github` action descriptions + contract doc `app_type` table

### Dependents
| Surface | Impact |
|---------|--------|
| `danielvm-git/.github` `bigbase-deploy` action | `app_type` input doesn't mention `php` |
| Contract doc `app_type` table | Lists only static/python/go/node |
| CI templates | No `deploy-php.yml` or `test-build-release-php.yml` |

### Multi-Language Impact
| Language | Has CI template? | Has contract doc entry? | Tested live? |
|----------|-----------------|----------------------|-------------|
| Node | ✅ Yes | ✅ Yes | ✅ Yes |
| Go | ✅ Yes | ✅ Yes | ✅ Yes |
| Python | ✅ Yes | ✅ Yes | ✅ Yes |
| **PHP** | ❌ **No** | ❌ **No** | ✅ Yes (canary) |
| Static | ✅ Yes | ✅ Yes | ✅ Yes |

**Impact severity: 🟡 MEDIUM**
- PHP code works in `engine.go` (build via Composer, start via `php -S`)
- But no CI template exists → users must manually write deploy workflows
- The canary `bigbase-canary-php` proved it works, but discovery requires reading Go source
- **e81 (Multi-Language Deploy)** explicitly lists PHP as a supported runtime — docs must match

### Risk: Medium
Code works, but invisible features = silent user friction. Blocks e81 from shipping cleanly.

### Recommended action
1. Add PHP row to contract doc
2. Create `deploy-php.yml` + `test-build-release-php.yml` templates
3. Add PHP to the canary test matrix

---

## Issue #58 — e70 Site Deploy Manifest (`bigbase.toml`)

### Target
New feature: `bigbase.toml` manifest + deploy defaults on site record + parameterized CI templates

### Status: e70s01 ✅ e70s02 ✅ — 4/7 BCPs complete

### Dependents (planned)
| Symbol | Current state | After e70 |
|--------|--------------|-----------|
| `engine.go::Trigger` | reads app_type from request | reads from manifest first, falls back to request |
| `engine.go::buildApp` | no manifest awareness | consumes `Manifest.Start`, `Manifest.Env` |
| `engine.go::startApp` | hardcoded per-type switch | manifest-driven start command |
| `env.go::BuildEnv` | only platform env | merges manifest env vars |
| MCP `create_site` | no deploy defaults | ✅ persists `app_type`, `passthrough_paths`, `health_path` |
| MCP `get_ci_template` | generic YAML | parameterized per app_type + site defaults |

### Multi-Language Impact
| Language | Pain today | After e70 |
|----------|-----------|-----------|
| Node | Must repeat `app_type: node` in every workflow | Declared once in `bigbase.toml` |
| Go | Must repeat `app_type: go` | Declared once |
| Python | Must repeat `app_type: python` + venv setup | Manifest handles it |
| PHP | Must repeat `app_type: php` + composer install | Manifest handles it |
| Static | Must repeat `passthrough_paths` for SPA routing | Declared once |

**Impact severity: 🟢 LOW (planned, not blocking)**
- e70 is Wave 2, depends on e57 (superseded)
- The pain is real (copy-paste across CI, redeploy.py, agent workflows) but not a bug
- Blocks: e81 multi-language deploy polish

### Risk: Low
Planned enhancement. No current breakage. 7 BCPs across 4 stories.

---

## Issue #41 — Environment/Secrets Resolution Seam (`env.go`)

### Target
`components/deploy/env.go::BuildEnv` + `FetchSiteEnvVars`

### Dependents (14 confirmed — highest blast radius of all open issues)

| Symbol | Depth | Edge |
|--------|-------|------|
| `env.go::buildCmdEnv` | 1 | calls BuildEnv |
| `engine.go::runBuildCommand` | 2 | uses buildCmdEnv |
| `orchestrator.go::resumeCandidates` | 2 | uses buildCmdEnv |
| `engine.go::buildApp` | 3 | calls FetchSiteEnvVars |
| `engine.go::nodeInstall` | 3 | uses buildCmdEnv |
| `deploy.go::Start` | 3 | uses buildCmdEnv |
| `engine.go::runDeployment` | 4 | calls startApp |
| `engine.go::Trigger` | 5 | calls runDeployment |
| `mcpDeployAdapter::Trigger` | 6 | calls Trigger |
| `gateway.go::HandleCreate` | 6 | calls Trigger |
| `samples.go::handleSamples` | 6 | calls Trigger |
| `main.go::startProxy` | 6 | calls Trigger |
| `gateway.go::handleDeploy` | 7 | calls HandleDeployByID |
| `main.go::main` | 7 | calls startProxy |

### Multi-Language Impact
| Language | Build env needs | Runtime env needs | Secrets impact |
|----------|----------------|-------------------|---------------|
| Node | `NODE_ENV`, `NPM_TOKEN`, `PNPM_HOME` | Same + `PORT` | `NPM_TOKEN` leaks = supply chain attack |
| Go | `GOROOT`, `GOPATH`, `CGO_ENABLED` | `PORT` | Less secret-heavy |
| Python | `VIRTUAL_ENV`, `PYTHONPATH` | Same | `DATABASE_URL`, API keys |
| PHP | `PHP_INI_DIR`, extension paths | Same | DB creds, API keys |
| Static | None | None | None (served by proxy) |

**Impact severity: 🟡 MEDIUM (design debt, not active bug)**
- `env.go` is a shallow helper, not a proper module
- No merge precedence (platform → user → secrets)
- No redaction in build logs
- `FetchSiteEnvVars` already exists but is separate from `BuildEnv`
- e61 (Secrets Management, superseded) was supposed to own this seam
- e81 (Multi-Language Deploy) will add more runtime-specific env needs

### Risk: Medium
Not causing active bugs today, but:
- A secret redaction bug in `buildCmdEnv` would leak across ALL 5 runtimes
- The split between `BuildEnv` (build time) and `FetchSiteEnvVars` (runtime) creates two code paths for precedence logic
- e81 adds 4 new runtimes (Rust, Java, Ruby + PHP already there) — env complexity grows

### Recommended action
1. Design `EnvResolver` interface before e81 lands
2. Single merge point: platform defaults → user env vars → secrets
3. Redaction view consumed by both build and runtime paths
4. Gate: must land before e81 stories that add new runtimes

---

## Issue #43 — Policy Gate for Route Access Control

### Target
Route registration in `main.go` + auth middleware chains

### Dependents
| Area | Count | Example |
|------|-------|---------|
| Deploy area | 82 symbols | All deploy routes need auth |
| Auth area | 108 symbols | Core auth component |
| API area | 26 symbols | CRUD routes |
| Proxy area | 40 symbols | Routing decisions |
| Monitoring area | 58 symbols | Metrics/logs endpoints |
| Functions area | 24 symbols | Serverless endpoints |
| Components (all) | 591 likely | Every component registers routes |

**Total blast radius: 622 symbols across 18 clusters**

### Multi-Language Impact
| Language | Routes affected | Current auth enforcement |
|----------|----------------|------------------------|
| All | `/api/deploy/*` | Middleware chain in main.go |
| All | `/api/sites/*` | Middleware chain |
| All | `/api/env/*` | Middleware chain |
| All | `/<hostname>/*` | Proxy-level auth policy |

**Impact severity: ⚪ LOW (deferred, tier 8)**
- The forcing function (e66 Multi-User Platform) is superseded and lowest priority
- Current manual middleware chaining works but is fragile
- Not blocking any current work

### Risk: Low (but very high if touched carelessly)
- 622 symbol blast radius = highest of all issues
- Any change to the `Component` interface ripples to 18 clusters
- Must be designed as an additive layer, not a refactor of existing wiring

### Recommended action
1. Defer until e66 (or successor) is actively planned
2. Design as additive: `Policy` struct + central enforcer, don't refactor existing middleware
3. Test with `BypassPolicy` injection

---

## Summary Matrix

| Issue | Severity | Blast Radius | Multi-Language Risk | Action | Effort |
|-------|----------|-------------|--------------------|--------|--------|
| #173 port allocator | 🔴 CRITICAL | 7 symbols, 4 entry points | All 5 runtimes | ✅ Close (fixed) | 1 min |
| #174 PHP docs | 🟡 MEDIUM | 3 surfaces | PHP blocked from CI | Create templates + docs | ~2 hrs |
| #58 e70 bigbase.toml | 🟢 LOW | New code (planned) | All runtimes benefit | Build after e57 recheck | 7 BCPs |
| #41 env/secrets seam | 🟡 MEDIUM | 14 symbols, 2 code paths | Secrets leak = all runtimes | Design before e81 | Design + impl |
| #43 policy gate | ⚪ LOW | 622 symbols, 18 clusters | All routes | Defer to e66 | Design only |

---

## Critical Path for Multi-Language Support

```
#173 (CLOSED) → #174 (fix PHP docs) → #41 (env resolver) → e81 (multi-language) → #58 (bigbase.toml)
                  ↑                                        ↑
                  Fast win, unblocks e81                  Must land before e81 adds more runtimes
```

**#41 is the hidden blocker** — it's not a bug today, but e81 adds Rust/Java/Ruby/PHP runtimes, each with unique env needs. Without a proper `EnvResolver`, the env injection logic gets copy-pasted per runtime, creating redaction gaps and precedence bugs.
