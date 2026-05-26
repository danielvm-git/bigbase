# BigBase Report — Análise Comparativa de BaaS Self-Hosted

> **Data:** 26 de Maio de 2026
> **Contexto:** Substituir stack Appwrite + GitLab CE (52 containers, ~14 GB RAM) por alternativa 100% open source, leve, e segura para uso comercial.

---

## Sumário

1. [Objetivo](#1-objetivo)
2. [Apps Estudados](#2-apps-estudados)
3. [Licenças Verificadas](#3-licenças-verificadas)
4. [Arquiteturas Propostas](#4-arquiteturas-propostas)
   - 4.1 BigBase Lite
   - 4.2 BigBase Pro
   - 4.3 BigBase Full MIT
   - 4.4 BigBase Full Apache
5. [Tabela Comparativa Final](#5-tabela-comparativa-final)
6. [Análise de Licenças para Uso Comercial](#6-análise-de-licenças-para-uso-comercial)
7. [Recomendações](#7-recomendações)

---

## 1. Objetivo

Substituir a stack atual (Appwrite + GitLab CE, 52 containers, ~14 GB RAM) por uma combinação de ferramentas:

- **Leve** — rodar em MacBook 8 GB ou VPS de $6/mês
- **100% open source** com licenças permissivas
- **Segura para monetizar** — oferecer como serviço cobrando sem restrições legais
- **Autossuficiente** — git, issues, CI/CD, auth, banco, storage, funções, admin

---

## 2. Apps Estudados

| App | Descrição | Estrelas GitHub | Site |
|---|---|---|---|
| **Gitea** | Git hosting leve com issues, PRs, kanban, wiki, Actions (CI/CD) | ~48k | gitea.com |
| **PocketBase** | BaaS em 1 binário: auth, SQLite, storage, funções JS, realtime, admin UI | ~43k | pocketbase.io |
| **Nhost** | BaaS completo: auth OIDC, PostgreSQL, storage, functions Node, GraphQL (Constellation) | ~8k | nhost.io |
| **Supabase** | Firebase alternativo: PostgreSQL, auth, storage, Edge Functions, Realtime | ~98k | supabase.com |
| **Appwrite** | BaaS tudo-em-um: auth, DB, storage, functions, messaging, hosting | ~56k | appwrite.io |
| **Directus** | Headless CMS + BaaS: REST, GraphQL, qualquer DB, admin Studio | ~36k | directus.io |
| **Coolify** | PaaS self-hosted: deploy apps, DBs, serviços com 1 clique | ~39k | coolify.io |
| **Neon** | Serverless PostgreSQL com branching, autoscale, scale-to-zero | ~16k | neon.tech |
| **Convex** | Reactive database + backend TypeScript nativo, realtime | ~12k | convex.dev |
| **Caddy** | Reverse proxy com Let's Encrypt automático (Caddyfile simples) | ~62k | caddyserver.com |
| **Traefik** | Reverse proxy cloud-native, Docker discovery, Let's Encrypt | ~63k | traefik.io |
| **MinIO** | Armazenamento S3-compatível self-hosted | ~52k | min.io |
| **SeaweedFS** | Armazenamento distribuído S3 + FUSE + Iceberg | ~32k | seaweedfs.com |
| **Gitblit** | Git server Java puro, Apache 2.0 | ~2k | gitblit.com |

---

## 3. Licenças Verificadas

Todas as licenças foram verificadas nos arquivos `LICENSE` ou `LICENSE.md` de cada repositório oficial no GitHub.

| App | Licença | SPDX | Permite cobrar? |
|---|---|---|---|
| **Gitea** | MIT | MIT | ✅ Sim, sem restrições |
| **PocketBase** | MIT | MIT | ✅ Sim, sem restrições |
| **Nhost** | MIT | MIT | ✅ Sim, sem restrições |
| **Traefik** | MIT | MIT | ✅ Sim, sem restrições |
| **Appwrite** | BSD 3-Clause | BSD-3-Clause | ✅ Sim (requer atribuição) |
| **Coolify** | Apache 2.0 | Apache-2.0 | ✅ Sim |
| **Supabase** | Apache 2.0 | Apache-2.0 | ✅ Sim |
| **Neon** | Apache 2.0 | Apache-2.0 | ✅ Sim |
| **Convex** | Apache 2.0 | Apache-2.0 | ✅ Sim |
| **Caddy** | Apache 2.0 | Apache-2.0 | ✅ Sim |
| **SeaweedFS** | Apache 2.0 | Apache-2.0 | ✅ Sim |
| **Gitblit** | Apache 2.0 | Apache-2.0 | ✅ Sim |
| **Directus** | **BSL 1.1** | BSL-1.1 | ⚠️ Limitado: $5M receita, vira GPL v3 em 3 anos |
| **MinIO** | **AGPL v3** | AGPL-3.0 | ⚠️ Risco: modificações precisam ser liberadas |

### Detalhamento das licenças restritivas

**Directus (BSL 1.1):**
- Uso comercial grátis até $5M de "Total Finances" nos últimos 12 meses
- Acima disso, precisa de licença comercial paga
- Após 3 anos da release, o código vira GPL v3 (copyleft forte)
- Licenciante: Monospace, Inc.

**MinIO (AGPL v3):**
- AGPL fecha a brecha ASP do GPL — se você modificar e oferecer como serviço, precisa liberar o código
- Uso sem modificações (Docker image oficial) é interpretação comum como seguro, mas há margem para debate
- MinIO Inc. vende licença comercial justamente para quem não quer o risco
- **MinIO Community Edition foi arquivado em Fev 2026** — não recebe mais atualizações

---

## 4. Arquiteturas Propostas

### 4.1 BigBase Lite

**Stack:** Caddy (Apache 2) + Gitea (MIT) + PocketBase (MIT)

```
┌──────────────┐
│    Caddy     │  Proxy reverso + SSL (Apache 2)
│  :443 → apps │
└──────┬───────┘
       │
       ├── Gitea (:3000)    — git, issues, PRs, kanban, wiki, CI/CD
       └── PocketBase (:8090) — auth, SQLite DB, storage, funções JS,
                                realtime WS, admin UI
```

| Componente | RAM | Função |
|---|---|---|
| **Caddy** | ~20 MB | Proxy, SSL, roteamento |
| **Gitea** | ~100 MB | Git hosting completo |
| **PocketBase** | ~30 MB | BaaS (auth, DB, storage, functions) |
| **Total** | **~150 MB** | **3 containers** |

**Storage:** PocketBase tem filesystem próprio (`tools/filesystem/`) — não precisa de S3.
**Gitea LFS:** usa disco local.
**CI/CD:** Gitea Actions nativo + `act_runner` (MIT), compatível com `.github/workflows`.

### 4.2 BigBase Pro

**Stack:** Caddy (Apache 2) + Gitea (MIT) + Nhost (MIT) + PostgreSQL

| Componente | RAM | Função |
|---|---|---|
| **Caddy** | ~20 MB | Proxy, SSL |
| **Gitea** | ~100 MB | Git hosting completo |
| **Nhost Auth** | ~50 MB | Auth service (Go, 30+ OAuth2 + OIDC server) |
| **Nhost Storage** | ~50 MB | File storage (Go) |
| **Nhost Functions** | ~100 MB | Node.js runtime |
| **Nhost Constellation** | ~80 MB | GraphQL (Go) |
| **PostgreSQL** | ~200 MB | Banco relacional |
| **Total** | **~500 MB** | **5 containers** |

**Diferencial:** Nhost Auth é também um **OIDC provider** — pode registrar Gitea como cliente OAuth do próprio Nhost. Um login único para todo o stack.
**Storage:** usa Backblaze B2 ou AWS S3 como backend (não self-hostear MinIO).

### 4.3 BigBase Full MIT

**Stack:** Traefik (MIT) + Gitea (MIT) + PocketBase (MIT) + SeaweedFS (Apache 2 → **não MIT**)

> **Nota:** Não existe storage S3-compatível sob licença MIT. SeaweedFS é Apache 2.0. Uma stack 100% MIT depende do filesystem embutido do PocketBase e dispensa S3 externo.

| Componente | Licença | RAM |
|---|---|---|
| **Traefik** | MIT | ~25 MB |
| **Gitea** | MIT | ~100 MB |
| **PocketBase** | MIT | ~30 MB |
| **Total** | **100% MIT** | **~155 MB** |

### 4.4 BigBase Full Apache

**Stack:** Caddy (Apache 2) + Gitblit (Apache 2) + Supabase (Apache 2) + SeaweedFS (Apache 2)

| Componente | Licença | RAM |
|---|---|---|
| **Caddy** | Apache 2.0 | ~20 MB |
| **Gitblit** | Apache 2.0 | ~120 MB |
| **Supabase** | Apache 2.0 | ~800 MB (14 containers) |
| **SeaweedFS** | Apache 2.0 | ~80 MB |
| **Total** | **100% Apache 2** | **~1 GB** |

> ⚠️ **Trade-off:** Gitblit é funcional mas não tem kanban, CI/CD nativo, nem wiki. Supabase tem 14 containers contra 1 do PocketBase. A pureza de licença Apache 2.0 custa em complexidade e recursos.

---

## 5. Tabela Comparativa Final

| Dimensão | **BigBase Lite** | **BigBase Pro** | **BigBase Full MIT** | **BigBase Full Apache** | Appwrite | Coolify | PocketBase solo | Supabase | Directus | Nhost | Neon | Convex |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **Licença** | MIT/Apache | MIT/Apache | **MIT puro** | **Apache 2 puro** | BSD-3 | Apache 2 | MIT | Apache 2 | BSL 1.1* | MIT | Apache 2 | Apache 2 |
| **Proxy** | Caddy | Caddy | Traefik | Caddy | — | — | — | — | — | — | — | — |
| **Git hosting** | **Gitea** ✅ | **Gitea** ✅ | **Gitea** ✅ | Gitblit ⚠️ | ❌ | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ |
| **Issues/PRs/Kanban** | **Gitea** ✅ | **Gitea** ✅ | **Gitea** ✅ | Gitblit ⚠️ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **CI/CD nativo** | **Gitea Actions** | **Gitea Actions** | **Gitea Actions** | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | GitHub | ❌ | ❌ |
| **Wiki** | **Gitea** ✅ | **Gitea** ✅ | **Gitea** ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **DB** | SQLite | PostgreSQL 16 | SQLite | PostgreSQL | MariaDB | — | SQLite | PostgreSQL | MySQL/PG | PostgreSQL | PostgreSQL | SQLite/PG |
| **Auth providers** | 30+ OAuth2 | 30+ OAuth2 + **OIDC** | 30+ OAuth2 | 10+ OAuth2 | 30+ OAuth2 | — | 30+ OAuth2 | 10+ OAuth2 | 20+ OAuth2 | 20+ + **OIDC** | — | 80+ |
| **Storage** | Built-in (PB) | S3 externo | Built-in (PB) | SeaweedFS (Ap2) | Built-in + S3 | — | Built-in | Built-in + S3 | Built-in + S3 | Built-in + S3 | — | Built-in |
| **Functions** | JS hooks (82 pts) | Node.js runtime | JS hooks (82 pts) | Deno Edge | Node/Py/Go/etc | Deploy apps | JS hooks (82 pts) | Deno Edge | Node.js | Node.js | SQL queries | TS nativo |
| **Realtime** | WS subscriptions | WS + Hasura | WS subscriptions | Broadcast/Presence | WS subs | — | WS subs | WS Broadcast | WS + GQL | Hasura + WS | — | ✅ Reativo |
| **GraphQL** | ❌ REST | ✅ Constellation | ❌ REST | ✅ Auto REST+GQL | ❌ REST | ❌ | ❌ REST | ✅ Auto | ✅ REST+GQL | ✅ Hasura | ❌ | TS queries |
| **Admin UI** | Gitea + PB Admin | Gitea + Nhost Dash | Traefik + Gitea + PB | Caddy + Gitblit + Studio | Appwrite Console | Coolify UI | PocketBase Admin | Supabase Studio | Directus Studio | Nhost Dashboard | Neon Dashboard | Convex Dashboard |
| **Containers** | **3** | **5** | **3** | **~17** | **52** | 1 + apps | **1 binário** | **14** | 4 | 6 | 2 | 2 |
| **RAM estimada** | **~150 MB** | **~500 MB** | **~155 MB** | **~1 GB** | **~14 GB** | ~200 MB | **~30 MB** | ~800 MB | ~200 MB | ~400 MB | ~300 MB | ~400 MB |
| **VPS mínimo** | **$6/mês** | **$15/mês** | **$6/mês** | **$20/mês** | $50-60/mês | $10/mês | $4/mês | $20/mês | $10/mês | $15/mês | $15/mês | $15/mês |
| **Cobrar permite?** | ✅ **Sim** | ✅ **Sim** | ✅ **100%** | ✅ **100%** | ✅ Sim | ✅ Sim | ✅ Sim | ✅ Sim | ⚠️ $5M cap | ✅ Sim | ✅ Sim | ✅ Sim |

---

## 6. Análise de Licenças para Uso Comercial

### Stack 100% seguro (MIT / Apache 2.0 / BSD-3)
Você pode usar, modificar, distribuir e **cobrar** sem restrições:

- **BigBase Lite** (Caddy Apache 2 + Gitea MIT + PocketBase MIT)
- **BigBase Pro** (Caddy Apache 2 + Gitea MIT + Nhost MIT)
- **BigBase Full MIT** (Traefik MIT + Gitea MIT + PocketBase MIT)
- **BigBase Full Apache** (Caddy Apache 2 + Gitblit Apache 2 + Supabase Apache 2 + SeaweedFS Apache 2)
- Appwrite (BSD-3 — requer atribuição)
- Coolify (Apache 2)
- Supabase (Apache 2)
- Nhost (MIT)
- Neon (Apache 2)
- Convex (Apache 2)

### Stack com ressalvas

| App | Licença | Risco |
|---|---|---|
| **Directus** | BSL 1.1 | Gratuito até $5M de receita. Acima disso precisa de licença paga. Em 3 anos vira GPL v3. |
| **MinIO** | AGPL v3 | Se modificar e oferecer como serviço, precisa liberar código. Community Edition arquivado (Fev 2026). |

---

## 7. Recomendações

### Para dev solo (agora)
**BigBase Lite** — 3 containers, ~150 MB RAM, $6/mês de VPS.
- Gitea cobre git + issues + kanban + CI/CD + wiki
- PocketBase cobre auth + DB + storage + funções + admin
- Caddy faz proxy + SSL automático
- Cresce sem rewrites: quando precisar de PostgreSQL/GraphQL, sobe o Pro

### Para produção com clientes (futuro)
**BigBase Pro** — 5 containers, ~500 MB RAM, $15/mês.
- Nhost substitui PocketBase: PostgreSQL + OIDC server + GraphQL
- Gitea continua como git host + CI/CD
- Storage no Backblaze B2 ($0.006/GB/mês) — sem self-hostear S3

### Se pureza de licença for crítica
**BigBase Full MIT** — Traefik + Gitea + PocketBase.
- Único sacrifício: Trocar Caddy (Apache 2) por Traefik (MIT).
- Na prática, Apache 2.0 e MIT são compatíveis — não há ganho jurídico real.

### Evitar
- **Directus** se planeja faturar >$5M ou não quer risco de licença mudar para GPL v3
- **MinIO** self-hosted se não quer risco AGPL ou não quer comprar licença comercial
- **Appwrite + GitLab CE** atual: 52 containers, ~14 GB RAM, desnecessariamente pesado

---

*Documento gerado em 26 de Maio de 2026.*
*Licenças verificadas nos arquivos LICENSE de cada repositório oficial no GitHub.*
