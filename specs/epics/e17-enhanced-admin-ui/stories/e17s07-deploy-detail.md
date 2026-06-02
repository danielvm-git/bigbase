---
id: e17s07
title: Deploy and CICI detail completion
status: in_progress
legacy_slice: "017-G"
tasks:
  - desc: Add TestDeployLogStream to deploy_test.go — test SSE log stream endpoint
    verify: "go test ./components/deploy/ -run TestDeployLogStream -v"
  - desc: Build StreamLog component — terminal-style log viewer with line streaming
    verify: "cd ui && grep -q 'export.*StreamLog' src/components/StreamLog.tsx"
  - desc: Build EnvVarEditor component — key-value pair editor for deployment env vars
    verify: "cd ui && grep -q 'export.*EnvVarEditor' src/components/EnvVarEditor.tsx"
  - desc: Write StreamLog.test.tsx covering log lines, empty state, error state
    verify: "cd ui && npx vitest run src/components/StreamLog.test.tsx"
  - desc: Write EnvVarEditor.test.tsx covering add/remove/edit pairs
    verify: "cd ui && npx vitest run src/components/EnvVarEditor.test.tsx"
  - desc: Write DeployPage.test.tsx covering render, create form, auto-polling
    verify: "cd ui && npx vitest run src/pages/DeployPage.test.tsx"
  - desc: Write CiciPage.test.tsx covering workflow list, YAML editor, run trigger, log viewer
    verify: "cd ui && npx vitest run src/pages/CiciPage.test.tsx"
  - desc: Full verify — Go tests + all 4 frontend test suites
    verify: "go test ./components/deploy/ -run TestDeployLogStream -v && cd ui && npm test -- StreamLog EnvVarEditor DeployPage CiciPage -- --coverage"

context: |
  DeployPage.tsx (143 lines) exists with deployments table + create form +
  auto-polling. CiciPage.tsx (194 lines) exists with workflow CRUD, run trigger,
  YAML editor, log viewer. But StreamLog, EnvVarEditor components don't exist.
  TestDeployLogStream Go test doesn't exist in deploy_test.go (30 other tests
  present). The StatusTimeline already exists inline in SiteDetailPage.tsx, so
  skip it here. CICI DAG visualization is out of scope for this story.
---

## Implementation Steps

**Context**: The verify command `go test ./components/deploy/ -run TestDeployLogStream -v && cd ui && npm test -- StreamLog EnvVarEditor DeployPage CiciPage -- --coverage` currently fails on all 5 counts: no Go test, StreamLog and EnvVarEditor components don't exist, and no frontend tests for DeployPage or CiciPage.

### Phase A: Go backend — log stream test

#### Step 1: Add TestDeployLogStream to deploy_test.go

The deploy component has a log streaming endpoint (`/api/deploy/:id/logs` likely
SSE). Add a Go test that:
- Creates a deployment, then streams its logs via HTTP
- Asserts Content-Type is `text/event-stream`
- Verifies log lines arrive

Follow the pattern of `TestDeployCreateSuccess` in the same file.

→ verify: `go test ./components/deploy/ -run TestDeployLogStream -v`

### Phase B: Frontend components

#### Step 2: Build StreamLog component (`ui/src/components/StreamLog.tsx`)

A terminal-style log viewer that:
- Accepts `logs: string[]` prop
- Renders each line in monospace with line numbers
- Supports `isStreaming: boolean` prop for animated cursor during live streaming
- Shows "No logs" empty state
- Exported via `components/index.ts`

→ verify: `cd ui && grep -q 'export.*StreamLog' src/components/StreamLog.tsx && npm run build`

#### Step 3: Build EnvVarEditor component (`ui/src/components/EnvVarEditor.tsx`)

A key-value pair editor for deployment environment variables:
- Accepts `vars: Record<string, string>` and `onChange: (vars: Record<string, string>) => void`
- Renders rows of key+value inputs
- "Add" button to append a new row
- "Remove" button per row
- Empty state: "No environment variables configured"
- Exported via `components/index.ts`

→ verify: `cd ui && grep -q 'export.*EnvVarEditor' src/components/EnvVarEditor.tsx && npm run build`

### Phase C: Component tests

#### Step 4: Write StreamLog.test.tsx

Following the pattern of `Badge.test.tsx`:
- Renders log lines with monospace class
- Shows "No logs" when array is empty
- Shows animated cursor when `isStreaming` is true
- Renders line numbers correctly

→ verify: `cd ui && npx vitest run src/components/StreamLog.test.tsx`

#### Step 5: Write EnvVarEditor.test.tsx

- Renders existing key-value pairs
- "Add" button inserts a new row
- "Remove" button deletes a row
- onChange fires with updated object on key/value change
- Empty state renders placeholder text

→ verify: `cd ui && npx vitest run src/components/EnvVarEditor.test.tsx`

### Phase D: Page tests

#### Step 6: Write DeployPage.test.tsx

- Renders page header "Deployments"
- Toggle form with "New Deployment" button
- Mock `/api/git/repos` and `/api/deploy` endpoints
- Create deployment form: select repo, set branch, submit
- Error state: shows "create failed" on HTTP error
- Auto-polling: `setInterval` clears on unmount

→ verify: `cd ui && npx vitest run src/pages/DeployPage.test.tsx`

#### Step 7: Write CiciPage.test.tsx

- Renders repo selector initially
- Selecting a repo fetches workflows and runs
- Create workflow form: name + YAML editor
- Run trigger: "Run" button calls `/api/cici/:repoId/workflows/:id/run`
- Log viewer: clicking a run row fetches and shows logs
- Error state: "failed to load" on fetch failure

→ verify: `cd ui && npx vitest run src/pages/CiciPage.test.tsx`

#### Step 8: Full verify

→ verify: `go test ./components/deploy/ -run TestDeployLogStream -v && cd ui && npm test -- StreamLog EnvVarEditor DeployPage CiciPage -- --coverage`

## Verification Script (Manual)

1. `go test ./components/deploy/ -run TestDeployLogStream -v` — green
2. `cd ui && npm run build` — builds cleanly
3. `cd ui && npx vitest run src/components/StreamLog.test.tsx` — all pass
4. `cd ui && npx vitest run src/components/EnvVarEditor.test.tsx` — all pass
5. `cd ui && npx vitest run src/pages/DeployPage.test.tsx` — all pass
6. `cd ui && npx vitest run src/pages/CiciPage.test.tsx` — all pass
7. `cd ui && npm test -- StreamLog EnvVarEditor DeployPage CiciPage -- --coverage` — green with coverage

## Out of scope
- SSE log stream integration into DeployPage (just the component and test)
- CICI DAG visualization (complex feature, needs separate story)
- StatusTimeline extraction to shared component (already inline in SiteDetailPage)
- EnvVarEditor integration into CreateSitePage wizard

## Risks
- deploy_test.go pattern: use `httptest.NewRecorder` + component handler setup
- StreamLog: line streaming from SSE requires EventSource handling; test with mock only
- CiciPage: YAML content area uses `<textarea>`; test with `fireEvent.change`
