# Agente_GCP — Security Architecture

## Authentication & Authorization

### OAuth 2.0 Flow
```
User → Google Login → Access Token + Refresh Token
    ↓
Chrome Extension stores token (chrome.storage)
    ↓
Go Backend validates token via Google API
    ↓
UserInfo + Workspace scopes granted
```

### Scopes (principle of least privilege)
| Scope | Resource | Justification |
|:------|:---------|:--------------|
| `bigquery.readonly` | DevDBT datasets | Read-only queries |
| `spreadsheets` | Google Sheets | Create/append only |
| `gmail.readonly` | Gmail inbox | Summarize threads |
| `chat.readonly` | Google Chat | Read messages |
| `meet.readonly` | Google Meet | Transcripts |

### Service Account (Backend)
```yaml
SA: agente-gcp-sa@dev1pruebas.iam.gserviceaccount.com
Roles:
  - roles/bigquery.dataViewer
  - roles/bigquery.jobUser
  - roles/run.invoker
  - roles/iam.serviceAccountTokenCreator
```

### API Security
- All endpoints behind Cloud Run IAM
- Rate limiting: 100 req/min per user
- CORS: restrict to Chrome Extension origin
- Request validation: JWT + API key
- Audit logging: Cloud Audit Logs for all BQ queries

## Data Security

### BigQuery
- Column-level security for PII
- Authorized views over tables (never direct table access)
- Query audit trail via INFORMATION_SCHEMA
- Data retention: 90 days for raw, 365 days for aggregated

### Encryption
- Data at rest: AES-256 (GCP default)
- Data in transit: TLS 1.3
- Secrets: Secret Manager (never env vars)
- SA keys: never committed to repo (use workload identity)

## Network Security

### Cloud Run
```
Cloud Run (agente-gcp)
  └── Ingress: Internal + Cloud CDN
  └── VPC: Serverless VPC Connector
  └── Egress: Private Google Access
  └── Auth: IAM + OAuth2
```

### VPC
- Serverless VPC Connector for Cloud Run → BQ
- VPC Service Controls to prevent data exfiltration
- Private Google Access for all GCP services

## Operational Security

### Monitoring & Alerting
| Alert | Threshold | Action |
|:------|:----------|:-------|
| Auth failures | > 5/min | PagerDuty + Discord |
| BQ query cost | > $10/query | Slack alert |
| API error rate | > 1% | Auto-scaling review |
| Rate limit hits | > 50% | Rate limit review |

### CI/CD Security
- GitHub Actions with OIDC (no static keys)
- SA key via Workload Identity Federation
- Secret scanning in PRs
- Dependency vulnerability scanning (Dependabot)

## Compliance Checklist

- [ ] OAuth consent screen configured
- [ ] Data Processing Agreement (DPA) in place
- [ ] Logs retention configured (30 days)
- [ ] Incident response plan documented
- [ ] Access reviews quarterly
- [ ] Penetration testing annually
