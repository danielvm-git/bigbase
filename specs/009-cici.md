# Slice 9: CI/CD — "See Pipelines"

**type:** epic  
**status:** planned  
**verify:** `git push` → workflow runs → green checkmark in admin UI

## Purpose

CI/CD pipeline runner compatible with GitHub Actions workflow YAML format. Executes workflows on Git push events.

## Scope

- YAML-based workflow parser (`.bigbase/workflows/*.yml`)
- Event triggers: `push`, `pull_request`, `schedule`, manual
- Step runners: shell commands, Go, Node.js
- Artifact upload/download
- Pipeline status tracking and logs
- Admin UI dashboard with live logs

## Design Decisions

- Compatible with GitHub Actions YAML syntax
- No container runtime — steps run as shell commands
- Concurrency limits configurable (default 2 parallel jobs)
- Logs stored in SQLite, tailed via WebSocket (realtime)
- Secrets managed via environment variables

## Workflow Format

```yaml
name: Test
on: [push]
jobs:
  test:
    runs-on: self-hosted
    steps:
      - uses: checkout@v1
      - run: go test ./...
      - run: go build .
```

## Implementation Plan

### components/cici/cici.go

```go
type CICI struct {
    db     *db.DB
    logger Logger
    git    *git.Git
    workers int
}

type Workflow struct {
    ID     string `json:"id"`
    RepoID string `json:"repo_id"`
    Name   string `json:"name"`
    YAML   string `json:"yaml"`
}

type PipelineRun struct {
    ID         string    `json:"id"`
    WorkflowID string    `json:"workflow_id"`
    Event      string    `json:"event"`
    Status     string    `json:"status"` // pending, running, success, failure
    CommitSHA  string    `json:"commit_sha"`
    StartedAt  time.Time `json:"started_at"`
    FinishedAt *time.Time `json:"finished_at"`
}
```

### API Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/cici/:repo/workflows` | List workflows |
| PUT | `/api/cici/:repo/workflows` | Save workflow YAML |
| POST | `/api/cici/:repo/workflows/:id/run` | Trigger manual run |
| GET | `/api/cici/runs` | List pipeline runs |
| GET | `/api/cici/runs/:id/logs` | Get run logs |

## Verify

```bash
# Push a workflow file
mkdir -p myproject/.bigbase/workflows
cat > myproject/.bigbase/workflows/test.yml << 'EOF'
name: Test
on: [push]
jobs:
  test:
    runs-on: self-hosted
    steps:
      - run: echo "Hello CI"
EOF

git add . && git commit -m "add workflow"
git push origin main
# Admin UI shows pipeline running → green checkmark
```
