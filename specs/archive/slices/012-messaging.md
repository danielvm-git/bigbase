# Slice 12: Messaging — "See Notifications"

**type:** epic  
**status:** planned  
**verify:** `POST /api/messaging/email` → email arrives in inbox

## Purpose

Multi-channel messaging: email (SMTP), push notifications (FCM/APNs), and SMS. Triggerable from functions, events, or admin UI.

## Scope

- Email via SMTP (SendGrid, Mailgun, or any SMTP server)
- Push notifications via Firebase Cloud Messaging (FCM) and Apple Push Notification Service (APNs)
- SMS via Twilio
- Template support (Go templates)
- Message queue with retry
- Delivery status tracking
- Admin UI: send test message, view delivery history

## Design Decisions

- SMTP adapter pattern — swap providers without changing code
- Messages queued in SQLite, sent asynchronously
- Retry with exponential backoff (3 attempts)
- Templates stored in SQLite, rendered with Go `text/template`
- Rate limiting per channel

## Implementation Plan

### components/messaging/messaging.go

```go
type Messaging struct {
    db      *db.DB
    logger  Logger
    email   EmailSender
    push    PushSender
    sms     SMSSender
}

type Message struct {
    ID        string    `json:"id"`
    Channel   string    `json:"channel"` // email, push, sms
    To        string    `json:"to"`
    Subject   string    `json:"subject"`
    Body      string    `json:"body"`
    Template  string    `json:"template,omitempty"`
    Status    string    `json:"status"` // queued, sent, failed
    CreatedAt time.Time `json:"created_at"`
}

type Template struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    Channel string `json:"channel"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}
```

### API Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/messaging/email` | Yes | Send email |
| POST | `/api/messaging/push` | Yes | Send push notification |
| POST | `/api/messaging/sms` | Yes | Send SMS |
| GET | `/api/messaging/messages` | Yes | List sent messages |
| GET | `/api/messaging/templates` | Yes | List templates |
| POST | `/api/messaging/templates` | Yes | Create template |

## Verify

```bash
# Send email (requires SMTP config)
curl -X POST /api/messaging/email \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"to":"user@example.com","subject":"Hello","body":"<h1>Test</h1>"}'

# Send SMS (requires Twilio config)
curl -X POST /api/messaging/sms \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"to":"+1234567890","body":"Hello from BigBase"}'
```
