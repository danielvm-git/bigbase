# Story e30s02: Webhook/Telegram Messaging Provider

**type:** feat
**context:** domain
**epic:** e30 — Backend for Bots & Integrations
**bcps:** 2
**status:** planned
**wsjf:** 6.5 (BV=5 TC=5 RR=3 / JS=2)
**scope:** `components/messaging/`, `main.go`

## Context

BigBase's Messaging component has channels for email, SMS, and push, but the provider map is empty — no providers are registered. For chatbot use cases (Telegram bots, custom webhooks), there's no way to send messages to external services. The component already has a clean `Provider` interface and `RegisterProvider()` method — we just need to implement a provider and wire it.

This story adds a generic `WebhookProvider` (POST JSON to a URL with Bearer token) and registers it as the `telegram` channel, with a dedicated `/api/messaging/telegram` endpoint that accepts Telegram Bot API payloads.

## Scope boundaries

- **In scope**: `WebhookProvider` implementing `Provider`, `telegram` channel registration, CLI flags (`--messaging-webhook-url`, `--messaging-webhook-token`), `POST /api/messaging/telegram` handler
- **Out of scope**: Other messaging channels (Discord, Slack), provider configuration hot-reload, Telegram-specific response parsing (send only)

## Acceptance Criteria (§17)

### AC1: WebhookProvider sends JSON POST
**Given** a WebhookProvider configured with URL and token
**When** `Send()` is called with a Message
**Then** an HTTP POST is made to the URL with `Authorization: Bearer <token>` header and JSON body `{channel, to_addr, body}`. On success, status remains "sent". On failure, status is "failed".

### AC2: Telegram route accepts chat_id + text
**Given** a POST to `/api/messaging/telegram` with `{"chat_id": "123", "text": "Hello", "token": "bot-token"}`
**When** a `telegram` provider is registered
**Then** a POST is made to `https://api.telegram.org/bot<token>/sendMessage` with `{chat_id, text}`

### AC3: Provider registration via config
**Given** BigBase started with `--messaging-webhook-url=https://example.com/webhook --messaging-webhook-token=abc`
**When** a message is sent via the `telegram` channel
**Then** it's routed through the WebhookProvider

## Out of scope

- Other channel providers (Discord, Slack, WhatsApp)
- Inbound webhook handling (this is outbound-only)
- Provider hot-reload at runtime

## Risks

| Risk | Detection | Mitigation |
|------|-----------|------------|
| Webhook response time blocks sender | Test with slow server | 10s timeout in http.Client |
| Token leaks in logs | Code review | Never log token; only URL prefix |
| Telegram API changes format | Contract test | Fixed payload shape in test |
