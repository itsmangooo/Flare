# API surface

All application endpoints use `/api/v1`; destructive endpoints require the `Administrator` role.

| Method | Path | Purpose |
|---|---|---|
| GET | `/health/live` | Process liveness |
| GET | `/health/ready` | Process and PostgreSQL readiness |
| GET/POST | `/api/v1/auth/bootstrap/status`, `/bootstrap` | One-time first administrator |
| POST | `/api/v1/auth/login`, `/refresh`, `/logout` | Session lifecycle and refresh rotation |
| GET | `/api/v1/auth/me` | Authenticated account |
| GET | `/api/v1/overview` | Host/container/history/activity snapshot |
| GET | `/api/v1/containers` | Container list and metrics |
| GET | `/api/v1/containers/{id}` | Allowlisted container details |
| GET | `/api/v1/containers/{id}/logs?tail=300&before=...` | Bounded log tail (1–2000 lines); optional timestamp pages backward |
| POST | `/api/v1/containers/{id}/start|stop|restart` | Explicit Docker operations |
| GET | `/api/v1/coolify/servers` | Coolify servers |
| GET | `/api/v1/coolify/servers/{uuid}/resources` | Server resources |
| GET | `/api/v1/coolify/applications`, `/services` | Coolify resources |
| GET | `/api/v1/coolify/deployments` | Paginated deployment history |
| GET | `/api/v1/coolify/deployments/{uuid}` | Deployment and available logs |
| POST | `/api/v1/coolify/applications/{uuid}/start|stop|restart|redeploy` | Documented application lifecycle |
| POST | `/api/v1/coolify/services/{uuid}/restart` | Documented service restart |
| GET | `/api/v1/activity` | Unified audit/infrastructure events |
| GET | `/api/v1/system/info` | API/server version |
| SignalR | `/hubs/telemetry` | Authenticated 3-second snapshots |

Errors use RFC Problem Details and include the response `X-Correlation-ID`. The API never includes credentials or internal production exception details.
