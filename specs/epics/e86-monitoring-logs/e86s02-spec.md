# e86s02 — Live log streaming via SSE + Logs tab subscribe

**type:** feat
**risk:** P1
**context:** domain
**BCPs:** 3

## Summary

Add `GET /api/monitoring/logs/stream` — a push-based SSE channel that broadcasts newly inserted log rows — and make the Logs tab subscribe so entries appear live. Reuses the established push-stream pattern (`eventStream` in events.go: subscribe/unsubscribe/broadcast with buffered channels + drop-on-full) rather than the polling variant in stream.go.

## Context

Monitoring component already has two stream mechanisms: `handleMetricsStream` (stream.go — 5s polling ticker) and `handleSSEEvents` (events.go — push broadcast of event-bus emissions). Log streaming wants push: insert → broadcast. The `eventStream` struct (events.go:24-51) is directly reusable; add a parallel `logStream` field or generalize. Auth: `/api/monitoring/*` sits behind the session-cookie-protected SPA routes, and the UI connects same-origin via `new EventSource(...)` (same as the metrics stream at MonitoringPage.tsx ~L113) — cookies flow, no Authorization header needed.

## Requirements

#### ADDED: Live log streaming endpoint
`GET /api/monitoring/logs/stream` returns `text/event-stream`; each inserted log row is broadcast as `data: {"id","level","message","created_at"}\n\n`; client disconnect stops the subscriber goroutine; route registered so it does not collide with the `/api/monitoring/logs/` catch-all.

#### ADDED: Logs tab subscribes to the live stream
When the Logs tab is active, the UI opens an `EventSource('/api/monitoring/logs/stream')`, prepends incoming entries deduped by `id`, shows a "● Live" indicator while connected, and closes the stream on tab switch/unmount.

## Implementation Steps

1. Backend — add a `logStream *eventStream` (or generalize the existing one) to `Monitoring`; initialize in `Start()`; add `broadcastLog(LogEntry)` that marshals and broadcasts. → verify: `go test ./components/monitoring/... -run TestLogStream -v`
2. Backend — `handleLogStream` SSE handler (copy the headers/keepalive/subscribe-loop from `handleSSEEvents` events.go:68-131); register `mux.HandleFunc("GET /api/monitoring/logs/stream", m.handleLogStream)` BEFORE the `/api/monitoring/logs/` catch-all so the mux resolves the exact path. → verify: `go test ./components/monitoring/... -run TestLogStream -v`
3. Backend — broadcast from `handleLogCreate` (monitoring.go:553) after the INSERT succeeds; also from the e86s04 ingestion seam once it lands. → verify: `go test ./components/monitoring/... -run TestLogStream -v`
4. Backend — stream test (extend `stream_test.go`): connect, POST a log, assert the event arrives with matching id/level/message; assert unsubscribe on client close. → verify: `go test ./components/monitoring/... -run TestLogStream -v`
5. UI — Logs tab opens the EventSource when `tab === 'logs'`, prepends entries deduped by id, closes on tab switch/unmount (mirror the metrics-stream effect, MonitoringPage.tsx ~L113). → verify: `cd ui && npx vitest run src/pages/MonitoringPage.test.tsx`
6. UI — "● Live" indicator bound to `es.readyState === EventSource.OPEN`; hidden when connecting/closed. → verify: `cd ui && npx vitest run src/pages/MonitoringPage.test.tsx && cd ui && npm run build`

## Verification Script (Step-by-Step)

1. Open Monitoring → Logs tab → "● Live" appears.
2. POST a log (curl `POST /api/monitoring/logs`) → row appears in the table within ~1s, no manual refresh.
3. Switch to Overview tab → EventSource closes (no state updates for new POSTs).
4. Return to Logs tab → "● Live" returns and streaming resumes.

## Out of scope

- Org-scoped stream filtering (e86s03) — stream broadcasts all rows until s03 lands, then filters per subscriber org.
- Backfill on subscribe — stream only delivers rows inserted after connect; initial page comes from `GET /api/monitoring/logs`.

## Risks

- SSE + cookie auth: EventSource cannot set headers — session cookie must be SameSite-compatible (existing metrics stream already proves the pattern works).
- Channel backpressure: drop-on-full (events.go:47-50) loses entries for slow clients — acceptable for a live log view (matches event-visualizer behavior).
- Route collision: `/api/monitoring/logs/stream` must be registered as an exact method+path pattern before the catch-all, or Go's mux resolves the more specific pattern anyway (1.22+ ServeMux).

## Acceptance Criteria

- [ ] Stream delivers a POSTed log to a connected client
- [ ] Client disconnect cleans up the subscriber
- [ ] UI appends live rows deduped by id; closes on tab switch
