# Architecture

```text
Android / Flutter + Riverpod
  secure tokens + HTTPS + SSE
              │
              ▼
Flare.Api / ASP.NET Core ─── PostgreSQL (Identity, hashed refresh tokens, audit, metrics)
       │             │
       │             └──── Coolify documented /api/v1 endpoints (server-side bearer token)
       └────────────────── Docker Engine narrow client + read-only host telemetry mounts
```

The mobile app never connects to Docker, Coolify, PostgreSQL, or host telemetry directly. `Flare.Contracts` is the only shared layer. Infrastructure adapters implement `IDockerService`, `IHostMetricsService`, `ICoolifyService`, and `IAuditService`; controllers expose explicit allowlisted operations.

The Go API publishes authenticated server-sent events at three-second intervals. The Flutter client stops the connection when Android backgrounds the app, reconnects with bounded exponential delays, and requests a fresh snapshot after resuming. Container metrics refresh without visually reordering rows.
