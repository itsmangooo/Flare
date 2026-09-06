# Architecture

```text
Android / Flutter + Riverpod
  secure tokens + HTTPS + SSE
              |
              v
Flare Go API ----- PostgreSQL (users, hashed refresh tokens, audit, metrics, alerts)
       |        |
       |        +---- Coolify and Cloudflare documented APIs (server-side tokens)
       +------------- Docker Engine narrow client + read-only host telemetry mounts
```

The mobile app never connects to Docker, Coolify, Cloudflare, PostgreSQL, or host telemetry directly. Small Go packages expose explicit allowlisted operations and keep integration credentials server-side.

The Go API publishes authenticated server-sent events at three-second intervals. The Flutter client stops the connection when Android backgrounds the app, reconnects with bounded delays, and requests a fresh snapshot after resuming. Container metrics refresh without visually reordering rows.

Alert history is stored in PostgreSQL with severity, source/resource association, active or recovered state, occurrence count, and timestamps. A partial unique fingerprint deduplicates active conditions atomically; repeated findings update the same alert and external delivery is limited to once per 15-minute cooldown. Recovery closes only the matching active condition. Server notification preferences can disable delivery, require a minimum severity, disable recoveries, or disable a source without suppressing persisted history. Read state is stored separately per authenticated user. The history API never exposes notification-provider credentials.

The Flutter Alerts screen reads only the authenticated alert-history API. It exposes status, severity, source, occurrence count, unread filtering, and per-user read toggles; provider credentials and internal fingerprints stay server-side.

Administrators can edit the global delivery threshold, recovery behavior, and source-specific delivery switches from Flutter. The Go API validates and persists the complete preference document; non-administrators may read the effective policy but receive `403 Forbidden` on updates.

The host telemetry sampler also feeds a small threshold evaluator. CPU and memory conditions require sustained samples and use recovery hysteresis; unavailable metrics remain unknown instead of being treated as healthy.

When Coolify is configured, a bounded background poll records integration availability transitions and new terminal deployments. Existing history is seeded without replaying stale failures, while recent failures and later successful deployments produce deduplicated alert/recovery pairs associated with the Coolify resource.

Cloudflare monitoring is also optional. Zone access supplies the integration-availability signal; when an account ID enables tunnel access, only documented tunnel health states produce unavailable/recovery transitions. Unknown states remain unknown, and neither credentials nor private tunnel origins enter activity or alert records.

The production image contains one static Go binary in a non-root distroless runtime. A one-shot invocation of the same image applies embedded, forward-only PostgreSQL migrations before the API starts.
