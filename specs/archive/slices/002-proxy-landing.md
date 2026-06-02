# Slice 2: Proxy + Landing Page — "See It in the Browser"

**type:** epic  
**status:** done  
**verify:** `curl http://localhost:9999/api/ping` → `{"status":"alive","version":"0.1.0"}`

## Purpose

HTTP server with routing, landing page, and health endpoint. First visual proof the server runs.

## Implementation

### components/proxy/proxy.go

- **HTTP server** — `net/http` on configurable port (default `8080`)
- **Routes** — `Handle(pattern string, handler func)` for sub-path registration
- **Landing page** — HTML dashboard at `/` showing component table with live status colors
- **Health endpoint** — `/health` returns JSON `{"status":"alive","version":"0.1.0"}`
- **Logging** — Request logging middleware wrapping each handler

### Lifecycle

```
Init → validates port config
Start → http.ListenAndServe
Stop → graceful shutdown via ctx
```

## Configuration

```jsonc
{ "port": 8080, "domain": "localhost", "ssl": false }
```

## Verify

```bash
go run . serve --port 9999 &
curl http://localhost:9999/health
curl http://localhost:9999/api/ping
```

## Files

```
components/proxy/
└── proxy.go
```
