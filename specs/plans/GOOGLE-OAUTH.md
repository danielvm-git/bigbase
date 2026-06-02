# Google Social Login (estilo Neon) — Implementation

## Architecture

Embedded OAuth relay no próprio BigBase. As credenciais OAuth do Google
(client_id + client_secret) são configuradas via flags de linha de comando.
O usuário não precisa registrar um Google OAuth app próprio.

```
Browser → BigBase (/api/auth/oauth/google) → accounts.google.com → consent
         → callback → BigBase troca code por id_token → cria/vincula user → JWT
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/auth/oauth/google` | Redirect to Google OAuth consent screen (404 if no config) |
| GET | `/api/auth/oauth/google/callback?code=...` | Exchange code for user info, create/link user, set JWT cookie |

## Config

```
go run . serve --port 9999 \
  --google-client-id "xxx.apps.googleusercontent.com" \
  --google-client-secret "GOCSPX-..."
```

Sem flags, o botão não aparece e os endpoints retornam 404.

## Implementation

- `GoogleVerifier` interface for mockable Google token verification
- `realGoogleVerifier` calls `https://oauth2.googleapis.com/token` with the code
- `findOrCreateGoogleUser` matches by google_id first, then by email (links accounts)
- Fallback: se Google API falhar, retorna 401 com "google auth failed"
- DB migration: adiciona colunas `google_id` e `avatar_url` na tabela `users`

## Tests

| Test | Description |
|------|-------------|
| `TestGoogleCallbackCreatesUser` | Mock Google callback cria novo user com google_id |
| `TestGoogleCallbackLinksExistingUser` | Mock Google callback vincula google_id a user existente (mesmo email) |
| `TestGoogleOAuthDisabledWhenNoConfig` | Endpoint retorna 404 quando sem config |
