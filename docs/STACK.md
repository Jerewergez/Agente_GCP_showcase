# Agente Workspace — Stack Completo

> Stack de herramientas, agentes y configuraciones del ecosistema DevDBT + Agente Workspace.
> Versiones actualizadas al 2 Agosto 2026.

---

## Stack Principal

| Herramienta | Versión | Propósito | Instalación |
|:------------|:--------|:----------|:------------|
| **gentle-ai** | 2.2.4 | Orquestador multi-agente + review (RDD) | `~/.local/bin/gentle-ai` |
| **Engram** | 1.20.0 | Memoria persistente multi-sesión | `~/.local/bin/engram` |
| **Pi** | 0.80.6 | Agente SDD + Frontend | `/usr/bin/pi` |
| **OpenCode** | 1.18.11 | Backend Go executor | `/usr/bin/opencode` |
| **altimate-opencode-plugin** | git (AltimateAI) | Skills dbt + tools `altimate_dbt_*` | `~/.opencode-plugins/altimate-opencode-plugin` (global) |
| **altimate-code** | 0.9.3 | Agente dbt + SQL + Data Engineering | `/usr/local/bin/altimate` |

## Stack de Datos

| Herramienta | Versión | Propósito |
|:------------|:--------|:----------|
| **dbt-core** | 1.11.12 | Transformaciones BigQuery |
| **dbt-bigquery** | 1.11.3 | Adapter BigQuery |
| **BigQuery** | — | Almacenamiento (dev1pruebas) |
| **Airflow** | Docker | Orquestación DAGs |
| **Docker** | 29.6.2 | Contenedores |

## Stack ML

| Herramienta | Propósito |
|:------------|:----------|
| **BQML** | Modelos predictivos (FCR, NPS, TMO) |
| **Gemma** | NL→SQL + inferencia (Cloud Run) |
| **Vertex AI** | Futuro: modelos custom |

## Stack de Infra

| Componente | Detalle |
|:-----------|:--------|
| **VPS** | Contabo · Ubuntu 24.04 · 2 cores · 145 GB |
| **GCP Project** | `dev1pruebas` |
| **Cloud Run** | API Gateway + ML serving (auto-scale 0→100) |
| **Cloud CDN** | Frontend estático Angular |
| **Cloud Tasks** | Async workers |
| **Cloud Monitoring** | Logs + alerts |
| **TokenRouter** | Modelos LLM external (kimi-k3 configurado) |

## Agentes del Ecosistema

```
┌──────────────────────────────────────────────────────────────┐
│                    Hermes Agent (orquestador)                  │
│                                                               │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────┐  │
│  │ Pi       │ │ OpenCode │ │ OpenCode │ │ altimate-code │  │
│  │ SDD + FE │ │ Go (BE)  │ │ Py (ML)  │ │ dbt + SQL     │  │
│  └──────────┘ └──────────┘ └──────────┘ └───────────────┘  │
│                                                               │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────┐  │
│  │ Atalaya  │ │ Agente   │ │ Front    │ │ dbt-power-user│  │
│  │ Watchdog │ │ Workspace│ │ Angular  │ │ VS Code ext   │  │
│  └──────────┘ └──────────┘ └──────────┘ └───────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## Configuración de Agentes

### altimate-code
```bash
# Uso básico (3 modos)
altimate builder    # Modo constructor: implementa modelos dbt
altimate analyst    # Modo analista: revisa y optimiza SQL
altimate plan       # Modo planificador: diseña pipelines

# Con conexión a BigQuery
export GOOGLE_APPLICATION_CREDENTIALS=~/.gcp/dev1pruebas-key.json
altimate --warehouse bigquery
```

### dbt-power-user (VS Code)
- **Instalar**: VS Code → Extensions → buscar "dbt power user"
- **Features**: lineage, health checks, cost estimator, AI docs, SQL validator
- **MCP**: El repo tiene `.mcp.json` para conectar con agentes

### TokenRouter
```bash
export TOKENROUTER_API_KEY="sk-..."
# Config en ~/.hermes/config.yaml:
# custom_providers:
#   tokenrouter:
#     model: kimi-k3
#     base_url: https://api.tokenrouter.com/v1
#     key_env: TOKENROUTER_API_KEY
```

## Repositorios

| Repo | Descripción | Agente |
|:-----|:------------|:-------|
| `Jerewergez/DevDBT` | Pipeline dbt + BQ (28 silver + 16 gold) | Pi |
| `Jerewergez/Agente_Workspace` | Backend Go + Chrome Ext + ML | OpenCode Go |
| `Jerewergez/Agente_Orquestador` | Atalaya watchdog | Pi |
| `Leoo2002/Front` | Frontend Angular 19 | Pi |

## Comandos Rápidos

```bash
# Actualizar todo el stack
gentle-ai version          # 2.2.4
engram version             # 1.20.0
pi --version               # 0.80.6
opencode --version         # 1.18.11
altimate --version         # 0.9.3

# dbt
cd ~/hermes/dbt_a_bigquery
uv run dbt run --select tag:silver-teco
uv run dbt test

# Agente Workspace
cd ~/hermes/Agente_GCP/backend/server
make build && make run
```
