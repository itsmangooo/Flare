# Flare

Flare is a self-hosted Android homelab administration app. The native .NET MAUI client talks only to the ASP.NET Core API; Docker and Coolify credentials never leave the homelab.

## What is included

- Compact native Android overview for host CPU, load, memory, disk, network, uptime, container state, metric history, and recent activity.
- Dense container browsing, bounded logs, live following, and allowlisted start/stop/restart actions.
- Coolify servers, resources, applications, services, deployments, deployment logs, and documented lifecycle/redeploy actions.
- SignalR telemetry with reconnect, stale/offline state, and foreground/background lifecycle handling.
- ASP.NET Core Identity, lockout, short-lived JWT access tokens, hashed rotating refresh tokens, reuse-family revocation, administrator policies, rate limiting, Problem Details, and correlation IDs.
- PostgreSQL EF Core migrations, audit/infrastructure events, liveness/readiness checks, multi-stage non-root container image, and CI for backend/tests/Android/container builds.

## Repository layout

```text
Flare.sln
src/Flare.Mobile/       Android-only .NET MAUI client
src/Flare.Api/          ASP.NET Core API
src/Flare.Contracts/    Shared wire contracts
tests/Flare.Api.Tests/  Security, metrics, and integration-contract tests
```

## Local verification

Install .NET SDK 10.0.300, Java 17, the Android SDK, and the `maui-android` workload, then run:

```powershell
dotnet workload install maui-android
dotnet tool restore
dotnet restore Flare.sln
dotnet build src/Flare.Api/Flare.Api.csproj -c Release
dotnet test tests/Flare.Api.Tests/Flare.Api.Tests.csproj -c Release
dotnet build src/Flare.Mobile/Flare.Mobile.csproj -c Release -r android-arm64
```

The installable APK is produced under `src/Flare.Mobile/bin/Release/net10.0-android/android-arm64/`. Local builds use the Android development certificate unless the documented production-signing properties are supplied.

## Production deployment

See [Production deployment](docs/DEPLOYMENT.md) for Coolify configuration, migrations, mounts, health checks, and Android signing. See [Security](docs/SECURITY.md) before granting Docker access.

The first administrator is created once with `POST /api/v1/auth/bootstrap`. There are no default credentials and no public registration. After any user exists, bootstrap permanently returns a conflict regardless of the token supplied.

```bash
curl --fail-with-body https://flare.example.com/api/v1/auth/bootstrap \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"<strong-password>","bootstrapToken":"<FLARE_BOOTSTRAP_TOKEN>"}'
```

Remove `FLARE_BOOTSTRAP_TOKEN` from the deployed environment after successful bootstrap.

## API boundary

Flare exposes explicit `/api/v1` operations only. It has no remote shell, generic host command, container deletion/exec endpoint, arbitrary Docker command endpoint, or raw Docker/Coolify proxy. See [API surface](docs/API.md).
