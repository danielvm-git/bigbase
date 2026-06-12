# BigBase Observability Stack

Stack completa de monitoramento para o BigBase rodando em VPS.

## Componentes

| Serviço | Porta local | Função |
|---------|-------------|--------|
| Grafana | 3000 | Dashboards de métricas e logs |
| Prometheus | 9090 | Armazenamento de séries temporais |
| Loki | 3100 (interno) | Agregação de logs |
| Promtail | — | Coleta logs dos containers → Loki |
| Uptime Kuma | 3001 | Uptime dos sites deployados |

O BigBase já expõe nativamente:
- **Métricas Prometheus** em `GET /api/monitoring/metrics/prometheus`
- **Logs JSON estruturado** no stdout (campos: `level`, `msg`, `component`, `request_id`)

## Setup

```bash
cd infra/observability

# 1. Copie as variáveis de ambiente
cp .env.example .env
# Edite .env com suas credenciais

# 2. Suba a stack
docker compose up -d

# 3. Verifique
docker compose ps
```

## Acessando

Todas as portas ficam em `127.0.0.1` por padrão. Exponha via nginx ou Caddy com autenticação.

**Grafana** → http://localhost:3000  
Usuário/senha: os que você definiu no `.env`  
O dashboard "BigBase" é provisionado automaticamente.

**Uptime Kuma** → http://localhost:3001  
Primeiro acesso cria a conta admin. Adicione os sites do BigBase como monitores HTTP(s).

## Configurando o BigBase como container

Para o Promtail capturar os logs, o container do BigBase precisa ter a label `com.bigbase=true`:

```yaml
# No docker-compose do BigBase
services:
  bigbase:
    image: ...
    labels:
      - "com.bigbase=true"
```

Ou, se rodar o BigBase diretamente (sem Docker), redirecione o stdout para um arquivo e ajuste o `__path__` no `promtail/promtail.yml`.

## Prometheus — verificar scrape

Acesse http://localhost:9090/targets — o alvo `bigbase` deve aparecer como `UP`.

Se estiver em `DOWN`, verifique:
1. O BigBase está rodando em `host.docker.internal:9999`?
2. O endpoint `/api/monitoring/metrics/prometheus` responde?

```bash
curl http://SEU_IP:9999/api/monitoring/metrics/prometheus
```

## Uptime Kuma — adicionar monitores

1. Acesse http://localhost:3001
2. "Add New Monitor" → tipo HTTP(s)
3. Adicione cada site deployado no BigBase
4. Configure notificações (email, Telegram, Discord, etc.)

## Alertas

O BigBase já tem um sistema de alertas nativo (`/api/monitoring/alerts`) para métricas de host (CPU, memória, disco). O Prometheus pode complementar com alertas sobre latência e taxa de erros usando AlertManager — adicione ao `docker-compose.yml` se necessário.

## Retenção de dados

- **Prometheus**: 30 dias (ajuste `--storage.tsdb.retention.time` no compose)
- **Loki**: sem limite configurado por padrão — ajuste via `local-config.yaml` se necessário
- **Uptime Kuma**: histórico de 90 dias por padrão
