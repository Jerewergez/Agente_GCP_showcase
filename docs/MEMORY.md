# Agente Workspace — Sistema de Memoria (Engram-like)

> Sistema de memoria persistente y eficiente inspirado en Engram, optimizado para BigQuery + Workspace.

---

## Arquitectura

```
┌─────────────────────────────────────────────────────────────┐
│                    Agente Workspace                           │
│                                                               │
│  ┌─────────────────────────┐   ┌──────────────────────────┐  │
│  │ Short-term Memory (RAM)  │   │ Long-term Memory (BQ)   │  │
│  │ · Session context (5min) │   │ · Query history         │  │
│  │ · User state             │   │ · Workspace interactions │  │
│  │ · Active conversations   │   │ · Decisions & patterns   │  │
│  │ TTL: TTL, LRU eviction   │   │ Retention: 90 days      │  │
│  └─────────────────────────┘   └──────────────────────────┘  │
│                │                           │                  │
│                ▼                           ▼                  │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Ephemeral Cache (Cloud Memorystore/Redis)             │  │
│  │  · Fast access (<1ms)                                 │  │
│  │  · Automatic TTL per entry                            │  │
│  │  · Max 1 GB (€15/mes)                                 │  │
│  └────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Componentes

### 1. Short-Term Memory (en memoria Go)
- **Alcance**: Sesión activa del usuario
- **Capacidad**: ~10MB por instancia
- **Estrategia**: LRU + TTL (5 minutos sin actividad)
- **Contenido**: Últimas 20 interacciones, estado de Workspace, cache de queries

### 2. Long-Term Memory (BigQuery)
- **Alcance**: Multi-sesión, compartido entre usuarios
- **Tablas**:
  | Tabla | Propósito | Retention | Costo/mes |
  |:------|:-----------|:----------|:----------|
  | `agente_memory_interactions` | Historial de consultas NL→SQL | 90 días | ~€2 |
  | `agente_memory_decisions` | Decisiones y patrones aprendidos | 365 días | ~€1 |
  | `agente_memory_workspace` | Interacciones con Workspace | 30 días | ~€1 |
  | `agente_memory_embeddings` | Vectores semánticos (similitud) | 90 días | ~€5 |

### 3. Ephemeral Cache (Redis/Memorystore)
- **Alcance**: Cache de respuestas frecuentes
- **Capacidad**: 1 GB, TTL configurable
- **Costos**: ~€15/mes (Redis 1 GB)
- **Uso**: Resultados de BQ, tokens OAuth, contexto de sesión

## Eficiencia

### Estrategia de Cache
```
Nivel 1: Go in-memory (LRU, 10MB, TTL 5min) → costo: €0
Nivel 2: Redis (1 GB, TTL 1h) → costo: €15/mes
Nivel 3: BigQuery (90 days, particionado) → costo: €3/mes
```

### Compresión de Datos
```sql
-- Almacenar interacciones comprimidas
CREATE TABLE agente_memory_interactions (
  user_id STRING,
  timestamp TIMESTAMP,
  query STRING,           -- NL original
  sql STRING,             -- SQL generado
  result_json STRING,     -- Resultado comprimido (JSON)
  latency_ms INT64,
  feedback INT64          -- 1=útil, -1=no útil, 0=sin feedback
)
PARTITION BY DATE(timestamp)
CLUSTER BY user_id
OPTIONS (partition_expiration_days = 90);
```

### Relevancia (Patrones Aprendidos)
```sql
-- BQML para predecir qué información es relevante para cada usuario
CREATE OR REPLACE MODEL `dev1pruebas.AGENTE_MEMORY.relevance_model`
OPTIONS(model_type='matrix_factorization',
  user_col='user_id', item_col='query_pattern',
  rating_col='frequency') AS
SELECT
  user_id,
  REGEXP_EXTRACT(query, r'^(\w+)') AS query_pattern,
  COUNT(*) AS frequency
FROM agente_memory_interactions
WHERE feedback > 0
GROUP BY user_id, query_pattern;
```

## Go Implementation

```go
// internal/memory/memory.go
package memory

type MemoryStore interface {
  Save(ctx context.Context, userID string, entry *Entry) error
  Search(ctx context.Context, userID string, query string, limit int) ([]*Entry, error)
  GetRelevant(ctx context.Context, userID string, pattern string) ([]*Entry, error)
}

type Entry struct {
  UserID    string    `bigquery:"user_id"`
  Timestamp time.Time `bigquery:"timestamp"`
  Type      string    `bigquery:"type"` // query, decision, workspace
  Content   string    `bigquery:"content"`
  Embedding []float64 `bigquery:"embedding,omitempty"`
  Metadata  string    `bigquery:"metadata"`
}

// Level 1: In-memory LRU
type ShortTermMemory struct {
  mu    sync.RWMutex
  items *lru.Cache[string, *Entry]
}

// Level 2: BigQuery long-term
type LongTermMemory struct {
  bq *bigquery.Client
  table string
}
```

## Costos de Memoria

| Componente | Costo/mes | Capacidad | Latencia |
|:-----------|:----------|:----------|:---------|
| Go in-memory (L1) | €0 | 10 MB | < 1µs |
| Redis/Memorystore (L2) | €15 | 1 GB | < 1ms |
| BigQuery (L3) | €3 | 10 GB | ~100ms |
| **Total** | **€18/mes** | | |

> 💡 La memoria es el componente más eficiente del sistema. BigQuery como almacenamiento L3 cuesta menos de €5/mes para 90 días de historial completo.
