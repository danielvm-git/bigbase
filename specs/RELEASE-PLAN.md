# Release Plan — Admin UI Pages

## Estratégia

Construir páginas admin para todos os componentes com API pronta. Cada slice adiciona uma página TypeScript + rota + link na sidebar + CSS. Priorizadas por utilidade imediata.

---

## Slices

### UI-1: Git Repos

| Page | Route | API |
|------|-------|-----|
| Lista de repositórios com create/delete | `/repos` | `GET/POST /api/git/repos`, `DELETE /api/git/repos/{id}` |

- Tabela com ID, nome, branch default, descrição, created_at
- Modal/form para criar novo repo
- Botão de delete com confirmação
- Link para deploy (quando UI-2 existir)

### UI-2: Deployments

| Page | Route | API |
|------|-------|-----|
| Lista + criar deployment | `/deploy` | `GET/POST /api/deploy`, `GET /api/deploy/{id}` |

- Tabela com status (colorido), repo, branch, app type, URL, port, created_at
- Botão "New Deployment" com selector de repo + branch
- Auto-refresh enquanto status = "pending" ou "building"
- URL clicável para abrir o site deployado

### UI-3: Messaging / Notificações

| Page | Route | API |
|------|-------|-----|
| Enviar + histórico de mensagens | `/messaging` | `POST /api/messaging/{email,sms,push}`, `GET /api/messaging/messages` |

- Abas: Email | SMS | Push
- Formulário com campos específicos de cada canal
- Tabela de histórico com canal, destinatário, status, data

### UI-4: Storage / Arquivos

| Page | Route | API |
|------|-------|-----|
| File browser com upload | `/storage` | `GET /api/storage/files`, `POST /api/storage/upload`, `DELETE /api/storage/files/{id}` |

- Tabela com nome, tamanho (formatado), mime type, data
- Upload drag-and-drop ou input file
- Download link
- Delete com confirmação

### UI-5: Functions

| Page | Route | API |
|------|-------|-----|
| CRUD + execução | `/functions` | `GET/POST /api/functions`, `GET/PUT/DELETE /api/functions/{id}`, `POST /api/functions/{id}/run` |

- Tabela com nome, runtime, trigger, timeout, created_at
- Modal/form para criar/editar (name, runtime select, source textarea, trigger, timeout)
- Botão "Run" que executa e mostra resultado/logs inline
- Delete com confirmação

### UI-6: Forge / Issues

| Page | Route | API |
|------|-------|-----|
| Issues + Board Kanban | `/forge` | Todos endpoints forge |

- Seletor de repo no topo
- Aba Issues: lista com filtro por status, create/edit modal
- Aba Board: kanban columns (open, in_progress, review, closed)
- Comentários inline na issue

### UI-7: CI/CD

| Page | Route | API |
|------|-------|-----|
| Workflows + Runs | `/cici` | Todos endpoints cici |

- Seletor de repo no topo
- Workflows: lista, create/edit YAML
- Runs: tabela com status, evento, started_at
- Logs: expandir run para ver output dos steps

---

## Dependências entre páginas

UI-1 (Git) → pré-requisito para UI-2 (Deploy) e UI-6 (Forge)
UI-2 (Deploy) → depende de UI-1 (precisa de repo)
UI-6 (Forge) → depende de UI-1 (issues são por repo)
UI-7 (CICI) → depende de UI-1 (workflows são por repo)

Ordem recomendada: **1 → 2 → 3 → 4 → 5 → 6 → 7**

---

## Padrão de implementação (cada slice)

```
ui/src/pages/
  GitReposPage.tsx       # Novo
  DeployPage.tsx         # Novo
  MessagingPage.tsx      # Novo
  StoragePage.tsx        # Novo
  FunctionsPage.tsx      # Novo
  ForgePage.tsx          # Novo
  CiciPage.tsx           # Novo

ui/src/App.tsx           # +7 rotas
ui/src/Layout.tsx        # +7 links na sidebar
ui/src/index.css         # estilos de cada página (~40-80 linhas cada)
```

Cada página segue o mesmo padrão das existentes:
- `useEffect` + `fetch` para carregar dados
- Estado: `loading`, `error`, `data`
- Tabelas com `key`, scroll horizontal
- Botões de ação (create, delete, refresh)
- Sem dependências externas (sem react-router-dom além do já usado, sem axios, sem UI library)
