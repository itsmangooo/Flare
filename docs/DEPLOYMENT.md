# Production deployment

## Coolify service

Create a PostgreSQL resource and an application from this repository. The root `Dockerfile` publishes only `Flare.Api` into a .NET 10 chiseled runtime image, runs as the built-in non-root user, listens on port 8080, and handles SIGTERM through ASP.NET Core's normal graceful shutdown.

Set the health check to `/health/live` and expose the service only through Coolify's HTTPS proxy. `/health/ready` additionally verifies PostgreSQL connectivity.

Configure these production variables:

| Variable | Required | Purpose |
|---|---:|---|
| `ASPNETCORE_ENVIRONMENT=Production` | yes | Production exception and HTTPS behavior |
| `ConnectionStrings__Postgres` | yes | Coolify PostgreSQL connection string |
| `FLARE_BOOTSTRAP_TOKEN` | first boot | High-entropy one-time first-admin token |
| `FLARE_JWT_SIGNING_KEY` | yes | At least 32 random bytes; retain across restarts |
| `COOLIFY_BASE_URL` | for Coolify | Trusted self-hosted HTTPS base URL |
| `COOLIFY_API_TOKEN` | for Coolify | Coolify token with only `read` and `deploy` permissions |
| `DOCKER_HOST` | for containers | Docker endpoint, normally `unix:///var/run/docker.sock` or a restricted proxy URL |
| `FLARE_HOST_NAME` | no | Display name, default `homelab` |
| `HOST_PROC_PATH` | for host metrics | Default `/host/proc` |
| `HOST_ROOTFS_PATH` | for disk metric | Default `/host/rootfs`; absent means disk is unavailable |
| `FLARE_ACCESS_TOKEN_MINUTES` | no | 5–60, default 15 |
| `FLARE_REFRESH_TOKEN_DAYS` | no | 1–90, default 30 |
| `FLARE_JWT_ISSUER` / `FLARE_JWT_AUDIENCE` | no | Token validation names |
| `FLARE_TRUST_ALL_FORWARDERS` | conditional | Set `true` only when port 8080 is isolated inside Coolify's trusted proxy network |

Never place secrets in the repository. `.env.example` contains placeholders only.
The chiseled image disables PostgreSQL GSS session encryption because Kerberos libraries are intentionally absent; configure `SSL Mode` in `ConnectionStrings__Postgres` when database transport encryption is required.

## Database migrations

Flare never calls `EnsureCreated`, drops, or recreates the production database. Migrations are compiled into the app. Back up PostgreSQL, then run the exact image as a one-off/pre-deployment command:

```text
dotnet Flare.Api.dll --migrate
```

The command applies pending EF migrations and exits. Run it once before switching traffic to a version containing schema changes. Normal startup does not mutate the schema.

## Host telemetry mounts

For CPU, load, RAM, network, and uptime, mount only proc read-only:

```text
/proc:/host/proc:ro
```

For root disk capacity, a separate read-only view is required:

```text
/:/host/rootfs:ro
```

The second mount exposes host filenames to the container even though it is read-only. Omit it if that exposure is unacceptable; Flare returns disk as unavailable rather than reporting its own container filesystem. Never mount the host filesystem writable.

## Docker access

Prefer a restricted Docker API proxy and set `DOCKER_HOST` to it. If the raw socket is unavoidable, mount `/var/run/docker.sock` and arrange a supplemental group matching the socket GID for Flare's non-root UID. A read-only socket mount does **not** make Docker API access read-only. See [Security](SECURITY.md).

## Android signing

Release builds support these MSBuild/environment properties without committing a keystore:

- `FLARE_ANDROID_KEYSTORE`
- `FLARE_ANDROID_STORE_PASSWORD`
- `FLARE_ANDROID_KEY_ALIAS`
- `FLARE_ANDROID_KEY_PASSWORD`

For a future signed GitHub release, store `FLARE_ANDROID_KEYSTORE_BASE64` plus the three password/alias values as GitHub Actions secrets, decode the keystore into the runner's temporary directory, pass its path as `FLARE_ANDROID_KEYSTORE`, and delete it after the build. CI uses the Android development certificate today and requires no production secret; that package is installable for verification but must not be distributed as a production release.
