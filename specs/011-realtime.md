# Slice 11: Realtime — "See Live Updates"

**type:** epic  
**status:** planned  
**verify:** `wscat connect ws://localhost:9999/realtime` → receives mutations

## Purpose

WebSocket-based realtime subscriptions. Clients receive live updates when data changes.

## Scope

- WebSocket server at `/realtime`
- Channel subscriptions: collection-level, record-level
- Broadcast on DB mutations (create, update, delete)
- Presence tracking (who's online)
- Automatic reconnection support (client-side)
- Auth required for WebSocket connection

## Design Decisions

- Native `gorilla/websocket` or `nhooyr.io/websocket`
- Subscribe via JSON message: `{"subscribe": "collection:posts"}`
- Messages relayed from DB component via event bus
- Heartbeat ping/pong every 30s
- Max 1000 concurrent connections (configurable)

## Implementation Plan

### components/realtime/realtime.go

```go
type Realtime struct {
    db       *db.DB
    logger   Logger
    hub      *Hub
}

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    rooms      map[string]map[*Client]bool
}

type Client struct {
    conn     *websocket.Conn
    send     chan []byte
    userID   int64
    rooms    map[string]bool
}
```

### Message Protocol

```json
// Subscribe
{"action": "subscribe", "channel": "collection:posts"}

// Unsubscribe
{"action": "unsubscribe", "channel": "collection:posts"}

// Incoming mutation
{"action": "mutation", "channel": "collection:posts", "type": "create", "data": {...}}

// Presence
{"action": "presence", "channel": "collection:posts", "users": [1, 2, 3]}
```

### API Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/realtime` | WebSocket upgrade (requires token query param) |

## Verify

```bash
# Terminal 1: watch for changes
wscat -c "ws://localhost:9999/realtime?token=$TOKEN"
> {"action": "subscribe", "channel": "collection:posts"}

# Terminal 2: create a record
curl -X POST /api/collections/posts \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Hello"}'

# Terminal 1 receives: {"action":"mutation","channel":"collection:posts","type":"create","data":{...}}
```
