# Agente_GCP — Proposal

> **Chrome Extension + Backend Agent para Google Workspace + BigQuery**
> Chatbot integrado al navegador que consulta datos auditados de DevDBT, ejecuta modelos ML (BQML + Gemma en Cloud Run), y automatiza tareas de Google Workspace (Gmail, Meet, Chat, Sheets).

---

## 1. The Problem

El stack de datos actual (DevDBT) produce datos auditados en BigQuery, pero:
- No hay forma **quick-access** de consultarlos desde el navegador
- Las tareas de Workspace (Sheets, Gmail, Chat) son manuales
- Los modelos ML (BQML, Gemma) no tienen interfaz directa
- Las decisiones no están conectadas a datos en tiempo real

## 2. Solution

### Chrome Extension (Frontend)
- Chatbot flotante integrado al navegador
- Consultas en lenguaje natural al dataset de DevDBT
- Exportación de datos a Google Sheets con 1 clic
- Resúmenes de Gmail/Meet/Chat vía Workspace APIs
- Conexión al backend local (OpenCode Go) o cloud (Cloud Run)

### Backend (Go + OpenCode)
- API REST/WebSocket para la extension
- Conexión BigQuery: queries SQL + BQML inferencia
- Gemma model serving autoescalado en Cloud Run
- Google Workspace APIs (Sheets, Gmail, Chat, Meet)
- Modo local: OpenCode Go + modelo Gemma-like (offline)

### ML Layer
- **BQML**: modelos entrenados sobre datos de DevDBT
- **Gemma**: fine-tuning + serving en Cloud Run (autoescalado)
- **Modo local**: modelo similar a Gemma para desarrollo

## 3. Architecture

```
┌─────────────────────────────────────────────────────┐
│              Chrome Browser                           │
│  ┌────────────────────────────────────────────────┐ │
│  │  Agente_GCP Extension                           │ │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────────────┐ │ │
│  │  │ Chatbot │ │ Sidebar │ │ Quick Actions    │ │ │
│  │  │ UI      │ │ Panels  │ │ Export · Sheets  │ │ │
│  │  └────┬────┘ └─────────┘ └──────────────────┘ │ │
│  └───────┼────────────────────────────────────────┘ │
└──────────┼──────────────────────────────────────────┘
           │ WebSocket / REST
┌──────────▼──────────────────────────────────────────┐
│              Backend (Go / OpenCode)                  │
│  ┌──────────┐ ┌──────────┐ ┌────────────────────┐  │
│  │ API      │ │ BigQuery │ │ Google Workspace   │  │
│  │ Server   │ │ Connector│ │ Gmail · Sheets ·   │  │
│  │          │ │ + BQML   │ │ Chat · Meet        │  │
│  └──────────┘ └──────────┘ └────────────────────┘  │
│  ┌──────────────────────────────────────────────┐  │
│  │  ML Engine                                   │  │
│  │  ┌──────────┐  ┌──────────┐  ┌────────────┐ │  │
│  │  │ BQML     │  │ Gemma    │  │ Local      │ │  │
│  │  │ Modelos  │  │ Cloud Run│  │ (OpenCode) │ │  │
│  │  └──────────┘  └──────────┘  └────────────┘ │  │
│  └──────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────┘
           │
┌──────────▼──────────────────────────────────────────┐
│              Google Cloud Platform                   │
│  ┌──────────┐ ┌──────────┐ ┌────────────────────┐  │
│  │ BigQuery │ │ Cloud Run│ │ Workspace APIs     │  │
│  │ DevDBT   │ │ Gemma    │ │ Gmail · Sheets ·   │  │
│  │ datasets │ │ serving  │ │ Chat · Meet · Drive│  │
│  └──────────┘ └──────────┘ └────────────────────┘  │
└────────────────────────────────────────────────────┘
```

## 4. Components

### Chrome Extension
| Feature | Tech | Status |
|:--------|:-----|:-------|
| Chatbot UI | React/Preact + Tailwind | Plan |
| Sidebar panel | Chrome Extension MV3 | Plan |
| BQ query by NL | WebSocket → Backend | Plan |
| Export to Sheets | WorksAPI Sheets v4 | Plan |
| Gmail/Meet summaries | WorksAPI Gmail | Plan |

### Backend (Go)
| Module | Description |
|:-------|:------------|
| `api/` | REST + WebSocket endpoints |
| `bq/` | BigQuery client + BQML inference |
| `workspace/` | Google Workspace API integration |
| `ml/` | Gemma client (Cloud Run) + local model |
| `auth/` | OAuth 2.0 for Workspace + GCP |

### ML Models
| Model | Platform | Training Data |
|:------|:---------|:--------------|
| FCR Prediction | BQML | silver_fcr_* tables |
| NPS Score | BQML | silver_NPS_smb |
| Agent Performance | BQML | gold RENDIMIENTO |
| NL to SQL | Gemma (Cloud Run) | Custom prompts |
| Anomaly Detection | Gemma + BQML | DevDBT metrics |

## 5. Development Phases

### Phase 1: Foundation
- [ ] Backend Go API + BigQuery connector
- [ ] Chrome extension skeleton (MV3)
- [ ] NL → SQL basic queries
- [ ] OpenCode Go local mode

### Phase 2: Workspace Integration
- [ ] Google OAuth 2.0
- [ ] Sheets export (create/append)
- [ ] Gmail read/summarize
- [ ] Chat bot integration

### Phase 3: ML
- [ ] BQML models on DevDBT data
- [ ] Gemma serving on Cloud Run
- [ ] NL → SQL with Gemma
- [ ] Predictive analytics

### Phase 4: Production
- [ ] Auto-scaling Cloud Run
- [ ] Auth (Google Identity)
- [ ] Chrome Web Store
- [ ] Monitoring + logging

## 6. Stack

| Component | Production | Local Dev |
|:----------|:-----------|:----------|
| Backend | Go + Cloud Run | Go + OpenCode |
| Extension | Chrome MV3 | Chrome unpacked |
| DB/ML | BigQuery + BQML + Gemma | BigQuery dev1pruebas + local model |
| Auth | Google OAuth 2.0 | Service Account |
| Workspace | APIs v4 (Gmail, Sheets, Chat, Meet) | Same (dev project) |
| Model local | — | Gemma-like (OpenCode Go) |
