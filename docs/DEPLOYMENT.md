# Production deployment

## Coolify Docker Compose service

Create a PostgreSQL resource and a Docker Compose application from this repository. Use `/` as the base directory and `/compose.yml` as the Docker Compose location. If PostgreSQL is a separate Coolify resource, enable **Connect to Predefined Network** and use its full internal service hostname in the connection string.

Assign an HTTPS domain to the `api` service and route it to container port `8080`. Do not add a host `ports` mapping: `compose.yml` exposes the port only to Coolify's proxy network. Keep Coolify's HTTP-to-HTTPS redirect enabled at the public edge; Flare.Api intentionally serves HTTP on the private proxy network and does not issue its own HTTPS redirect. Forwarded headers let it recognize the original public HTTPS scheme. The address entered in the Android app is the public URL, for example `https://flare.example.com`, with no `/api` suffix. `/health/live` is the container health check and `/health/ready` additionally verifies PostgreSQL connectivity.

The root `Dockerfile` publishes only `Flare.Api` into a .NET 10 chiseled runtime image, runs as the built-in non-root user, listens on port 8080, and handles SIGTERM through ASP.NET Core's normal graceful shutdown.

Configure these production variables:

| Variable | Required | Purpose |
|---|---:|---|
| `ConnectionStrings__Postgres` | yes | Coolify PostgreSQL connection string |
| `FLARE_JWT_SIGNING_KEY` | yes | At least 32 random bytes; retain across restarts |
| `COOLIFY_BASE_URL` | yes | Trusted self-hosted HTTPS base URL, without `/api/v1` |
| `COOLIFY_API_TOKEN` | yes | Coolify token with only the required `read` and `deploy` permissions |
| `DOCKER_SOCKET_GID` | yes | Numeric group ID owning `/var/run/docker.sock` on the deployment server |
| `FLARE_BOOTSTRAP_TOKEN` | first boot | High-entropy one-time first-admin token; clear it after bootstrap |
| `DOCKER_HOST` | no | Defaults to `unix:///var/run/docker.sock` |
| `FLARE_HOST_NAME` | no | Display name, default `homelab` |
| `FLARE_ACCESS_TOKEN_MINUTES` | no | 5–60, default 15 |
| `FLARE_REFRESH_TOKEN_DAYS` | no | 1–90, default 30 |
| `FLARE_JWT_ISSUER` / `FLARE_JWT_AUDIENCE` | no | Token validation names |

`compose.yml` fixes `ASPNETCORE_ENVIRONMENT=Production`, `ASPNETCORE_URLS=http://+:8080`, the mounted host telemetry paths, and `FLARE_TRUST_ALL_FORWARDERS=true`. The latter is safe here because the Compose definition does not publish port 8080 directly; traffic reaches it through Coolify's proxy network.

Find the Docker socket group ID on the Coolify server with:

```bash
stat -c '%g' /var/run/docker.sock
```

Enter the result as `DOCKER_SOCKET_GID`. Configure all secrets as runtime-only Coolify variables. If a value contains `$`, enable Coolify's **Literal** option so Compose does not interpolate it. The Android app needs none of these secrets; it only stores the public HTTPS URL and authentication tokens.

The required values in Coolify's Developer view have this shape (replace every placeholder):

```dotenv
ConnectionStrings__Postgres=Host=<postgres-service-name>;Port=5432;Database=<database>;Username=<username>;Password=<password>;SSL Mode=Prefer
FLARE_JWT_SIGNING_KEY=<at-least-32-random-bytes>
COOLIFY_BASE_URL=https://<your-coolify-domain>
COOLIFY_API_TOKEN=<least-privilege-read-and-deploy-token>
DOCKER_SOCKET_GID=<numeric-gid>
FLARE_BOOTSTRAP_TOKEN=<one-time-random-token>
FLARE_HOST_NAME=<display-name>
```

Only `FLARE_BOOTSTRAP_TOKEN` should be cleared after the first administrator has been created. Retain the signing key and database credentials across every restart and deployment; changing the signing key immediately invalidates all access tokens.

Never place secrets in the repository. `.env.example` contains placeholders only.
The chiseled image disables PostgreSQL GSS session encryption because Kerberos libraries are intentionally absent; configure `SSL Mode` in `ConnectionStrings__Postgres` when database transport encryption is required.

## Database migrations

Flare never calls `EnsureCreated`, drops, or recreates the production database. Migrations are compiled into the app. In the Compose deployment, the one-shot `migrate` service applies pending migrations and must exit successfully before `api` starts. Coolify is instructed to exclude that completed one-shot container from ongoing health checks.

Back up PostgreSQL before deploying a release containing schema changes. For a standalone Dockerfile deployment, run the exact image as a one-off/pre-deployment command:

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

dotnet build src\Flare.Mobile\Flare.Mobile.csproj -c Release -r android-arm64
```

The signed artifact is written as `src/Flare.Mobile/bin/Release/net10.0-android/android-arm64/Flare-v1.1.0-android-arm64.apk`. Its name comes from `ApplicationDisplayVersion`; the Android manifest receives the same semantic `versionName` and the integer `ApplicationVersion` as `versionCode`.

Before every release, increase `ApplicationDisplayVersion` using semantic versioning and increase `ApplicationVersion` to an integer greater than every previously published build. Always use the same release keystore and alias. Debug builds and CI use debug signing and must not be distributed.

Install an update with `adb install -r <apk-path>` or open the APK normally on the device. APKs published before the stable release key was introduced cannot be upgraded in place because their signing certificate differs; uninstall one of those builds once, then install the first stable-key release. Later stable-key releases upgrade normally.
