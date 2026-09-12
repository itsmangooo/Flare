# Production deployment

## Coolify Docker Compose service

Create a PostgreSQL resource and a Docker Compose application from this repository. Use `/` as the base directory and `/compose.yml` as the Docker Compose location. If PostgreSQL is a separate Coolify resource, enable **Connect to Predefined Network** and use its full internal service hostname in the connection string.

Assign an HTTPS domain to the `api` service and route it to container port `8080`. Do not add a host `ports` mapping: `compose.yml` exposes the port only to Coolify's proxy network. Keep Coolify's HTTP-to-HTTPS redirect enabled at the public edge; Flare intentionally serves HTTP only on the private proxy network. The address entered in the Android app is the public URL, for example `https://flare.example.com`, with no `/api` suffix. `/health/live` is the container health check and `/health/ready` additionally verifies PostgreSQL connectivity.

The root `Dockerfile` builds a static Go binary into a minimal distroless image, runs as the built-in non-root user, listens on port 8080, and handles SIGTERM with a bounded graceful shutdown.

Configure these production variables:

| Variable | Required | Purpose |
|---|---:|---|
| `ConnectionStrings__Postgres` | yes | Coolify PostgreSQL connection string |
| `FLARE_JWT_SIGNING_KEY` | yes | At least 32 random bytes; retain across restarts |
| `FLARE_VERSION` | no | Image build version shown by the public and system information endpoints |
| `COOLIFY_BASE_URL` | no | Trusted self-hosted HTTPS base URL, without `/api/v1` |
| `COOLIFY_API_TOKEN` | with `COOLIFY_BASE_URL` | Coolify token with only the required `read` and `deploy` permissions |
| `CLOUDFLARE_API_TOKEN` | no | Least-privilege token for optional zone/DNS reads |
| `CLOUDFLARE_ACCOUNT_ID` | no | Account identifier required only for Tunnel inventory |
| `DOCKER_SOCKET_GID` | yes | Numeric group ID owning `/var/run/docker.sock` on the deployment server |
| `FLARE_BOOTSTRAP_TOKEN` | first boot | High-entropy one-time first-admin token; clear it after bootstrap |
| `DOCKER_HOST` | no | Defaults to `unix:///var/run/docker.sock` |
| `FLARE_HOST_NAME` | no | Display name, default `homelab` |
| `FLARE_ALERT_CPU_PERCENT` | no | Sustained host CPU alert threshold, 1-100; default 90 |
| `FLARE_ALERT_MEMORY_PERCENT` | no | Sustained host memory alert threshold, 1-100; default 90 |
| `FLARE_ALERT_SUSTAINED_SAMPLES` | no | Consecutive breach/recovery samples required, 2-20; default 5 |
| `FLARE_ACCESS_TOKEN_MINUTES` | no | 5–60, default 15 |
| `FLARE_REFRESH_TOKEN_DAYS` | no | 1–90, default 30 |
| `FLARE_JWT_ISSUER` / `FLARE_JWT_AUDIENCE` | no | Token validation names |
| `NTFY_BASE_URL` | no | HTTPS root URL of an optional ntfy notification server |
| `NTFY_TOPIC` | with `NTFY_BASE_URL` | Private topic receiving infrastructure alerts; 1-64 letters, numbers, `_`, or `-` |
| `NTFY_TOKEN` | no | Optional ntfy Bearer token with write access to only the alert topic |

`compose.yml` fixes the mounted host telemetry paths. It does not publish port 8080 directly; traffic reaches the API through Coolify's private proxy network.

Find the Docker socket group ID on the Coolify server with:

```bash
stat -c '%g' /var/run/docker.sock
```

Enter the result as `DOCKER_SOCKET_GID`. Configure all secrets as runtime-only Coolify variables. If a value contains `$`, enable Coolify's **Literal** option so Compose does not interpolate it. The Android app needs none of these secrets; it only stores the public HTTPS URL and authentication tokens.

The required values in Coolify's Developer view have this shape (replace every placeholder):

```dotenv
ConnectionStrings__Postgres=Host=<postgres-service-name>;Port=5432;Database=<database>;Username=<username>;Password=<password>;SSL Mode=Prefer
FLARE_JWT_SIGNING_KEY=<at-least-32-random-bytes>
FLARE_VERSION=<release-version>
COOLIFY_BASE_URL=https://<your-coolify-domain>
COOLIFY_API_TOKEN=<least-privilege-read-and-deploy-token>
DOCKER_SOCKET_GID=<numeric-gid>
FLARE_BOOTSTRAP_TOKEN=<one-time-random-token>
FLARE_HOST_NAME=<display-name>
FLARE_ALERT_CPU_PERCENT=90
FLARE_ALERT_MEMORY_PERCENT=90
FLARE_ALERT_SUSTAINED_SAMPLES=5
NTFY_BASE_URL=https://<your-ntfy-domain>
NTFY_TOPIC=<private-random-alert-topic>
NTFY_TOKEN=<optional-topic-write-token>
```

Only `FLARE_BOOTSTRAP_TOKEN` should be cleared after the first administrator has been created. Retain the signing key and database credentials across every restart and deployment; changing the signing key immediately invalidates all access tokens.

Never place secrets in the repository. `.env.example` contains placeholders only.
The static image does not include Kerberos libraries; configure `SSL Mode` in `ConnectionStrings__Postgres` when database transport encryption is required.

The Go backend treats ntfy as optional. When configured, it publishes Docker outage, unexpected-stop, out-of-memory, unhealthy, restart-loop, and recovery notifications through ntfy's JSON API. Repeated active conditions share a fingerprint and update persisted history immediately, while external notifications use a 15-minute cooldown; recovery notifications are sent once when a matching active condition closes. Delivery is retried and never includes Docker labels, raw daemon errors, tokens, or internal network addresses. Use a private, hard-to-guess topic and preferably protect it with a dedicated write-scoped access token.

## Database migrations

Flare never calls `EnsureCreated`, drops, or recreates the production database. Migrations are compiled into the app. In the Compose deployment, the one-shot `migrate` service applies pending migrations and must exit successfully before `api` starts. Coolify is instructed to exclude that completed one-shot container from ongoing health checks.

Back up PostgreSQL before deploying a release containing schema changes. For a standalone Dockerfile deployment, run the exact image as a one-off/pre-deployment command:

```text
/app/flare --migrate
```

The command applies embedded forward-only Go migrations and exits. Run it once before switching traffic to a version containing schema changes. Normal startup does not mutate the schema.

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

CPU and memory alerts require the configured number of consecutive three-second samples. Recovery requires the same number of samples at least five percentage points below the alert threshold, which avoids flapping around the boundary. Missing host metrics never create or recover an alert.

## Docker access

The supplied Compose deployment mounts `/var/run/docker.sock` and adds the configured socket GID as a supplemental group for Flare's non-root UID. A read-only socket mount would **not** make Docker API access read-only, because authorization happens through API calls; the mount is intentionally read/write for allowlisted start/stop/restart operations.

Prefer a restricted Docker API proxy when available. To use one, set `DOCKER_HOST` to its private URL and remove both the raw socket bind mount and `group_add` from your Compose override. See [Security](SECURITY.md) before using either approach.

## Android release signing

Android accepts an APK as an update only when its application ID is unchanged, it is signed by the same key as the installed APK, and its `versionCode` is higher. Flare keeps the application ID `io.github.itsmangooo.flare`. Release builds fail if the persistent release key is not configured; they never fall back to debug signing.

Create the release key once and keep it outside the repository. This command prompts for the passwords instead of placing them in shell history:

```powershell
New-Item -ItemType Directory -Force "$env:USERPROFILE\.flare" | Out-Null
keytool -genkeypair -keystore "$env:USERPROFILE\.flare\flare-release.keystore" -alias flare-release -keyalg RSA -keysize 4096 -validity 10000 -storetype JKS
```

Back up the keystore and its passwords in secure, separate locations. Losing the key permanently prevents upgrades of installations signed by it. Never commit or upload the unencrypted keystore as a release asset; `*.keystore` and `*.jks` are ignored by Git.

Set the four required environment variables for the build process:

```powershell
$env:FLARE_ANDROID_KEYSTORE = "$env:USERPROFILE\.flare\flare-release.keystore"
$env:FLARE_ANDROID_KEY_ALIAS = "flare-release"
$env:FLARE_ANDROID_KEYSTORE_PASSWORD = "<from-secure-password-store>"
$env:FLARE_ANDROID_KEY_PASSWORD = "<from-secure-password-store>"

Set-Location flare_mobile
flutter pub get
.\tool\build-release.ps1
```

The signed artifact is written under `flare_mobile/build/app/outputs/flutter-apk/`. Flutter reads `versionName` and `versionCode` from the `version` field in `pubspec.yaml`; for example, `version: 1.2.0+6` produces semantic version `1.2.0` and Android version code `6`. The release helper also copies it to a versioned filename such as `Flare-v1.2.0-build6-android.apk`.

Before every release, increase the semantic portion before `+` and increase the integer build number after `+` to a value greater than every previously published build. Always use the same release keystore and alias. Debug builds and CI use debug signing and must not be distributed.

Install an update with `adb install -r <apk-path>` or open the APK normally on the device. APKs published before the stable release key was introduced cannot be upgraded in place because their signing certificate differs; uninstall one of those builds once, then install the first stable-key release. Later stable-key releases upgrade normally.

## Standalone Docker Compose deployment

Flare can run on a standard Docker host without Coolify.

For standalone deployments, use `compose.docker.yml`. This stack runs:

- PostgreSQL with persistent storage
- a one-shot Flare migration container
- the Flare API
- Docker integration through the host Docker socket
- read-only host telemetry mounts

Coolify, Cloudflare, and ntfy remain optional integrations and are not required for the core Flare functionality.

### Requirements

Install:

- Docker Engine
- Docker Compose v2
- Git

Clone the repository:

```bash
git clone https://github.com/itsmangooo/Flare.git
cd Flare
```

The deployment host must expose the Docker socket at:

```text
/var/run/docker.sock
```

### Environment

Create a `.env` file in the repository root.

At minimum configure:

```dotenv
POSTGRES_DB=flare
POSTGRES_USER=flare
POSTGRES_PASSWORD=<random-database-password>

FLARE_VERSION=<release-version>

FLARE_JWT_SIGNING_KEY=<random-value-at-least-32-bytes>
FLARE_BOOTSTRAP_TOKEN=<one-time-random-bootstrap-token>

FLARE_JWT_ISSUER=Flare.Api
FLARE_JWT_AUDIENCE=Flare.Mobile

FLARE_ACCESS_TOKEN_MINUTES=15
FLARE_REFRESH_TOKEN_DAYS=30

FLARE_HOST_NAME=homelab

FLARE_ALERT_CPU_PERCENT=90
FLARE_ALERT_MEMORY_PERCENT=90
FLARE_ALERT_SUSTAINED_SAMPLES=5

DOCKER_HOST=unix:///var/run/docker.sock
DOCKER_SOCKET_GID=<docker-socket-group-id>

FLARE_BIND_ADDRESS=0.0.0.0
FLARE_HTTP_PORT=8080
```

Generate independent random values for the PostgreSQL password, JWT signing key, and bootstrap token.

For example:

```bash
openssl rand -hex 32
```

Keep `FLARE_JWT_SIGNING_KEY` stable across deployments. Changing it invalidates existing access tokens.

Never commit `.env`.

### Docker socket permissions

Flare runs as a non-root container and needs permission to access the host Docker socket.

Find the numeric group ID that owns the socket:

```bash
stat -c '%g' /var/run/docker.sock
```

Set the returned value as:

```dotenv
DOCKER_SOCKET_GID=<gid>
```

For example:

```dotenv
DOCKER_SOCKET_GID=998
```

### Validate the configuration

Before starting the stack:

```bash
docker compose -f compose.docker.yml config
```

This should complete without missing-variable or Compose validation errors.

### Start Flare

Build and start the stack:

```bash
docker compose -f compose.docker.yml up -d --build
```

Check the service state:

```bash
docker compose -f compose.docker.yml ps
```

The migration service is expected to run once and exit successfully.

Inspect it with:

```bash
docker compose -f compose.docker.yml logs migrate
```

Follow API logs with:

```bash
docker compose -f compose.docker.yml logs -f api
```

### Access the API

By default Flare listens on:

```text
http://<server-ip>:8080
```

The Android client can use that URL directly when Flare is reachable only over a trusted local network.

For remote or internet-facing deployments, place Flare behind HTTPS using a reverse proxy or secure tunnel.

Examples include:

- Caddy
- Nginx
- Traefik
- Cloudflare Tunnel

When a reverse proxy runs on the same host, bind Flare to localhost instead:

```dotenv
FLARE_BIND_ADDRESS=127.0.0.1
```

The proxy can then forward traffic to:

```text
http://127.0.0.1:8080
```

Do not expose an unauthenticated plaintext deployment directly to the public internet.

### Health checks

Check liveness:

```bash
curl http://localhost:8080/health/live
```

Check readiness:

```bash
curl http://localhost:8080/health/ready
```

The readiness endpoint also verifies PostgreSQL connectivity.

### First administrator

Flare has no default credentials and no public registration.

Create the first administrator using the configured `FLARE_BOOTSTRAP_TOKEN`:

```bash
curl --fail-with-body http://localhost:8080/api/v1/auth/bootstrap \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "admin@example.com",
    "password": "<strong-password>",
    "bootstrapToken": "<FLARE_BOOTSTRAP_TOKEN>"
  }'
```

After the first administrator has been created, clear the bootstrap token:

```dotenv
FLARE_BOOTSTRAP_TOKEN=
```

Apply the updated environment:

```bash
docker compose -f compose.docker.yml up -d
```

After any user exists, the bootstrap endpoint permanently rejects additional bootstrap attempts.

### Optional integrations

Standalone Docker does not disable any optional Flare integration.

Coolify:

```dotenv
COOLIFY_BASE_URL=https://coolify.example.com
COOLIFY_API_TOKEN=<least-privilege-token>
```

Cloudflare:

```dotenv
CLOUDFLARE_API_TOKEN=<least-privilege-token>
CLOUDFLARE_ACCOUNT_ID=<account-id>
```

ntfy:

```dotenv
NTFY_BASE_URL=https://ntfy.example.com
NTFY_TOPIC=<private-topic>
NTFY_TOKEN=<optional-write-token>
```

Leaving these values empty does not affect Docker container monitoring, host telemetry, authentication, or the core Flare API.

### Updating

Before deployments containing database schema changes, back up PostgreSQL.

Then pull the latest version:

```bash
git pull
```

Rebuild and restart:

```bash
docker compose -f compose.docker.yml up -d --build
```

The migration service runs before the new API container starts.

Verify:

```bash
docker compose -f compose.docker.yml ps
docker compose -f compose.docker.yml logs migrate
```

### Stopping

Stop the stack while preserving PostgreSQL data:

```bash
docker compose -f compose.docker.yml down
```

Start it again with:

```bash
docker compose -f compose.docker.yml up -d
```

Do not use:

```bash
docker compose -f compose.docker.yml down -v
```

unless the PostgreSQL database should be permanently deleted.

### Production notes

The standalone deployment should follow the same security model as the Coolify deployment:

- keep PostgreSQL persistent and backed up
- keep the JWT signing key stable
- remove the bootstrap token after first setup
- use HTTPS for remote access
- use least-privilege credentials for integrations
- never commit secrets
- keep host filesystem mounts read-only
- understand that access to `/var/run/docker.sock` grants powerful access to the Docker daemon

For stronger Docker isolation, configure a restricted Docker API proxy, set `DOCKER_HOST` to that private endpoint, and remove the raw socket mount and `group_add` configuration from `compose.docker.yml`.
