# Slice 10: Functions — "See Code Run"

**type:** epic  
**status:** planned  
**verify:** `POST /api/functions/run` → function output in response

## Purpose

Serverless function runtime supporting JavaScript (via goja), with event-driven triggers and scheduled execution.

## Scope

- JavaScript runtime using `goja` (pure Go JS engine)
- Function CRUD (create, update, delete, list)
- Function execution (synchronous HTTP + async)
- Event triggers: on DB mutation, on auth event, on file upload
- Scheduled triggers (cron expressions)
- Console logs and error capture
- Timeout and memory limits per function

## Design Decisions

- `goja` for JS runtime (no Node.js dependency, pure Go)
- Functions stored as text in SQLite
- Execution sandbox with timeout (default 30s)
- Console.log captured and returned in response
- Function environment variables configurable
- Max 10 concurrent executions (configurable)

## Implementation Plan

### components/functions/functions.go

```go
type Functions struct {
    db      *db.DB
    logger  Logger
    timeout time.Duration
    maxConcurrent int
}

type Function struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Runtime   string    `json:"runtime"` // "javascript"
    Source    string    `json:"source"`
    Trigger   string    `json:"trigger"` // "http", "event", "cron"
    Schedule  string    `json:"schedule,omitempty"` // cron expression
    Env       map[string]string `json:"env,omitempty"`
    Timeout   int       `json:"timeout"` // seconds
    CreatedAt time.Time `json:"created_at"`
}
```

### API Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/functions` | Yes | List functions |
| POST | `/api/functions` | Yes | Create function |
| GET | `/api/functions/:id` | Yes | Get function |
| PUT | `/api/functions/:id` | Yes | Update function |
| DELETE | `/api/functions/:id` | Yes | Delete function |
| POST | `/api/functions/:id/run` | Yes | Execute function |

### JavaScript API

```javascript
// Available in function context
context = {
    req: { body, headers, query },   // HTTP request
    db: { query, exec },             // SQLite access
    log: { info, warn, error },      // Console logging
    env: { get(key) },               // Environment variables
    storage: { get, put },           // File access
};
```

## Verify

```bash
# Create a function
curl -X POST /api/functions \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name":"hello",
    "runtime":"javascript",
    "source":"context.log.info(\"Hello from function!\"); return {greeting: \"Hello World\"};",
    "trigger":"http"
  }'

# Run it
curl -X POST /api/functions/<id>/run \
  -H "Authorization: Bearer $TOKEN"
```
