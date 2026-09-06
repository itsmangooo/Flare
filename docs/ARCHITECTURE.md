# Architecture

```text
Android / Flutter + Riverpod
  secure tokens + HTTPS + SSE
              |
              v
Flare Go API ----- PostgreSQL (users, hashed refresh tokens, audit, metrics)
       |        |
       |        +---- Coolify and Cloudflare documented APIs (server-side tokens)
       +------------- Docker Engine narrow client + read-only host telemetry mounts
```

The mobile app never connects to Docker, Coolify, Cloudflare, PostgreSQL, or host telemetry directly. Small Go packages expose explicit allowlisted operations and keep integration credentials server-side.

The Go API publishes authenticated server-sent events at three-second intervals. The Flutter client stops the connection when Android backgrounds the app, reconnects with bounded delays, and requests a fresh snapshot after resuming. Container metrics refresh without visually reordering rows.

The production image contains one static Go binary in a non-root distroless runtime. A one-shot invocation of the same image applies embedded, forward-only PostgreSQL migrations before the API starts.
