# e72s01: AI Deploy Failure Diagnosis
## Story ID: e72s01 | Epic: e72 | BCPs: 3 | Status: planned

## 1. Type
**feat** · domain

## 2. Context
When a BigBase user's deployment fails, they see a build log and an error message. The logs can be hundreds of lines of npm/webpack/go output — overwhelming for non-expert users. The `deploy` component already emits `deploy.failed` events with the build log attached. The `mcp` component already has LLM connectivity. We can bridge these: when a deployment fails, feed the truncated build log + error to an LLM and return a human-readable diagnosis with fix suggestions.

Tracer-Cloud's opensre demonstrates the value of AI-driven incident analysis. This story applies that pattern to BigBase's specific domain: deployment failures.

## 3. Problem / Opportunity
- Failed deployments produce verbose build logs that users must parse manually
- BigBase already has LLM access via the MCP component (OpenAI/Anthropic)
- The `deploy.failed` event already carries build_log + error_message
- No existing mechanism to translate raw errors into actionable fix suggestions

## 4. Proposed solution

### New component or extension?
**Extension to `monitoring`** — this is an observability feature. Add a `DiagnoseDeployFailure` method that:
1. Listens for `deploy.failed` events on the kernel event bus (already subscribed to `deploy` hook)
2. Truncates the build log to the last 200 lines (LLM context window safety)
3. Constructs a structured prompt: app type, error message, build log tail, BigBase platform context
4. Calls the LLM via `mcp` component's existing provider resolution (`core/agent_harness/llm_resolution.py` → but we need a Go equivalent)

**Architecture decision:** Rather than calling Python MCP from Go, add a lightweight LLM call helper in the monitoring component. BigBase already carries `openai` and `anthropic` Go SDK dependencies? **No — those are Python opensre dependencies, not BigBase's.** 

BigBase's `mcp` component is an MCP *server* (tools for AI agents), not a client. For LLM calls from within BigBase, we need a simple HTTP client to OpenAI-compatible APIs.

**Revised approach (ADR 0007):** Add `components/internal/llm/` and `deploy_diagnoses` table. Subscribe to **`deploy.failed`** (emitted once by deploy on `TransitionState → failed`). Deploy Gateway owns `GET /api/deploy/:id/diagnosis` via injected `DeployDiagnosisReader` — monitoring implements, `main.go` wires. No cross-component imports.

## 5. Alternatives considered
- **Call Python opensre**: Cross-language, cross-process — violates ECC architecture. Rejected.
- **Add LLM client to deploy component**: Deploy is already 1819 lines. Better to keep diagnosis separate.
- **External webhook to an AI service**: Adds network dependency, latency. In-process is simpler.
- **Do nothing — users read build logs**: Ok for experts, hostile to non-experts. Misses the platform value proposition.

## 6. Who are the users?
- **BigBase users** who deploy apps and see "Deployment failed" with cryptic build output
- **Platform operators** who triage user-reported deployment issues

## 7. Dependencies
- `net/http` (stdlib)
- `encoding/json` (stdlib)
- `deploy` component (event bus emission of `deploy.failed`)
- `monitoring` component (event bus subscription)
- [DeepSeek API](https://api-docs.deepseek.com) — OpenAI-compatible chat completions (default provider)
- `BIGBASE_LLM_API_KEY` env var (or `DEEPSEEK_API_KEY` fallback)
- `BIGBASE_LLM_BASE_URL` env var (default: `https://api.deepseek.com`)
- `BIGBASE_LLM_MODEL` env var (default: `deepseek-chat`)

## 8. Assumptions
- OpenAI-compatible chat completion API (DeepSeek default) is available
- Build logs are text, not binary
- Most deployment failures have actionable causes (missing dependency, wrong Node version, build script error)
- LLM cost per diagnosis is negligible with deepseek-chat (~$0.001 per call)

## 9. Risks
| Risk | Probability | Impact | Mitigation |
|------|-----------|--------|------------|
| LLM hallucinates wrong fix | Medium | Low | Label as "AI suggestion — verify before applying" |
| API key missing → feature silently unavailable | Medium | Low | Log warning, return nil diagnosis |
| LLM call slows down event processing | Low | Medium | Goroutine with 30s timeout; non-blocking event handler |
| Build logs contain secrets | Low | High | Strip env var values from logs before sending to LLM |

## 10. Non-goals
- Interactive follow-up questions (v1 is one-shot diagnosis)
- Fix auto-application (v1 only suggests)
- Multi-turn investigation
- LLM provider abstraction beyond OpenAI-compatible API

## 11. Migration plan
**Not applicable** — net-new feature. Falls back gracefully if no API key configured. No database migration.

## 12. Wireframes / Diagrams
```
deploy.failed event emitted (once, from deploy.TransitionState)
       │
       ▼
monitoring event handler (goroutine, non-blocking)
       │
       ▼
┌──────────────────────────────────────────────┐
│  BuildDiagnosisPrompt(appType, error, logs)  │
│  ├─ "Role: You are a deployment debugging   │
│  │   assistant for BigBase, a BaaS platform. │
│  │   App type: Node.js                       │
│  │   Error: exit status 1                    │
│  │   Build log tail (last 200 lines): ..."   │
│  └─ Returns prompt string                    │
└──────────────────┬───────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────┐
│  internal/llm.Complete(prompt)                │
│  ├─ POST https://api.deepseek.com/chat/...    │
│  ├─ Authorization: Bearer $BIGBASE_LLM_API_KEY│
│  ├─ Model: deepseek-chat (configurable)       │
│  └─ Returns diagnosis text or error           │
└──────────────────┬───────────────────────────┘
                   │
                   ▼
Store diagnosis in deploy_diagnoses
Emit "deploy.diagnosed" event with diagnosis
Expose via GET /api/deploy/:id/diagnosis
```

## 13. API / Data Model

### New HTTP endpoint
```
GET /api/deploy/:id/diagnosis
→ 200 { "diagnosis": "...", "created_at": "...", "model": "deepseek-chat" }
→ 404 { "error": "no diagnosis available" }
```

### New event
```
deploy.diagnosed {
  deploy_id: string
  diagnosis: string
  model: string
  created_at: string
}
```

### Env vars (LLM — DeepSeek default)
```
BIGBASE_LLM_API_KEY   — DeepSeek API key (fallback: DEEPSEEK_API_KEY)
BIGBASE_LLM_BASE_URL  — Default: https://api.deepseek.com
BIGBASE_LLM_MODEL     — Default: deepseek-chat
```

## 14. Affected code
| File | Change |
|------|--------|
| `components/internal/llm/llm.go` | **NEW** — chat completion client |
| `components/internal/llm/llm_test.go` | **NEW** — unit tests with httptest |
| `components/monitoring/monitoring.go` | Add `handleDeployDiagnosis` route + `DiagnoseDeployFailure` method |
| `components/monitoring/monitoring_test.go` | Test deploy.failed → diagnosis flow |
| `main.go` | Wire LLM config env vars |
| `specs/tech-architecture/tech-stack.md` | Document LLM integration |

## 15. Testing strategy
- **Unit**: Table-driven tests for `BuildDiagnosisPrompt` (various app types, error messages, log lengths)
- **Unit**: `llm.ChatCompletion` with `httptest.NewServer` mock
- **Integration**: End-to-end: trigger a failing deploy → verify diagnosis appears in API
- **No synthetic tests** needed — this is an assistive feature, not safety-critical

## 16. Rollback plan
Unset `BIGBASE_LLM_API_KEY` — feature silently disables. No data loss.

## 17. Acceptance Criteria
```gherkin
Scenario: AI diagnosis on deploy failure
  Given BIGBASE_LLM_API_KEY is configured
  And a deployment fails with a build error
  When the deploy.failed event is emitted
  Then a diagnosis is generated within 30 seconds
  And GET /api/deploy/:id/diagnosis returns the diagnosis

Scenario: Graceful degradation without API key
  Given BIGBASE_LLM_API_KEY is not configured
  When a deployment fails
  Then no error is surfaced to the user
  And GET /api/deploy/:id/diagnosis returns 404

Scenario: Build log truncation for large logs
  Given a deployment fails with a 5000-line build log
  When the diagnosis prompt is constructed
  Then only the last 200 lines are included in the prompt
```

## 18. Implementation Steps (see e72s01-tasks.yaml)

## 19. Verification Script (for manual UAT)
1. `export BIGBASE_LLM_API_KEY="<deepseek-api-key>"`  # or DEEPSEEK_API_KEY
2. `go run . serve --port 9999`
3. Create a repo with a deliberately broken `package.json` (missing dependency)
4. Deploy the repo via `bigbase deploy --repo <id>`
5. Wait for failure
6. `curl http://localhost:9999/api/deploy/<id>/diagnosis`
7. Verify diagnosis contains actionable suggestion

## 20. Out of scope
- Multi-turn conversational diagnosis
- Auto-apply suggested fixes
- Non-OpenAI LLM providers (v1 — extensible later)
- User feedback loop ("was this helpful?")
