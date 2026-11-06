# Async tests — poll, never assert on timing

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

### Async tests — poll, never assert on timing
After launching a goroutine, supervisor, or anything that changes state
asynchronously, **poll for the expected state before asserting** — never assert
once and hope the timing lined up. A bare `go func()` followed by
`require.Equal` is the recurring flake class in this repo (22 race/flake fixes
in history).

- Use a `waitForX` helper (or `require.Eventually` / a channel sync) that retries
  with a short interval up to a timeout, then assert.
- Canonical example: `waitForHostRegistration` in the deploy component tests
  (commit `336614804`) — copy that shape rather than sprinkling `time.Sleep`.
- Audit-trail example: `waitForStatusHistory` (`components/deploy/drain_test.go`,
  e90s04). `TransitionState` commits `status` before appending
  `status_history`, so status and history are eventually consistent for
  readers — poll for the transition, never read history once after seeing the
  status.
- Prove it: `go test -race -count=20 ./...` must run without flakes.
