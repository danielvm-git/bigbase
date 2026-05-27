# Slice 8: Forge — "See Issues"

**type:** epic  
**status:** planned  
**verify:** `POST /api/forge/issues` → issue appears in kanban board

## Purpose

Issue tracking, pull requests, kanban boards, and wiki. Full project management tooling integrated with Git repos.

## Scope

- Issues (create, list, update, close, labels, milestones)
- Pull requests (create, review, merge, diff view)
- Kanban boards (drag-and-drop columns: To Do, In Progress, Done)
- Wiki (Markdown pages per repo)
- Comments on issues and PRs
- Labels and milestone management

## Design Decisions

- All data in SQLite (no separate DB needed)
- Issues and PRs linked to `git_repos` via foreign key
- Wiki pages stored as files in `data/wiki/<repo>/` (Markdown)
- Kanban is a view of issues grouped by status
- Authorization: repo-level (read/write/admin)

## Implementation Plan

### components/forge/forge.go

```go
type Forge struct {
    db     *db.DB
    logger Logger
}

type Issue struct {
    ID          int64     `json:"id"`
    RepoID      string    `json:"repo_id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Status      string    `json:"status"` // open, closed
    Priority    string    `json:"priority"` // low, medium, high, critical
    AssigneeID  *int64    `json:"assignee_id"`
    Labels      []string  `json:"labels"`
    Milestone   *string   `json:"milestone"`
    CreatedBy   int64     `json:"created_by"`
    CreatedAt   time.Time `json:"created_at"`
}

type PullRequest struct {
    ID         int64     `json:"id"`
    RepoID     string    `json:"repo_id"`
    Title      string    `json:"title"`
    SourceBranch string  `json:"source_branch"`
    TargetBranch string  `json:"target_branch"`
    Status     string   `json:"status"` // open, merged, closed
    CreatedBy  int64    `json:"created_by"`
    CreatedAt  time.Time `json:"created_at"`
}
```

### API Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/forge/:repo/issues` | List issues |
| POST | `/api/forge/:repo/issues` | Create issue |
| PATCH | `/api/forge/:repo/issues/:id` | Update issue |
| GET | `/api/forge/:repo/issues/:id` | Get issue |
| GET | `/api/forge/:repo/board` | Get kanban board |
| GET | `/api/forge/:repo/wiki/:page` | Get wiki page |
| PUT | `/api/forge/:repo/wiki/:page` | Save wiki page |

## Verify

```bash
# Create issue
curl -X POST /api/forge/myproject/issues \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Bug: login fails","priority":"high","labels":["bug"]}'

# Get kanban board
curl -H "Authorization: Bearer $TOKEN" \
  /api/forge/myproject/board

# Write wiki
curl -X PUT /api/forge/myproject/wiki/Home \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"content":"# Welcome to myproject"}'
```
