# Agente_GCP — Architecture for Massive Operational Use

> **Front (Angular) + Go Backend + BigQuery + Cloud Run = Escala operativa masiva**
> Frontend Angular 19 con Supabase migrado a BigQuery + Cloud Run.

---

## Arquitectura de Escala Masiva

```
Users (1000s concurrent)
       │
       ▼
┌─────────────────────────────────────────────────────────┐
│  Cloud CDN (static assets)                              │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Front (Angular 19 PWA)                         │   │
│  │  chrome extension · service worker · offline    │   │
│  └────────────────────┬────────────────────────────┘   │
└───────────────────────┼─────────────────────────────────┘
                        │ HTTPS
┌───────────────────────▼─────────────────────────────────┐
│  Cloud Run (auto-scale 0→N)                            │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Go API Gateway                                  │  │
│  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌────────┐ ┌────┐  │  │
│  │  │Auth  │ │Query │ │ML    │ │Export  │ │Chat│  │  │
│  │  │OAuth2│ │→BQ   │ │→Gemma│ │→Sheets │ │Bot │  │  │
│  │  └──────┘ └──────┘ └──────┘ └────────┘ └────┘  │  │
│  └──────────────────────────────────────────────────┘  │
│                                                         │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Async Workers (Cloud Tasks + Pub/Sub)           │  │
│  │  ┌──────────┐ ┌──────────┐ ┌───────────────┐   │  │
│  │  │Reports   │ │Gmail     │ │ML Training    │   │  │
│  │  │generation│ │campaigns │ │pipeline       │   │  │
│  │  └──────────┘ └──────────┘ └───────────────┘   │  │
│  └──────────────────────────────────────────────────┘  │
└───────────────────────┼─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│  Google Cloud Platform                                   │
│  ┌────────────┐ ┌────────────┐ ┌────────────────────┐  │
│  │ BigQuery   │ │ Cloud Run  │ │ Google Workspace   │  │
│  │ DevDBT     │ │ Gemma      │ │ Gmail · Sheets ·   │  │
│  │ 28 tables  │ │ auto-scale │ │ Chat · Meet · Drive│  │
│  │ + BQML     │ │ + OpenCode │ │                    │  │
│  └────────────┘ └────────────┘ └────────────────────┘  │
│  ┌────────────┐ ┌────────────┐ ┌────────────────────┐  │
│  │ Cloud CDN  │ │ Cloud Tasks│ │ Cloud Monitoring   │  │
│  │ Angular    │ │ + Pub/Sub  │ │ logs · alerts ·    │  │
│  │ static     │ │ async jobs │ │ tracing            │  │
│  └────────────┘ └────────────┘ └────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Estrategia de Escala

### Cloud Run — Autoescalado
| Componente | Min instances | Max | CPU | Memory | Concurrency |
|:-----------|:-------------|:----|:----|:-------|:------------|
| API Gateway | 1 (cold start) | 100 | 2 vCPU | 1 GB | 80 requests |
| ML Inference | 0 (event-driven) | 50 | 4 vCPU | 8 GB | 10 requests |
| Async Workers | 0 | 20 | 1 vCPU | 512 MB | 1 request |

### BigQuery
- **Dataset**: dev1pruebas (ANALYTICS_*)
- **Slot reservation**: autoscaling (pay-per-query until needed)
- **Caching**: results cached 24h by default
- **Materialized views**: pre-aggregated monthly
- **BQML**: models stored in dataset, inference via SQL

### Frontend (Angular PWA)
- **Static**: Cloud CDN + Cloud Storage bucket
- **Offline**: Service worker + IndexedDB cache
- **Auth**: Google OAuth 2.0 → Workspace token
- **Deploy**: `ng build --configuration production` → Cloud Storage → CDN

## Integración Front → Agente_GCP

### Backend (Go) conecta:
1. **Supabase → BigQuery**: Migrar datos de Supabase a BigQuery (o usar ambos)
2. **Auth**: Google OAuth 2.0 reemplaza auth de Supabase
3. **Google Workspace APIs**: Sheets, Gmail, Chat, Meet
4. **ML**: BQML para predicciones + Gemma para NL→SQL

### Fases de Migración

| Phase | Cambio | Impacto |
|:------|:-------|:--------|
| 1 | Go backend + Cloud Run deploy | Sin cambios en frontend |
| 2 | BigQuery connector (lectura) | Datos DevDBT disponibles en Front |
| 3 | OAuth Workspace | Reemplazar auth Supabase |
| 4 | Sheets export + Gmail | Nuevas features |
| 5 | ML (BQML + Gemma) | Predicciones en dashboard |
| 6 | Chrome Extension | Overlay en cualquier web |

## Deploy

```bash
# Frontend (Angular → Cloud Run / Cloud Storage)
cd frontend
npm install
ng build --configuration production
gcloud storage cp -r dist/ gs://agente-gcp-frontend/

# Backend (Go → Cloud Run)
cd backend/server
gcloud builds submit --tag gcr.io/dev1pruebas/agente-gcp
gcloud run deploy agente-gcp \
  --image gcr.io/dev1pruebas/agente-gcp \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --concurrency 80 \
  --cpu 2 \
  --memory 1Gi \
  --min-instances 1 \
  --max-instances 100

# ML (Gemma → Cloud Run)
gcloud run deploy gemma-server \
  --source ./models/gemma \
  --platform managed \
  --region us-central1 \
  --cpu 4 \
  --memory 8Gi \
  --concurrency 10 \
  --min-instances 0 \
  --max-instances 50 \
  --set-env-vars "MODEL_ID=gemma-2b-it"
```

## Monitoreo

| Herramienta | Qué monitorea |
|:------------|:--------------|
| Cloud Monitoring | API latency, error rate, request count |
| Cloud Logging | BigQuery query logs, errors |
| Cloud Trace | Distributed tracing (BQ → API → Frontend) |
| Custom metrics | ML inference latency, cache hit ratio |
| Alerts | P99 latency > 1s, error rate > 1%, BQ slot usage > 80% |
