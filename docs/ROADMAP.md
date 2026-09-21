# Agente Workspace — Roadmap

> Futuras implementaciones: IA, automatización, scale-out multi-dominio.

---

## Fase 1: MVP (Julio 2026) ✅

| Feature | Estado |
|:--------|:-------|
| Go backend + Cloud Run | ✅ |
| BigQuery client | ✅ |
| OAuth + middleware | ✅ |
| WebSocket hub | ✅ |
| Rate limiter + CORS | ✅ |
| Arquitectura + costos | ✅ |

## Fase 2: Core (Agosto 2026)

| Feature | Prioridad | Esfuerzo | Impacto |
|:--------|:----------|:---------|:--------|
| NL→SQL con template matching | Alta | 3 días | 💥 Query instantáneo |
| WebSocket query endpoint | Alta | 2 días | 💥 Interfaz real-time |
| Google Sheets export (1-click) | Alta | 3 días | 💥 Productividad |
| Gmail summaries | Media | 4 días | 💫 Automatización |
| Front frontend integration | Alta | 5 días | 💥 UX completa |

**Costo estimado:** €50–80 (desarrollo)
**Costo operativo:** +€30/mes

## Fase 3: ML (Septiembre 2026)

| Feature | Prioridad | Esfuerzo | Impacto |
|:--------|:----------|:---------|:--------|
| BQML modelos predictivos (FCR, NPS, TMO) | Alta | 5 días | 💥 Predicciones |
| Gemma Cloud Run serving | Alta | 3 días | 💥 NL→SQL avanzado |
| NL→SQL con Gemma (slow path) | Alta | 4 días | 💥 Queries complejas |
| Confidence intervals en predicciones | Media | 2 días | 📊 Decisiones informadas |

**Costo estimado:** €150–300 (ML infra)
**Costo operativo:** +€150/mes (Gemma Cloud Run)

## Fase 4: Chrome Extension (Octubre 2026)

| Feature | Prioridad | Esfuerzo | Impacto |
|:--------|:----------|:---------|:--------|
| Extension skeleton (MV3) | Alta | 2 días | 💥 Nuevo canal |
| Chat UI (side panel) | Alta | 3 días | 💥 Interacción rápida |
| OAuth flow en extension | Alta | 2 días | 💥 Auth sin fricción |
| Quick actions (Sheets, Gmail) | Media | 3 días | 💫 1-click productividad |
| Offline cache (IndexedDB) | Media | 2 días | 💫 Sin conexión |

**Costo estimado:** €80–120
**Costo operativo:** +€10/mes (CDN)

## Fase 5: Memoria & Aprendizaje (Noviembre 2026)

| Feature | Prioridad | Esfuerzo | Impacto |
|:--------|:----------|:---------|:--------|
| Engram-like memory system | Alta | 5 días | 💥 Contexto persistente |
| LRU cache + Redis | Alta | 2 días | 💥 Performance |
| Pattern learning (BQML) | Media | 4 días | 📊 Recomendaciones |
| Feedback loop (útil/no útil) | Media | 2 días | 📊 Mejora continua |
| Relevance model training | Media | 3 días | 📊 Personalización |

**Costo estimado:** €100–150
**Costo operativo:** +€18/mes (Redis + BQ memory)

## Fase 6: Multi-Dominio (Diciembre 2026)

| Feature | Prioridad | Esfuerzo | Impacto |
|:--------|:----------|:---------|:--------|
| Domain isolation | Alta | 5 días | 🔒 Seguridad |
| Multi-dataset routing | Alta | 3 días | 🔒 Escalabilidad |
| Institutional governance | Alta | 4 días | 🔒 Compliance |
| Admin dashboard | Media | 5 días | 📊 Control |
| Billing per domain | Media | 3 días | 💰 Costos |

**Costo estimado:** €200–300
**Costo operativo:** +€50/mes (auditoría + governance)

## Fase 7: Agentes Autónomos (Enero 2027)

| Feature | Prioridad | Esfuerzo | Impacto |
|:--------|:----------|:---------|:--------|
| Pi agent (orchestrator) | Alta | 5 días | 💥 Automatización |
| OpenCode (Go executor) | Alta | 3 días | 💥 Backend autónomo |
| Auto-scheduling de tareas | Media | 4 días | 💫 Sin intervención |
| Anomaly detection autónomo | Media | 5 días | 🔒 Seguridad proactiva |
| Self-healing pipelines | Baja | 8 días | 🔧 Resiliencia |

**Costo estimado:** €250–400
**Costo operativo:** +€30/mes (Cloud Tasks + Pub/Sub)

---

## Costo Total Proyectado

| Período | Desarrollo | Operación/mes | Acumulado/año |
|:--------|:-----------|:--------------|:--------------|
| Julio 2026 | €0 (actual) | €50 | €0 |
| Agosto 2026 | €150 | €80 | €150 |
| Septiembre 2026 | €250 | €230 | €400 |
| Octubre 2026 | €200 | €240 | €600 |
| Noviembre 2026 | €250 | €258 | €850 |
| Diciembre 2026 | €500 | €308 | €1,350 |
| **2027 (proyectado)** | — | **€400/mes** | **€4,800** |

> 💡 **ROI estimado:** Con 200 usuarios y ahorro de 30 min/día por usuario en tareas Workspace, el retorno es de ~€15,000/mes en productividad recuperada vs €300/mes de operación.
