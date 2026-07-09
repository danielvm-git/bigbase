# e72s02: Deploy Pipeline Timeline
## Story ID: e72s02 | Epic: e72 | BCPs: 2 | Status: planned

## 1. Type
**feat** · domain

## 2. Context
BigBase deployments go through multiple stages: clone → build → start → health_check → running. Currently, the deploy component tracks these states internally via `TransitionState` but the timing between stages is not exposed. Users only see "pending" → "running" or "failed". 

Tracer-Cloud's pipeline visualization (showing per-stage timing, bottleneck identification) inspired this story. We can expose the stage timeline from BigBase's existing deploy state machine without any external dependencies.

## 3. Problem / Opportunity
- Users can't see *where* in the pipeline a deployment is slow
- No visibility into build-vs-startup time breakdown
- The deploy component already records state transitions internally — just not exposed
- A pipeline timeline helps users optimize their builds and understand platform performance

## 4. Proposed solution
Add a `PipelineTimeline` struct and tracking to the deploy component (ADR 0007 — renamed from `StageTimeline` to avoid collision with `DeploymentState`):

```go
type PipelineTimeline struct {
    CloneStart    time.Time `json:"clone_start,omitempty"`
    CloneEnd      time.Time `json:"clone_end,omitempty"`
    BuildStart    time.Time `json:"build_start,omitempty"`
    BuildEnd      time.Time `json:"build_end,omitempty"`
    StartStart    time.Time `json:"start_start,omitempty"`
    StartEnd      time.Time `json:"start_end,omitempty"`
    HealthStart   time.Time `json:"health_start,omitempty"`
    HealthEnd     time.Time `json:"health_end,omitempty"`
}
```

Store the timeline in the `deployments` table as `pipeline_timeline` JSON column. Expose via:
- `GET /api/deploy/:id` — add `pipeline_timeline` field

The deploy component's `runDeployment` / `buildApp` / `startApp` / `runHealthCheck` methods already have natural instrumentation points.

## 5. Alternatives considered
- **Separate timeline table**: Over-normalized for JSON timeline data. Inline JSON in deployments row is simpler.
- **External tracing (OpenTelemetry spans)**: Too heavy for a pipeline timeline. Plain timestamps suffice.
- **Only expose via SSE stream**: Useful later, but REST API is the MVP.

## 6. Who are the users?
- **Developers** deploying apps who want to know where their build time is spent
- **Platform operators** diagnosing slow deployments
- **Admin UI** — can render a timeline component showing colored stage bars

## 7. Dependencies
- `deploy` component (state machine, TransitionState)
- `monitoring` component (exposes API — or deploy itself handles the new endpoint)
- `database/sql` (JSON column in SQLite/Postgres)

## 8. Assumptions
- State transitions happen sequentially (clone→build→start→health→running)
- Timeline data is small enough for inline JSON storage (< 1KB)
- Timestamps from `time.Now()` are sufficient fidelity

## 9. Risks
| Risk | Probability | Impact | Mitigation |
|------|-----------|--------|------------|
| Partial timeline on crash | Medium | Low | Always write what we have; mark incomplete stages |
| JSON column migration on existing DBs | Low | Low | Default to null — backward compatible |

## 10. Non-goals
- Sub-stage timing (e.g., npm install vs npm build within build phase)
- Per-process resource usage during each stage
- Timeline comparison across deployments (v1 per-deployment only)

## 11. Migration plan
Add migration `pipeline_timeline TEXT` column to deployments table (nullable, backward compatible).

## 12. Wireframes / Diagrams
```
Deployment Timeline (GET /api/deploy/:id)
┌──────────────────────────────────────────────────────────────────┐
│  Clone      ████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  12.3s        │
│  Build      ░░░░░░░░████████████████████████████████  45.1s      │
│  Start      ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░██  2.1s      │
│  Health     ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░██  5.0s      │
│  Running    ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  ✓    │
│                                                                  │
│  Total: 64.5s                                                    │
└──────────────────────────────────────────────────────────────────┘
```

## 13. API / Data Model
```json
// GET /api/deploy/:id — add pipeline_timeline field
{
  "id": "abc123",
  "status": "running",
  "pipeline_timeline": {
    "clone_start": "2026-07-08T12:00:00Z",
    "clone_end": "2026-07-08T12:00:12Z",
    "build_start": "2026-07-08T12:00:12Z",
    "build_end": "2026-07-08T12:00:57Z",
    "start_start": "2026-07-08T12:00:57Z",
    "start_end": "2026-07-08T12:00:59Z",
    "health_start": "2026-07-08T12:00:59Z",
    "health_end": "2026-07-08T12:01:04Z"
  }
}
```

## 14. Affected code
| File | Change |
|------|--------|
| `components/deploy/deploy.go` | Add `StageTimeline` struct, instrument clone/build/start/health/run methods |
| `components/deploy/deploy.go` | Add `ensureStageTimelineColumn` migration |
| `components/deploy/deploy_test.go` | Test timeline populated across deployment lifecycle |
| `components/deploy/deploy.go` — `HandleList` / `handleDeployByID` | Include stage_timeline in JSON response |

## 15. Testing strategy
- **Unit**: Instrumentation test — trigger deploy, verify timeline has expected stages
- **Unit**: Test partial timeline when deployment fails mid-build
- **Contract**: Verify JSON schema of stage_timeline in response

## 16. Rollback plan
Drop `stage_timeline` column or set to NULL. No functional impact.

## 17. Acceptance Criteria
```gherkin
Scenario: Full pipeline timeline
  Given a deployment completes successfully
  When GET /api/deploy/:id is called
  Then response includes stage_timeline with clone, build, start, health timestamps
  And all stage durations are positive

Scenario: Partial timeline on failure
  Given a deployment fails during build stage
  When GET /api/deploy/:id is called
  Then stage_timeline includes clone and build_start
  And build_end, start, health are null
```

## 18. Implementation Steps (see e72s02-tasks.yaml)

## 19. Verification Script (for manual UAT)
1. `go run . serve --port 9999 --db :memory:`
2. Deploy a test app: `bigbase deploy --repo <id>`
3. `curl http://localhost:9999/api/deploy/<id> | jq .stage_timeline`
4. Verify all stages present with non-zero durations

## 20. Out of scope
- Per-stage resource metrics (CPU/mem during build)
- Timeline comparison across multiple deployments
- Real-time SSE streaming of stage transitions (exists via deploy logs, not timeline-specific)
