# Agente Workspace

> **Asistente inteligente para Google Workspace + BigQuery + ML**
> Chatbot + Chrome Extension + Automatización productiva para equipos basados en GCP.

---

## Documentación

- [Stack Completo](docs/STACK.md) — Versiones, agentes, herramientas, comandos
- [Arquitectura](ARCHITECTURE.md) — Diseño de sistema, componentes, flujo de datos
- [Seguridad](docs/SECURITY.md) — OAuth2, IAM, red, compliance
- [Escalado](docs/SCALING.md) — Cloud Run autoescalado, BigQuery, CDN, PWA
- [Costos](docs/COSTS.md) — Estimación de costos operativos en EUR
- [Memoria](docs/MEMORY.md) — Sistema de memoria persistente (Engram-like)
- [Gobernanza](docs/GOVERNANCE.md) — Gestión institucional de datos GCP
- [Roadmap](docs/ROADMAP.md) — Futuras implementaciones y fases

---

## Stack

| Componente | Tecnología | Estado |
|:-----------|:-----------|:-------|
| **Backend** | Go 1.23 + Cloud Run | ✅ Phase 0 |
| **Frontend** | Front (Angular 19) | 🚧 Pendiente |
| **DB/ML** | BigQuery + BQML + Gemma | 🔲 Fase 3 |
| **Workspace** | Gmail · Sheets · Chat · Meet APIs | 🔲 Fase 2 |
| **Extension** | Chrome MV3 + WebSocket | 🔲 Fase 4 |
| **Memoria** | Engram + BigQuery audit log | 🔲 Fase 5 |

## Quick Start

```bash
# Backend
cd backend/server
cp .env.example .env
make build
make run

# Tests
make test

# Frontend (Front)
cd ../Front
npm install
ng serve
```

## Multi-Agent Configuration

Ver [.hermes/agents.md](.hermes/agents.md) para la configuración completa de agentes.

| Agente | Rol | Tecnología |
|:-------|:----|:-----------|
| **Pi** | Orquestador SDD + Frontend | Hermes Agent |
| **OpenCode** | Backend Go + ML | OpenCode Go |
| **Workspace** | APIs + Extension | Chrome MV3 |

---

*Agente Workspace · Jerewergez · 2026*
