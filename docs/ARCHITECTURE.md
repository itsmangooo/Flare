# Architecture

```text
Android / .NET MAUI
  secure tokens + HTTPS + SignalR
              │
              ▼
Flare.Api / ASP.NET Core ─── PostgreSQL (Identity, hashed refresh tokens, audit, metrics)
       │             │
       │             └──── Coolify documented /api/v1 endpoints (server-side bearer token)
       └────────────────── Docker Engine narrow client + read-only host telemetry mounts
```

The mobile app never connects to Docker, Coolify, PostgreSQL, or host telemetry directly. `Flare.Contracts` is the only shared layer. Infrastructure adapters implement `IDockerService`, `IHostMetricsService`, `ICoolifyService`, and `IAuditService`; controllers expose explicit allowlisted operations.

SignalR publishes at three-second intervals. Android stops the connection when its window stops, reconnects with bounded exponential delays, and requests a fresh snapshot after resuming. Container metrics refresh every five seconds and update rows in place to avoid list reordering.
