# Story e30s01: Functions Runtime — fetch, db, request context, env, schedule

**type:** feat
**context:** domain
**epic:** e30 — Backend for Bots & Integrations
**bcps:** 8
**status:** planned
**wsjf:** 3.25 (BV=10 TC=8 RR=8 / JS=8)
**scope:** `components/functions/`

## Context

BigBase's Functions runtime (`jsRuntime` via goja) currently injects only `console.log`/`info`/`warn`/`error`. This makes Functions useless for integrations: no HTTP fetch, no database access, no access to `env` variables, no request context passed in, and `trigger=schedule` is stored but never executed.

This story upgrades the Functions runtime from a sandbox toy to a real integration engine by injecting five capabilities: HTTP client (`fetch`), database binding (`db.collection()`), environment variables (`env`), HTTP request context (`request`), and a cron-based schedule executor.

## Scope boundaries

- **In scope**: `fetch` with allowlist, `db` binding for CRUD on collections, `env` injection, `request` context for HTTP-triggered functions, schedule loop via `robfig/cron`
- **Out of scope**: Async/promise-based `fetch` (sync for now), `db` aggregation/query operators, `request` body parsing beyond JSON/text, multi-tenant org isolation (follows existing patterns), schedule persistence across restarts (functions re-register on Start)

## Acceptance Criteria (§17)

### AC1: fetch injection
**Given** a Function with `trigger=http` and source containing `fetch('https://example.com/api')`
**When** the function is executed via `POST /api/functions/:id/run`
**Then** the runtime performs an HTTP GET to the URL, returns `{status, headers, body}` to the JS, and logs the response

### AC2: fetch allowlist enforcement
**Given** a Function with `env.ALLOWED_HOSTS = "api.github.com"`
**When** the function calls `fetch('https://evil.com')`
**Then** the runtime blocks the request, returns an error, and logs a blocked-host message

### AC3: db binding
**Given** a Function with source calling `db.collection('messages').create({text: 'hello'})`
**When** the function executes
**Then** a record is inserted into the `messages` collection table and the function receives `{id: <int>}` as the return value

### AC4: env injection
**Given** a Function with `env: {API_KEY: "sk-123", REGION: "us-east"}`
**When** the function accesses `env.API_KEY` in source
**Then** the value `"sk-123"` is available. `env` is read-only (writes are silently ignored).

### AC5: request context
**Given** a Function with `trigger=http` and an HTTP POST with `{"name": "test"}` body and header `X-Custom: abc`
**When** the function accesses `request.method`, `request.body`, `request.headers['x-custom']`
**Then** values are `"POST"`, `{"name": "test"}`, and `"abc"` respectively

### AC6: schedule execution
**Given** a Function with `trigger=schedule` and `schedule="* * * * *"` (every minute)
**When** the BigBase server runs for 2+ minutes
**Then** the function executes at least once, with execution logged to `function_executions`

### AC7: schedule flag
**Given** BigBase started with `--functions-schedule=false`
**When** a minute passes
**Then** no schedule-triggered functions execute

## Out of scope

- Async `fetch` (promises/goja event loop)
- `db.collection().where()` or aggregation
- Multi-tenant org isolation (uses existing patterns from `api` component)
- Schedule persistence — re-registered on Start()

## Risks

| Risk | Detection | Mitigation |
|------|-----------|------------|
| goja doesn't support `fetch` promise pattern | Test early with sync HTTP call | Implement sync `fetch` that blocks (acceptable for MVP) |
| Schedule goroutine leak on Stop | Test `TestScheduleShutdown` | Use `cron.Stop()` which returns context that drains |
| Allowlist bypass via DNS rebinding | Security review | Phase 1: hostname check only; Phase 2: IP pinning |
| `db` binding SQL injection via collection name | Test with malicious names | Reuse `sanitize()` from api component |
