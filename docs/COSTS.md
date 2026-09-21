# Agente Workspace — Costos Estimados (EUR)

> Costos operativos mensuales estimados para el proyecto Agente Workspace en Google Cloud Platform.
> Todos los valores en **EUR**, calculados para un equipo de 50–500 usuarios.

---

## Resumen

| Componente | Bajo (50 users) | Medio (200 users) | Alto (500 users) |
|:-----------|:----------------|:------------------|:------------------|
| **Cloud Run API** | €45/mes | €150/mes | €450/mes |
| **Cloud Run ML** | €0 | €120/mes | €400/mes |
| **BigQuery** | €80/mes | €300/mes | €800/mes |
| **Cloud CDN + Storage** | €5/mes | €15/mes | €40/mes |
| **Cloud Tasks + Pub/Sub** | €2/mes | €10/mes | €30/mes |
| **Cloud Monitoring** | €8/mes | €25/mes | €60/mes |
| **Secret Manager** | €3/mes | €6/mes | €12/mes |
| **Total estimado** | **€143/mes** | **€626/mes** | **€1,792/mes** |

---

## Desglose por Componente

### 1. Cloud Run — API Gateway

| Recurso | Valor | Costo/hora | Costo/mes |
|:--------|:------|:-----------|:----------|
| CPU | 2 vCPU | €0.0324 | — |
| Memoria | 1 GB | €0.0040 | — |
| Solicitudes | 10M/mes | €0.40/1M | €4.00 |
| Min instances | 1 | — | €27.00 |
| Max instances | 100 | — | — |
| **Subtotal** | | | **€31–450/mes** |

**Cálculo:** 1 instancia mínima 24/7 = €27. Instancias adicionales bajo demanda + solicitudes.

### 2. Cloud Run — ML Inference (Gemma)

| Recurso | Valor | Costo/hora | Costo/mes |
|:--------|:------|:-----------|:----------|
| CPU | 4 vCPU | €0.0648 | — |
| Memoria | 8 GB | €0.0320 | — |
| Min instances | 0 (cold start) | — | €0 |
| Max instances | 50 | — | — |
| 1000 predicciones/día | 15 min ejecución | — | **€29–120/mes** |

### 3. BigQuery

| Concepto | Bajo | Medio | Alto |
|:---------|:-----|:------|:-----|
| **Storage** (datos silver/gold) | 10 GB × €0.02 = €0.20 | 50 GB × €0.02 = €1 | 200 GB × €0.02 = €4 |
| **Queries** (on-demand) | 500 GB/mes × €5 = €2.50 | 2 TB/mes × €5 = €10 | 10 TB/mes × €5 = €50 |
| **Slots** (flat-rate flexible) | 100 slots × €4 = €400 | 200 slots × €4 = €800 | 500 slots × €4 = €2,000 |
| **ML.BQML** (entrenamiento) | €5/mes | €20/mes | €50/mes |
| **Recomendado** | **on-demand: €8/mes** | **mixto: €30–100/mes** | **flat: €100–800/mes** |

> 💡 **Optimización**: Usar on-demand para desarrollo, flat-rate para producción. 
> Las tablas silver son views (sin storage). Solo gold y bronze consumen.

### 4. Cloud CDN + Storage

| Recurso | Costo/mes |
|:--------|:----------|
| Cloud Storage (Angular static) | €2 (1 GB) |
| Cloud CDN egress | €3 (100 GB) |
| **Subtotal** | **€5/mes** |

### 5. Networking

| Concepto | Costo/mes |
|:---------|:----------|
| VPC Serverless Connector | €10 |
| Cloud NAT (si aplica) | €15 |
| **Subtotal** | **€10–25/mes** |

---

## Optimización de Costos

### BigQuery
```sql
-- Usar tablas particionadas y clustered para reducir scans
CREATE TABLE ANALYTICS_GOLD.rendimiento
PARTITION BY FECHA_MES
CLUSTER BY FILTRO_PLIEGO, AGENTE
AS SELECT * FROM source;

-- Usar materialized views para consultas frecuentes
CREATE MATERIALIZED VIEW ANALYTICS_GOLD.kpi_resumen AS
SELECT FECHA_MES, FILTRO_PLIEGO,
  COUNT(*) AS total,
  AVG(fcr_7d) AS fcr_prom
FROM ANALYTICS_GOLD.RENDIMIENTO_AGENTE_MENSUAL
GROUP BY FECHA_MES, FILTRO_PLIEGO;
```

### Cloud Run
```yaml
# Min instances = 1 evita cold starts, cuesta ~€27/mes
# Para ahorrar, usar min-instances = 0 (aceptar cold start)
resources:
  cpu: 2
  memory: 1Gi
  min_instances: 0  # Ahorra ~€27/mes
  max_instances: 50
```

### Proyección Anual

| Escenario | Mensual | Anual |
|:----------|:--------|:------|
| **Desarrollo** (1 dev) | €50–80 | €600–960 |
| **Producción baja** (50 users) | €143 | €1,716 |
| **Producción media** (200 users) | €626 | €7,512 |
| **Producción alta** (500 users) | €1,792 | €21,504 |

> 🔧 **Nota**: Costos redondeados. Los precios reales pueden variar según región GCP (us-central1 es la más económica). Descuentos por commitment de 1 año: ~20–30% off.
