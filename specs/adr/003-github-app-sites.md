# ADR 003 — GitHub App for Sites deploy

## Status

Accepted

## Context

BigBase Sites needs to list private GitHub repositories, mirror them into internal git, and redeploy on push—similar to Appwrite Sites.

## Decision

- Use a **GitHub App** (not user OAuth alone) for repo listing, clone tokens, and webhooks.
- Mirror with `git clone --bare` into `data/git/{id}.git` and register `git_repos`.
- Push deploys via kernel event **`onGitHubPush`**; **deploy** component subscribes and calls `TriggerDeployment`.
- **Sites** component exposes `/api/sites` and delegates deploy to `TriggerDeploy` injected from `main.go` (composition root).

## Configuration

```
--github-app-id
--github-app-slug
--github-app-private-key-path
--github-webhook-secret
```

Webhook URL: `POST /api/github/webhook` (no auth; HMAC verified when secret set).

## Consequences

- Self-hosters must create a GitHub App once.
- Without flags, GitHub endpoints return empty/disconnected state; UI degrades to BigBase git repos and preview fixtures.
