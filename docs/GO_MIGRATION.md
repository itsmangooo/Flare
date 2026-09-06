# Go backend migration

The Go service is being built alongside `Flare.Api`. The production Dockerfile and Coolify deployment continue to run the C# service until parity is tested. No migration step may delete or recreate existing data.

## Compatibility inventory

Public endpoints:

- `GET /health/live`
- `GET /health/ready` (PostgreSQL readiness)

Authentication endpoints and payloads:

- `GET /api/v1/auth/bootstrap/status`
- `POST /api/v1/auth/bootstrap`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`
- HS256 access tokens with the existing issuer, audience, claim names, and string role claims
- rotating, SHA-256-hashed refresh tokens with family revocation after reuse
- ASP.NET Identity V3 password-hash verification and existing lockout fields

Authenticated infrastructure endpoints:

- overview and host metrics: `GET /api/v1/overview`
- container list, detail, bounded logs, start, stop, and restart
- Coolify servers, server resources, applications, services, deployments, and deployment detail
- allowlisted Coolify application start/stop/restart/redeploy and service restart
- paginated unified activity/audit feed
- `GET /api/v1/system/info`
- live overview updates currently published through `/hubs/telemetry`

The Flutter client expects camel-case JSON, ISO-8601 timestamps, string enum values, Problem Details errors, bearer authentication, and status codes including 202, 401, 403, 404, 423, 429, and 503.

## Existing database

The Go implementation reads the current EF Core schema in place, including `Users`, `Roles`, `UserRoles`, `RefreshTokens`, `AuditEvents`, `InfrastructureEvents`, and `MetricSamples`. `flare --migrate` uses the separate `FlareSchemaMigrations` version table and embedded, forward-only SQL. Its idempotent baseline adopts an existing EF-created database without deleting or recreating data and initializes the tables required by a fresh Go-only installation. EF migrations remain authoritative until the deployment switch.

## Cutover gate

The root Dockerfile and `compose.yml` must not switch to Go until contract tests, Flutter integration tests, migration rehearsal, container/Coolify tests, telemetry reconnection tests, and a production-like deployment all pass. Rollback keeps the same PostgreSQL schema and returns traffic to the C# container.
