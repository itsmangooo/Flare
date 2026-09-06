# Flare

Flare is a self-hosted Android homelab administration app. The native Flutter client talks only to the Go API; Docker, Coolify, and integration credentials never leave the homelab.

## What is included

- Compact native Android overview for host CPU, load, memory, disk, network, uptime, container state, metric history, and recent activity.
- Dense container browsing, bounded logs, live following, and allowlisted start/stop/restart actions.
- Coolify servers, resources, applications, services, deployments, deployment logs, and documented lifecycle/redeploy actions.
- Authenticated SSE telemetry with reconnect, stale/offline state, and foreground/background lifecycle handling.
- Go authentication compatible with existing password hashes, lockout, short-lived JWT access tokens, hashed rotating refresh tokens, reuse-family revocation, administrator policies, rate limiting, Problem Details, and correlation IDs.
- Forward-only PostgreSQL migrations, audit/infrastructure events, liveness/readiness checks, a minimal non-root Go container image, and CI for backend/tests/Android/container builds.

## Repository layout

```text
cmd/flare/              Go API executable
internal/               Go API packages and integrations
src/Flare.Mobile/       Flutter/Dart Android client
src/Flare.Api/          Legacy C# parity reference pending removal
```

## Local verification

Install Go 1.26, Flutter 3.47.2, Java 17, and the Android SDK, then run:

```powershell
go test ./...
go vet ./...
go build -trimpath ./cmd/flare
cd src/Flare.Mobile
flutter pub get
flutter analyze --fatal-infos
flutter test
flutter build apk --debug
```

Debug builds use Android debug signing. Distributable Release builds require the persistent release keystore and four signing values documented in [Production deployment](docs/DEPLOYMENT.md#android-release-signing); a Release build fails if any value is missing.

## Production deployment

The root [`compose.yml`](compose.yml) is the production entry point for a Coolify Docker Compose deployment. It builds the Go API, applies pending forward-only migrations through a one-shot service, and starts the API only after migration succeeds. See [Production deployment](docs/DEPLOYMENT.md) for the required Coolify variables, PostgreSQL networking, public HTTPS domain, mounts, and Android signing. See [Security](docs/SECURITY.md) before granting Docker access.

The first administrator is created once with `POST /api/v1/auth/bootstrap`. There are no default credentials and no public registration. After any user exists, bootstrap permanently returns a conflict regardless of the token supplied.

```bash
curl --fail-with-body https://flare.example.com/api/v1/auth/bootstrap \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"<strong-password>","bootstrapToken":"<FLARE_BOOTSTRAP_TOKEN>"}'
```

Remove `FLARE_BOOTSTRAP_TOKEN` from the deployed environment after successful bootstrap.

## API boundary

Flare exposes explicit `/api/v1` operations only. It has no remote shell, generic host command, container deletion/exec endpoint, arbitrary Docker command endpoint, or raw Docker/Coolify proxy. See [API surface](docs/API.md).
