---
type: implementation-plan
context: realtime WebSocket subscriptions for live mutation events
---

# Slice 11: Realtime — Implementation Plan

## Prerequisites

| Step | File | Change |
|------|------|--------|
| 1 | `components/auth/auth.go` | Add exported `ValidateToken(token) (*Claims, error)` |
| 2 | `components/auth/jwt.go` | Export `Claims` (already exported) |
| 3 | `components/api/api.go` | Add `Bus` interface to Options; emit mutation events in create/update/delete |
| 4 | `go get github.com/gorilla/websocket` | Add WebSocket library |
| 5 | `components/realtime/realtime.go` | New component: Hub, Client, WebSocket upgrade, event subscription |
| 6 | `components/realtime/realtime_test.go` | Tests: connect, subscribe, broadcast, invalid token |
| 7 | `main.go` | Wire realtime component + auth ValidateToken |

## Event Flow

```
POST /api/collections/posts  →  api.handleCreate()
                                   ↓
                              api.EventBus.Emit("mutation", {collection, type, data})
                                   ↓
                              realtime.receiveMutation()
                                   ↓
                              hub.broadcast to subscribers of "collection:posts"
                                   ↓
                              WebSocket clients receive JSON
```

## Message Protocol (JSON over WebSocket)

Client → Server:
```json
{"action": "subscribe", "channel": "collection:posts"}
{"action": "unsubscribe", "channel": "collection:posts"}
```

Server → Client:
```json
{"action": "mutation", "channel": "collection:posts", "type": "create", "data": {...}}
```

## Verify

```bash
# Terminal 1
wscat -c "ws://localhost:9999/realtime?token=$TOKEN"
> {"action":"subscribe","channel":"collection:posts"}

# Terminal 2
curl -X POST /api/collections/posts \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Hello"}'

# Terminal 1 receives mutation event
```
