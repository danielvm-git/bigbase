# BigBase Evolution Plan — ECC Architecture

> **Produto:** Plataforma única, component-based, single binary
> **Base:** Go (inspirado em PocketBase + Gitea + Caddy)
> **Arquitetura:** Entity-Component-Construct (ECC) — kernel + componentes plugáveis
> **Licença:** MIT

---

## O que é ECC?

ECC é o padrão usado pelo ["Everything Claude Code"](https://github.com/affaan-m/ECC) (affaan-m/ECC) — uma arquitetura onde:

- **Entity** = O sistema em execução (BigBase server)
- **Component** = Submódulo independente com ciclo de vida próprio (auth, db, proxy, git...)
- **Construct** = A composição/configuração que decide quais componentes rodam juntos

```
┌──────────────────────────────────────────────────────────┐
│  Entity: bigbase serve --config bigbase.jsonc            │
│                                                          │
│  Construct (bigbase.jsonc):                              │
│  ┌──────────────────────────────────────────────────┐    │
│  │ { "components": ["proxy","auth","db","git",...] } │    │
│  └──────────────────────────────────────────────────┘    │
│                                                          │
│  Kernel (núcleo mínimo):                                │
│  ┌──────────────────────────────────────────────────┐    │
│  │  • Descoberta de componentes                      │    │
│  │  • Resolução de dependências                      │    │
│  │  • Ciclo de vida (Init → Start → Stop)            │    │
│  │  • Event bus (hooking entre componentes)          │    │
│  │  • Config merge (defaults + user override)        │    │
│  └──────────────────────────────────────────────────┘    │
│                                                          │
│  Componentes carregados:                                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │
│  │ proxy    │ │ auth     │ │ db       │ │ git      │   │
│  ├──────────┤ ├──────────┤ ├──────────┤ ├──────────┤   │
│  │ forge    │ │ storage  │ │ functions│ │ realtime │   │
│  ├──────────┤ ├──────────┤ ├──────────┤ ├──────────┤   │
│  │ messaging│ │ deploy   │ │ admin    │ │ monitoring│   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │
└──────────────────────────────────────────────────────────┘
```

---

## Diferença do Plano Anterior

| Aspecto | Antes (monolítico) | Agora (ECC components) |
|---|---|---|
| **Estrutura** | `internal/` packages fixos | `components/` independentes + kernel |
| **Acoplamento** | Tudo compilado junto | Componentes se comunicam por eventos |
| **Seleção de features** | Tudo ou nada | Escolhe quais componentes rodar |
| **Dependências** | Import direto entre packages | Interface + Event bus |
| **Teste** | Testa o binário inteiro | Testa componente isolado |
| **Plugins externos** | ❌ Não suporta | ✅ WASM ou gRPC futuramente |
| **Perfil de deploy** | Um binário fixo | Compilação com seleção de componentes |

---

## Interface do Componente

```go
// kernel/component.go

type Component interface {
    // Identificação
    Name() string
    Version() string

    // Dependências: outros componentes que precisam estar ativos
    Dependencies() []string

    // Schema JSON do config específico deste componente
    ConfigSchema() json.RawMessage

    // Ciclo de vida
    Init(ctx *Context, config json.RawMessage) error
    Start(ctx *Context) error
    Stop(ctx *Context) error

    // Hooks que este componente escuta/fornece
    Hooks() []HookDef
}

type HookDef struct {
    Name     string   // "onAuth", "beforeRequest", "onMutation"
    Priority int      // ordem de execução
    Handler  HookFunc
}

type Context struct {
    Kernel    *Kernel
    Logger    Logger
    Components map[string]Component  // acesso a outros componentes
    Config    map[string]json.RawMessage
}
```

---

## Catálogo de Componentes

| Componente | Depende de | O que entrega | Referência |
|---|---|---|---|
| **proxy** | — | Reverse proxy, SSL, rate limit, router, dashboard | Caddy, Traefik |
| **auth** | db | Email/senha, 30+ OAuth2, OIDC Provider, MFA, Magic URL, SMS | PocketBase, Nhost |
| **db** | — | SQLite + PostgreSQL, migrations, schema manager | PocketBase, Supabase |
| **api** | db | REST automático, GraphQL auto-generated | Supabase, Directus |
| **storage** | db | Filesystem local + S3, image thumbs, CDN | PocketBase, SeaweedFS |
| **git** | db, auth | Repo hosting (SSH + HTTPS), LFS | Gitea |
| **forge** | git, auth | Issues, PRs, kanban, wiki, milestones | Gitea |
| **cici** | git, forge | CI/CD (Gitea Actions compatível), runner management | Gitea Actions |
| **functions** | db, api | Runtimes (JS/WASM/Python/Go), scheduling, reactive | PocketBase, Convex |
| **realtime** | auth | WebSocket subscriptions, broadcast, presence | PocketBase, Supabase |
| **messaging** | db | Push (FCM/APNs), Email (SMTP), SMS (Twilio/etc) | Appwrite |
| **deploy** | git, proxy | 1-click deploy, DB provisioning, branch preview | Coolify, Neon |
| **admin** | — | Admin UI (SPA embutido), Data Studio, SQL Editor | Supabase Studio, Directus |
| **monitoring** | db | Métricas, logs, uptime, dashboard | Coolify, Grafana |

---

## Estrutura do Projeto

```
bigbase/
├── main.go                      ← entry: kernel.Init() + kernel.Start()
├── go.mod
├── go.sum
│
├── kernel/                      ← Núcleo do ECC
│   ├── kernel.go                ← descoberta, ciclo de vida
│   ├── component.go             ← interface Component
│   ├── eventbus.go              ← hooks entre componentes
│   ├── config.go                ← merge config → componente
│   └── registry.go              ← registro de componentes disponíveis
│
├── components/
│   ├── proxy/
│   │   ├── component.go         ← Init/Start/Stop
│   │   ├── router.go            ← roteamento HTTP
│   │   ├── ssl.go               ← Let's Encrypt
│   │   ├── ratelimit.go
│   │   └── dashboard.go
│   │
│   ├── auth/
│   │   ├── component.go
│   │   ├── oauth2/              ← 30+ providers (Google, GitHub, etc.)
│   │   ├── oidc.go              ← OIDC Provider
│   │   ├── magiclink.go
│   │   ├── mfa.go
│   │   └── sms.go
│   │
│   ├── db/
│   │   ├── component.go
│   │   ├── sqlite.go
│   │   ├── postgres.go
│   │   ├── migrations.go
│   │   └── vector.go            ← pgvector
│   │
│   ├── api/
│   │   ├── component.go
│   │   ├── crud.go              ← REST genérico
│   │   ├── graphql.go           ← resolver automático
│   │   └── middleware.go
│   │
│   ├── storage/
│   │   ├── component.go
│   │   ├── local.go
│   │   ├── s3.go
│   │   ├── thumb.go
│   │   └── cdn.go
│   │
│   ├── git/
│   │   ├── component.go
│   │   ├── ssh.go               ← SSH server para git
│   │   ├── repo.go              ← create/clone/push
│   │   ├── lfs.go
│   │   └── hooks.go             ← git hooks → CI/CD
│   │
│   ├── forge/
│   │   ├── component.go
│   │   ├── issue.go
│   │   ├── pr.go
│   │   ├── kanban.go
│   │   └── wiki.go
│   │
│   ├── cici/
│   │   ├── component.go
│   │   ├── runner.go
│   │   ├── workflow.go          ← parser .github/workflows.yml
│   │   └── executor.go
│   │
│   ├── functions/
│   │   ├── component.go
│   │   ├── runtime_js.go        ← goja (JS/TS)
│   │   ├── runtime_wasm.go      ← wasmtime
│   │   ├── runtime_python.go    ← sidecar
│   │   ├── scheduler.go         ← cron
│   │   └── reactive.go          ← queries reativas
│   │
│   ├── realtime/
│   │   ├── component.go
│   │   ├── subscription.go
│   │   ├── broadcast.go
│   │   └── presence.go
│   │
│   ├── messaging/
│   │   ├── component.go
│   │   ├── push.go
│   │   ├── email.go
│   │   └── sms.go
│   │
│   ├── deploy/
│   │   ├── component.go
│   │   ├── apps.go              ← 1-click deploy (Next.js, API, static)
│   │   ├── databases.go         ← DB provisioning
│   │   └── preview.go           ← branch → URL preview
│   │
│   ├── admin/
│   │   ├── component.go
│   │   └── ui/                  ← React SPA (//go:embed)
│   │       ├── src/
│   │       ├── package.json
│   │       └── dist/
│   │
│   └── monitoring/
│       ├── component.go
│       ├── metrics.go
│       ├── logs.go
│       └── dashboard.go
│
├── config/                      ← configs padrão
│   ├── defaults.jsonc
│   └── profiles/
│       ├── lite.jsonc           ← proxy + db + auth + api + admin + storage
│       ├── pro.jsonc            ← + git + forge + cici + functions
│       └── full.jsonc           ← + deploy + messaging + realtime + monitoring
│
└── ui/                          ← Admin UI source (React)
    ├── src/
    ├── package.json
    └── vite.config.ts
```

---

## Componentes por Perfil

```jsonc
// config/profiles/lite.jsonc  (~60 MB RAM)
{
  "components": ["proxy", "db", "auth", "api", "storage", "admin"],
  "db": { "driver": "sqlite" },
  "proxy": { "domain": "localhost", "ssl": false }
}
```

```jsonc
// config/profiles/pro.jsonc  (~100 MB RAM)
{
  "components": [
    "proxy", "db", "auth", "api", "storage",
    "git", "forge", "cici", "functions", "realtime", "admin"
  ],
  "db": { "driver": "postgres" },
  "auth": { "oidc": true, "providers": ["google","github","email"] }
}
```

```jsonc
// config/profiles/full.jsonc  (~200 MB RAM)
{
  "components": [
    "proxy", "db", "auth", "api", "storage",
    "git", "forge", "cici", "functions", "realtime",
    "messaging", "deploy", "monitoring", "admin"
  ],
  "db": { "driver": "postgres", "vector": true },
  "deploy": { "runtimes": ["node", "python", "go"] }
}
```

---

## Event Bus (Hooks entre Componentes)

Componentes se comunicam por eventos, não por import direto:

```
proxy (recebe request)
  → emite "onRequest" (com método, path, headers)
    → auth escuta "onRequest" e valida token
      → se válido, anexa usuário ao contexto
    → api escuta "onRequest" e roteia pra CRUD
    → storage escuta "onRequest" e serve arquivos
    → monitoring escuta "onRequest" e registra métrica

db (sofre mutation)
  → emite "onMutation" (com coleção, operação, dados)
    → realtime escuta "onMutation" e notifica subscribers
    → functions escuta "onMutation" e dispara hooks JS
    → messaging escuta "onMutation" e envia push/email

git (recebe push)
  → emite "onPush" (com repo, branch, commit)
    → cici escuta "onPush" e dispara workflow
    → deploy escuta "onPush" e faz preview deploy
```

```go
// Exemplo: component realtime escutando mutations do db
component.realtime.Hooks() = []HookDef{
    {
        Name:     "onMutation",
        Priority: 10,
        Handler: func(ctx *Context, event Event) error {
            // event.Data = { collection, operation, record }
            ctx.RealTime.NotifySubscribers(event.Data)
            return nil
        },
    },
}
```

---

## Roadmap com ECC

### Sprint 1-2: Kernel + Componentes Core

| Tarefa | Entrega |
|---|---|
| `kernel/` — interface Component, ciclo de vida, registry | Kernel funcional |
| `components/proxy/` — router + SSL + rate limit | Proxy rodando |
| `components/db/` — SQLite + migrations | Banco funcional |
| `components/auth/` — email + Google GitHub OAuth | Auth funcional |
| `components/api/` — CRUD REST automático | API rodando |
| `components/admin/` — SPA embutido | Admin UI básica |
| `components/storage/` — filesystem local | Upload/download |

### Sprint 3-4: Git + Forge + CI/CD

| Tarefa | Entrega |
|---|---|
| `components/git/` — SSH server + repo management | `git clone/push` funcional |
| `components/forge/` — issues + PRs + kanban | Forja mínima |
| `components/cici/` — workflow parser + runner | CI/CD rodando |

### Sprint 5-6: Functions + Realtime + PostgreSQL

| Tarefa | Entrega |
|---|---|
| `components/functions/` — runtime JS (goja) | `onRequest` + `onCron` hooks |
| `components/realtime/` — subscriptions + broadcast | Tempo real |
| `components/db/` — adicionar PostgreSQL + vector | PG + pgvector |
| `components/api/` — adicionar GraphQL | GraphQL automático |
| `components/auth/` — OIDC Provider + MFA + Magic | Auth completo |

### Sprint 7-8: Deploy + Messaging + Monitoring

| Tarefa | Entrega |
|---|---|
| `components/deploy/` — 1-click apps + DB + preview | PaaS interno |
| `components/messaging/` — push + email + SMS | Notificações |
| `components/monitoring/` — métricas + logs + dashboard | Observabilidade |
| `components/functions/` — runtimes Python + WASM | Multi-runtime |

---

## O que muda com ECC

| Aspecto | Monolítico antigo | ECC novo |
|---|---|---|
| **Adicionar feature** | Criar pasta em `internal/` + mexer em 5 lugares | Criar `components/novo/component.go` + registrar |
| **Desligar feature** | Compila do mesmo jeito | Remove do construct, kernel não carrega |
| **Testar feature** | Sobe o binário inteiro | `component.Init() + component.Start()` isolado |
| **Dependência** | `import` direto (acoplamento) | `EventBus.Emit("onX")` (desacoplado) |
| **Plugin de terceiros** | Editar código fonte | Soltar `.so` ou `.wasm` no `components/` |
| **Perfis de deploy** | Compilação condicional com build tags | `--profile lite` no runtime |
| **Manutenibilidade** | Package cresce sem limites | Kernel ~500 linhas, cada componente ~500-2000 |

---

## Exemplo de Uso

```bash
# Dev — sobe só o essencial
bigbase serve --profile lite
# → proxy + db(sqllite) + auth + api + storage + admin
# → localhost:443 já rodando

# Produção — tudo
bigbase serve --config meuapp.jsonc
# → proxy + db(postgres) + auth + git + forge + cici + functions
#   + realtime + messaging + deploy + monitoring + admin
# → SSL automático, CI/CD, deploy 1-click

# Listar componentes disponíveis
bigbase components list
# proxy v0.1  [proxy, ssl, ratelimit]
# auth  v0.2  [email, oauth2(32), oidc, mfa, magic, sms]
# db    v0.1  [sqlite, postgres, vector]
# ...

# Status do sistema
bigbase status
# proxy  ████████  running  :443
# auth   ████████  running  32 providers
# db     ████████  running  postgres:16
# git    ████████  running  12 repos
# forge  ████████  running  3 open issues
```

---

## Comparação Final (ECC)

| App | Arquitetura | Customizar | Plugins | Perfis | RAM |
|---|---|---|---|---|---|
| **PocketBase** | Monolítico | Hooks JS | JS hooks | ❌ | ~30 MB |
| **Gitea** | Monolítico | Forks | ❌ | ❌ | ~100 MB |
| **Supabase** | 14 containers | Config | ❌ | ❌ | ~800 MB |
| **Appwrite** | 52 containers | Config | ❌ | ❌ | ~14 GB |
| **Coolify** | Monolítico | Config | ❌ | ❌ | ~200 MB |
| **BigBase (ECC)** | Kernel + Components | Selection + Hooks | **WASM futuramente** | **✅ lite/pro/full** | **~60-200 MB** |

Nenhum concorrente tem arquitetura component-based com perfis selecionáveis. É o diferencial.

---

## Próximo Passo

Já tenho o design completo do componente. Quer começar escrevendo o `kernel/` e o primeiro componente (`components/proxy/`) no repositório?
