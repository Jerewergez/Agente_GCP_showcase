# Agente Workspace — Gobernanza de Datos Institucional

> Gestión institucional de datos gestionados por cuentas del dominio GCP.
> Alineación con Google Cloud Identity, Workspace y BigQuery.

---

## Modelo de Gobernanza

```
Dominio GCP: dev1pruebas.iam.gserviceaccount.com
       │
       ├── Cuentas de Servicio (automation)
       │   ├── agente-workspace-sa → Cloud Run + BQ
       │   └── front-sa → Frontend + Supabase
       │
       ├── Usuarios Workspace (humanos)
       │   ├── admin@example.com → Owner
       │   ├── gerente@example.com → Admin
       │   └── agente@example.com → Viewer
       │
       └── Datos Auditados
           ├── BigQuery: DevDBT (bronze/silver/gold)
           ├── Workspace: Gmail, Sheets, Chat, Meet
           └── Memoria: Interacciones, decisiones, patrones
```

## Principios

### 1. Data Lineage
Toda consulta y acción debe ser trazable al usuario original:
```sql
-- Cada tabla de memoria incluye user_id y timestamp
SELECT user_id, query, sql, timestamp
FROM agente_memory_interactions
WHERE DATE(timestamp) >= DATE_SUB(CURRENT_DATE(), INTERVAL 30 DAY)
  AND user_id IN (SELECT email FROM gobernanza.usuarios_autorizados);
```

### 2. Least Privilege
```yaml
# IAM Roles por perfil
Usuario Workspace Viewer:
  - roles/bigquery.dataViewer (datasets ANALYTICS_*)
  - roles/workspace.invoker
  
Usuario Workspace Editor:
  - roles/bigquery.dataEditor
  - roles/workspace.invoker
  - roles/iam.serviceAccountTokenCreator

Admin:
  - roles/bigquery.admin
  - roles/iam.admin
  - roles/workspace.admin
```

### 3. Audit Trail
```sql
-- BigQuery INFORMATION_SCHEMA para auditoría
SELECT 
  user_email,
  query,
  TIMESTAMP(creation_time) AS query_time,
  total_bytes_billed / 1024 / 1024 / 1024 AS cost_gb,
  error_result
FROM `region-us.INFORMATION_SCHEMA.JOBS`
WHERE DATE(creation_time) >= CURRENT_DATE() - 7
  AND user_email LIKE '%@example.com';
```

## Estructura de Datos Auditados

### Tablas de Memoria Institucional
```sql
-- Historial completo de interacciones
CREATE TABLE dev1pruebas.GOBERNANZA.memory_interactions (
  interaction_id STRING NOT NULL,
  user_email STRING NOT NULL,
  user_role STRING,
  query_nl STRING,
  query_sql STRING,
  result_summary STRING,
  workspace_action STRING,  -- sheets_export, gmail_summary, etc.
  latency_ms INT64,
  feedback INT64,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP(),
  data_origin STRING DEFAULT 'agente-workspace'
)
PARTITION BY DATE(created_at)
CLUSTER BY user_email;

-- Políticas de acceso por dominio
CREATE TABLE dev1pruebas.GOBERNANZA.access_policies (
  domain STRING NOT NULL,   -- example.com
  dataset STRING NOT NULL,  -- ANALYTICS_GOLD
  permission STRING,        -- READER, WRITER, ADMIN
  granted_by STRING,
  created_at TIMESTAMP
);
```

### Matching Institucional
El agente matchea automáticamente usuarios con sus datos:
```sql
-- Match usuario → dominio → datasets permitidos
SELECT 
  u.email,
  u.role,
  d.dataset_name,
  d.permission_level
FROM gobernanza.usuarios u
JOIN gobernanza.dataset_permissions d 
  ON SPLIT(u.email, '@')[OFFSET(1)] = d.domain
WHERE u.email = @current_user;
```

## Cumplimiento

### Data Retention
| Tipo de dato | Retención | Acción al vencer |
|:-------------|:----------|:-----------------|
| Queries temporales | 7 días | DELETE |
| Interacciones usuario | 90 días | DELETE |
| Decisiones aprendidas | 365 días | Archive to coldline |
| Workspace actions | 30 días | DELETE |
| Logs de auditoría | 365 días | Export to Cloud Storage |

### Acceso Multi-Dominio
```yaml
# Configuración de dominios permitidos
allowed_domains:
  - example.com
  - example.com.ar
  - vam.com.ar

# Cada dominio tiene acceso solo a sus propios datos
domain_datasets:
  example.com: [ANALYTICS_SILVER, ANALYTICS_GOLD]
  example.com.ar: [ANALYTICS_AR_SILVER]
  vam.com.ar: [DEV_VAM_SILVER, DEV_VAM_GOLD]
```

## Monitoreo de Gobernanza

### Alertas
| Evento | Alerta | Acción |
|:-------|:-------|:-------|
| Acceso desde dominio no autorizado | Email + Slack | Bloquear IP |
| Query > 1 TB escaneado | Email | Revisar optimización |
| Export masivo de datos | PagerDuty | Investigar |
| Cambio en IAM policies | Audit log | Verificar |

### Dashboard de Gobernanza
```sql
CREATE VIEW dev1pruebas.GOBERNANZA.governance_dashboard AS
SELECT
  DATE(created_at) AS day,
  user_email,
  COUNT(*) AS interactions,
  SUM(total_bytes_billed) / 1e12 AS tb_processed,
  COUNT(DISTINCT workspace_action) AS actions_used
FROM gobernanza.memory_interactions
GROUP BY day, user_email;
```

## Costos de Gobernanza

| Componente | Costo/mes | Descripción |
|:-----------|:----------|:------------|
| BigQuery (gobernanza data) | €5 | Tablas de auditoría |
| Cloud Logging | €8 | Logs de auditoría |
| Secret Manager | €3 | Keys de SA |
| Cloud KMS | €5 | Encriptación de datos sensibles |
| **Total** | **€21/mes** | |
