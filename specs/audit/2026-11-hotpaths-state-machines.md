# Auditoria — Hot paths, async sequencial e máquinas de estado

Data: 2026-11-03
Branch: `audit/hotpaths-state-machines`
Escopo: Go (kernel + components) e UI React (`ui/src`, `packages/`, `agentic-cps/`).

## 1. Método

1. Varredura de todos os call sites de I/O (`fetch`, `http.Client.Do`, `QueryContext`) à
   procura de awaits/loops sequenciais que pudessem ser concorrentes.
2. Extração de todos os `CREATE TABLE` com colunas `status`/`state`/`cert_status` e de
   todos os literais de estado gravados no código (fonte da verdade real).
3. Comparação entre as máquinas de estado implícitas e as explícitas, incluindo specs
   (`specs/epics/e45-drain.yaml`) e comentários de código.
4. Testes RED→GREEN para cada correção; suíte completa executada no final.

## 2. Hot paths — async sequencial

### 2.1 Corrigido

| Local | Antes | Depois | Impacto |
|---|---|---|---|
| `ui/src/pages/SiteDetailPage.tsx` (`load`) | `await getSite` → `await getSiteDeployments` → `await getRollbackEvents` | `Promise.all([...])` | 3 RTT → 1 RTT. É hot path: esse `load` roda a cada 2s enquanto há deploy ativo. |
| `ui/src/pages/CreateSitePage.tsx` (`init`) | `await getGitHubStatus` → `await getGitHubRepos` → `await getGitRepos` | `Promise.all([...])` | 3 RTT → 1 RTT no bootstrap da página. |

Testes de concorrência adicionados (provam que os três começam antes de qualquer um
resolver, com fail-safe de 100 ms): `SiteDetailPage.test.tsx` e
`CreateSitePage.test.tsx`.

### 2.2 Já estava correto

- `ui/src/pages/DashboardPage.tsx`, `CiciPage.tsx`, `ForgePage.tsx` já usam `Promise.all`.
- `fetch(...).then(r => r.json())`: sequencial por necessidade (o body depende da resposta).

### 2.3 Avaliado e mantido sequencial (intencional)

| Local | Motivo |
|---|---|
| `ui/src/components/SiteEnvVarsTab.tsx` (import em massa) | Precisa do resultado por chave para agregar erros; SQLite tem writer único, paralelizar só enfileira no servidor. Se o volume crescer, criar endpoint bulk no servidor e usar concorrência limitada no cliente. |
| `ui/src/lib/secretsData.ts` (`importEnvSecrets`) | Mesmo motivo; o resultado reporta `failedKeys` por chave. |
| `components/deploy/health.go` | Retry com backoff: sequencial por definição. |
| `components/internal/llm/llm.go` | Retry com backoff. |
| `components/deploy/deploy_runner.go` | Start de background processes: ordem importa. |
| `kernel/eventbus` | Hooks são despachados por prioridade: a ordem é contrato. |

### 2.4 Oportunidade adicional (não corrigida)

`getSiteDeployments(siteId)` (`ui/src/lib/sitesData.ts`) chama `getSite(siteId)`
internamente só para obter `git_repo_id`. Em `SiteDetailPage`, `getSite` é chamado duas
vezes por `load`. Passar `repoId` como parâmetro elimina uma requisição por ciclo de
polling (a cada 2s durante deploys).

### 2.5 Go — N+1 / fan-out

Nenhum loop com `QueryContext`/`QueryRowContext`/`http.Do` foi encontrado em código de
produção fora de retries intencionais. Não há fan-out sequencial a corrigir.

## 3. Máquinas de estado implícitas (inventário)

| # | Máquina | Fonte da verdade | Estados | Terminal | Onde é decidida |
|---|---|---|---|---|---|
| 1 | **Deployments** | `components/deploy/state_machine.go` + tabela `deployments` | `pending → building → deploying → running → draining → stopped → running`; `failed`, `rolled_back` | `failed`, `rolled_back` | Explícita (`StateMachine.CanTransition`), mas com escapes (ver §5) |
| 2 | **CICI runs** | `components/cici/runs.go` + `cici_runs` | `pending` (default nunca usado), `running`, `success`, `failure` | `success`, `failure` | Implícita: strings literais em `triggerRun`/`finishRun`; sem validação de transição |
| 3 | **Secret key rotation** | `components/secrets/schema.go` (`CHECK`) + `rotation.go` | `active`, `rotating`, `retired` | `retired` | Semi-explícita (CHECK no banco), transições em Go |
| 4 | **Legacy migration checkpoint** | `components/backup/legacy_migration.go` | `in_progress`, `done` | `done` | Implícita (`markCheckpoint`) |
| 5 | **Monitoring incidents** | `components/monitoring/observability.go` | aberto (`resolved_at IS NULL`) / resolvido (`resolved_at` preenchido) | resolvido | Implícita; `status=open` é filtro de query |
| 6 | **Org invites** | `components/auth/membership.go` (`org_invites`) | pendente (`accepted_at IS NULL`) / aceito / expirado | aceito/expirado (por `expires_at`) | Implícita |
| 7 | **Password reset** | `components/auth/passwordreset.go` | não usado / usado / expirado | usado/expirado | Implícita (`used`, `expires_at`) |
| 8 | **Refresh tokens** | `components/auth/refreshtoken.go` | ativo / usado / expirado / família revogada (reuse detection) | usado/expirado/revogado | Implícita |
| 9 | **API/site keys** | `components/auth/apikeys.go` | ativo / revogado | revogado | Implícita (`revoked` 0/1) |
| 10 | **Email verify / magic link / OTP** | `components/auth/{emailverify,magiclink,otp,phone}.go` | pendente / verificado / expirado / bloqueado por tentativas | verificado/expirado | Implícita (token + `expires_at`/`attempts`; magic link/OTP em memória) |
| 11 | **Domain TLS cert** | `components/sites/domains.go` | `provisioning`, `valid`, `expired` | `valid`/`expired` | Derivada do cache de certificado |
| 12 | **Forge issues** | `components/forge/forge.go` | `open`, `closed` | `closed` (reabrível) | Implícita |
| 13 | **Messages** | `components/messaging/messaging.go` | `sent`, `failed` | ambos | Implícita |
| 14 | **Function executions** | `components/functions/functions.go` | `success`, `error` | ambos | Implícita |
| 15 | **UI deployment polling** | `ui/src/pages/SiteDetailPage.tsx`, `CreateSitePage.tsx` | `pending`, `building`, `deploying` = ativo; demais = parado | n/a | Implícita (condição de `setInterval`) |
| 16 | **UI create-site wizard** | `ui/src/pages/CreateSitePage.tsx` | `source → configure → review → deploy` | `deploy` (sucesso/erro) | `useState`, sem máquina explícita |
| 17 | **MCP service status** | `components/mcp/knowledge.go` | lido do registry (`running`/`stopped`) | n/a | Somente leitura |

## 4. Estados não-terminais (risco de "preso para sempre")

| Máquina | Estado não-terminal | Persistido? | Timeout em memória | Reconciliação no boot | Risco |
|---|---|---|---|---|---|
| Deployments | `pending`, `building`, `deploying` | Sim | `runDeployment` tem `context.WithTimeout(5m)` | **Não** — `loadResumeCandidates` só busca `running AND port > 0` | **ALTO**: restart/crash no meio do deploy deixa o registro preso; alguns caminhos negam delete (`409`) e a UI polla para sempre |
| Deployments | `running` com `port = 0` | Sim | — | **Não** (filtro `port > 0`) | **ALTO**: nunca é retomado nem marcado `failed` |
| Deployments | `draining` | Sim | `DrainTimeout` (30s) no goroutine | **Não** | MÉDIO: se o processo morrer durante o drain, fica `draining` |
| Deployments | `stopped` | Sim | — | n/a | BAIXO: semi-terminal reversível (`stopped → running`) |
| CICI runs | `running` | Sim | `context.WithTimeout(10m)` | **Não** | **ALTO**: crash do processo deixa `running` para sempre; `finishRun` engole erro de escrita |
| CICI runs | `pending` | Sim (default) | — | **Não** | BAIXO: estado órfão, nunca usado (o insert grava `running`) |
| Secret rotation | `rotating` | Sim | — | Retomável: `BeginKeyRotation` usa `ON CONFLICT DO NOTHING` + reload | MÉDIO: exige reexecução manual da rotação, mas não perde decryptabilidade |
| Legacy checkpoint | `in_progress` | Sim | — | Reexecução idempotente (checkpoint != done re-roda) | BAIXO |
| Incidents | aberto | Sim | — | Loop de avaliação/resolução | MÉDIO (por design): pode ficar aberto indefinidamente |
| Domain cert | `provisioning` | Derivado | — | Depende do cache externo | MÉDIO: sem timeout explícito |
| Invites / reset / refresh / verify | pendente | Sim | `expires_at` | Validação por expiração | BAIXO: limitado no tempo |

### Resumo dos não-terminais

- **Deployments**: `pending`, `building`, `deploying`, `running`, `draining`, `stopped`
  (apenas `failed` e `rolled_back` são terminais de fato).
- **CICI runs**: `running` (e o órfão `pending`).
- **Secrets**: `rotating` (recuperável).
- **Backup**: `in_progress` (recuperável).
- **Monitoring**: incidente aberto (por design).
- **Domains**: `provisioning`.
- **Tokens/sessões**: pendentes, mas com `expires_at`.

## 5. Conflitos encontrados

### 5.1 Corrigidos

1. **`deploying` ausente nos guards de "deploy ativo"**
   - `components/sites/sites.go` (`hasActiveDeployment`): `IN ('pending','building','running')`
     permitia deletar um site enquanto o deploy estava em `deploying`.
   - `components/deploy/gateway.go` (delete de deployment): `status == "pending" || "building"`
     permitia deletar em `deploying`.
   - Ambos corrigidos + testes de regressão (`409` para `deploying`).

### 5.2 Encontrados e documentados (não corrigidos nesta varredura)

2. **Spec vs. código (e45)**: `specs/epics/e45-drain.yaml` declara `stopped` terminal
   ("no outgoing transitions"), mas `state_machine.go` permite `stopped → running` porque
   `rollback.go` restaura um deployment parado para `running`. Um dos dois está errado.
3. **Comentário vs. código**: `state_machine.go` diz que `running` é o estado terminal, mas
   `running` transiciona para `failed`, `rolled_back` e `draining`. Em
   `state_transitions.go`, `running` e `failed` são tratados como "terminal" apenas para
   gravar `build_log` atomicamente — `draining`/`stopped`/`rolled_back` ficam de fora.
4. **Estado órfão `"replaced"`**: `gateway.go` grava `"replaced"` via `stopDeployment`, mas
   `"replaced"` não existe em `DeploymentState`. O rollback depende do modo leniente
   ("unknown state") para sair de `replaced`. Isso fura a máquina de estados.
5. **Rollback escolhe alvo em andamento**: a query de "previous successful deployment"
   exclui `failed`, `pending`, `building`, mas **não** `deploying` nem `draining`. Em
   zero-downtime, o deployment mais novo pode estar `deploying` e ser escolhido como alvo.
6. **TOCTOU em `TransitionState`** (`state_transitions.go`): o `SELECT status` + validação
   acontece **antes** de `d.mu.Lock()`; o `UPDATE` não tem `WHERE status = ?`. Duas
   transições concorrentes podem validar contra o mesmo estado e ambas gravarem.
7. **`stopDeployment` burla a máquina**: faz `UPDATE deployments SET status = ?` direto,
   sem validação, sem `status_history` e sem evento `deploy.state_changed`.
8. **CICI sem máquina**: `cici_runs` aceita qualquer status; o default `pending` nunca é
   usado; não há validação de transição nem deadline persistido.

## 6. Recomendações priorizadas

**P0**
- Reconciliação no boot para `pending`/`building`/`deploying` (e `running` com `port = 0`)
  mais antigos que N minutos → `failed` com `error_message`.
- Reconciliação no boot para `cici_runs.status = 'running'` órfãos.
- Tornar `TransitionState` atômico (`UPDATE ... WHERE id = ? AND status = ?`, checar
  rows-affected) ou manter o lock durante SELECT + validação + UPDATE.
- Declarar `replaced` (ou removê-lo) e alinhar e45 (`stopped` terminal vs. reversível).

**P1**
- Exportar `IsActive`/`IsTerminal` da máquina de deploy e reutilizar em sites, gateway e
  rollback (hoje cada um repete a lista literal e divergiu).
- Criar máquina explícita para CICI (`queued/running/success/failure/cancelled`) com
  validação e `deadline_at` persistido.
- Timeout/refresh para `cert_status = provisioning`.

**P2**
- Remover a chamada duplicada de `getSite` dentro de `getSiteDeployments`.
- Concorrência limitada nos imports em massa, se/quando existir endpoint bulk.
- Métrica/alerta para "estado não-terminal há mais de X minutos" por máquina.

## 7. Evidência

```
RED (antes da correção):
  sites:  TestDeleteSiteActiveDeploymentConflict/deploying → expected 409, got 204
  deploy: TestDeleteDeployment/409_on_deploying_deployment → expected 409, got 204
  UI:     loads ... concurrently → expected 1 (sequential) to be 3

GREEN (depois):
  go test ./components/sites/ ./components/deploy/ ./components/cici/ -count=1
    → 476 passed in 3 packages
  vitest run
    → 94 files / 740 tests passed
  go build ./...            → exit 0
  go vet (sites, deploy)    → no issues
  tsc -b --pretty false     → exit 0
  eslint (arquivos tocados) → exit 0
```

## 8. Arquivos alterados

- `ui/src/pages/SiteDetailPage.tsx` — `load` concorrente.
- `ui/src/pages/SiteDetailPage.test.tsx` — teste de concorrência + mock de `getSite`.
- `ui/src/pages/CreateSitePage.tsx` — `init` concorrente (removeu `loadGitHub`).
- `ui/src/pages/CreateSitePage.test.tsx` — teste de concorrência.
- `components/sites/sites.go` — `deploying` no guard de site ativo.
- `components/sites/sites_test.go` — caso `deploying`.
- `components/deploy/gateway.go` — `deploying` no guard de delete.
- `components/deploy/deploy_test.go` — caso `deploying`.

## 9. Rastreamento no GitHub

Itens acionáveis abertos como issues (uma por item):

| Issue | Item | Épico vinculado |
|---|---|---|
| #300 | Reconciliação de deployments em voo no boot | e45 |
| #301 | Reconciliação de CICI runs órfãos no boot | e09 |
| #302 | Transições atômicas (TOCTOU) | e45 |
| #303 | Remover estado `replaced` e alinhar e45 | e45 |
| #304 | Centralizar predicados + filtro de rollback | e45 |
| #305 | Máquina de estados explícita do CICI | e09 |
| #306 | Timeout/refresh para `cert_status=provisioning` | e46 |
| #307 | Remover `getSite` duplicado | e15 |
| #308 | Alerta para estado não-terminal antigo | e24 |
| #309 | Concorrência limitada / bulk import | e41 |

Os épicos receberam um campo `follow_ups:` com os números correspondentes.
As duas correções já aplicadas (`deploying` ausente nos guards) não geraram issue.

