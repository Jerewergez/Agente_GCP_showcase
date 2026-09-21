# Agente_GCP — Scaling Architecture

## Cloud Run Auto-scaling

### API Gateway
```yaml
# service.yaml
resources:
  cpu: 2
  memory: 1Gi
  concurrency: 80
  min_instances: 1
  max_instances: 100

autoscaling:
  target_cpu_utilization: 0.6
  max_scaledown_replicas: 1
```

### ML Inference (Gemma)
```yaml
resources:
  cpu: 4
  memory: 8Gi
  concurrency: 10
  min_instances: 0
  max_instances: 50
  startup_cpu_boost: true
```

## BigQuery Performance

### Query Optimization
```sql
-- Use materialized views for frequent aggregations
CREATE MATERIALIZED VIEW dev1pruebas.ANALYTICS_GOLD.rendimiento_mv AS
SELECT
  FECHA_MES,
  FILTRO_PLIEGO,
  COUNT(DISTINCT LEGAJO) AS agentes,
  SUM(FCR7D_ATENDIDOS) AS fcr7d_atendidos,
  AVG(NPS_SCORE) AS nps_promedio
FROM dev1pruebas.ANALYTICS_GOLD.RENDIMIENTO_AGENTE_MENSUAL
GROUP BY FECHA_MES, FILTRO_PLIEGO;

-- Use clustering on frequently filtered columns
ALTER TABLE my_table
SET OPTIONS ( clustering = ["FILTRO_PLIEGO", "FECHA_MES"] );
```

### Caching Strategy
| Cache Type | TTL | Size | Use Case |
|:-----------|:----|:-----|:---------|
| BQ Query Results | 24h | 10 GB | Dashboard queries |
| Cloud CDN | 1h | 100 MB | Angular assets |
| Go in-memory | 5m | 100 MB | API responses |
| Service Worker | 24h | 50 MB | Offline data |

## Frontend Scaling (Front)

### PWA Strategy
```typescript
// Service worker caches Angular assets
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open('agente-gcp-v1').then((cache) => {
      return cache.addAll([
        '/', '/index.html', '/main.js',
        '/styles.css', '/assets/'
      ]);
    })
  );
});

// Network-first for API, cache-first for assets
self.addEventListener('fetch', (event) => {
  if (event.request.url.includes('/api/')) {
    // Network-first for data
    event.respondWith(networkFirst(event.request));
  } else {
    // Cache-first for assets
    event.respondWith(cacheFirst(event.request));
  }
});
```

### CDN
```bash
# Deploy Angular app to Cloud Storage + CDN
ng build --configuration production
gcloud storage cp -r dist/ gs://agente-gcp-cdn/
gcloud compute url-maps import agente-gcp-lb \
  --source load-balancer.yaml
```

## Database Scaling

### BigQuery Slots
| Usage | Slots | Cost/mo |
|:------|:------|:--------|
| Dev | 100 (flat) | ~$200 |
| Prod | 500 (flex) | ~$1,000 |
| Peak | 2000 (auto) | ~$4,000 |

### Partitioning
```sql
-- All tables partitioned by month
FECHA_MES DATE → PARTITION BY MONTH
-- Clustering on high-cardinality filters
CLUSTER BY FILTRO_PLIEGO, AGENTE
```

## Load Testing

```bash
# Install hey
go install github.com/rakyll/hey@latest

# Test API Gateway
hey -z 30s -c 100 \
  -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
  https://agente-gcp-xxxxx-uc.a.run.app/api/query

# Expected results:
# P50 < 200ms
# P95 < 500ms
# P99 < 1000ms
# Error rate < 0.1%
```
