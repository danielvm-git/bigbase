# Release Plan

## UI Pages (completed)

All 7 admin UI pages built in a single batch. Each page follows the same
pattern: `useEffect` + `fetch` + `loading`/`error`/`data` state + table.

| Page | Route | Status |
|------|-------|--------|
| Git Repos | `/repos` | done |
| Deployments | `/deploy` | done |
| Messaging | `/messaging` | done |
| Storage | `/storage` | done |
| Functions | `/functions` | done |
| Forge | `/forge` | done |
| CI/CD | `/cici` | done |

## Google OAuth (completed)

Embedded OAuth relay no próprio BigBase. Usuário não precisa registrar
Google OAuth app próprio. Config via `--google-client-id` e
`--google-client-secret`.

| Componente | Mudança |
|------------|---------|
| Auth | `GoogleVerifier` interface, `findOrCreateGoogleUser`, handlers initiate + callback |
| main.go | 2 flags: `--google-client-id`, `--google-client-secret` |
| UI | Botão "Sign in with Google" na LoginPage com divider "or" |

## Dashboard Aprimorado (completed)

- **Stats grid** com contagens de repos, deploys, messages, files, functions
- **Bar charts** CSS-only: deploys por status (running/failed/building/pending) e messages por canal (email/sms/push)
- **Activity feed** com últimos 5 deployments e últimas 5 mensagens
- **Storage total** formatado (KB/MB)
- **System card** com contagem de componentes

## Próximos

- Notificações push/setup de provedores SMTP (email real)
- UI para Monitoring (gráficos, alertas históricos)
