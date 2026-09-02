# Security model

## Docker is a privileged boundary

Access to `/var/run/docker.sock` is effectively root-equivalent on the Docker host. The `:ro` bind-mount flag protects the socket file itself; it does not limit API methods sent through it. Anyone who compromises a process with unrestricted socket access can normally create a privileged container and control the host.

Flare reduces exposure at its HTTP boundary: only list/inspect/log/start/stop/restart methods exist, container identifiers are constrained, destructive actions require the Administrator role, and every attempted administrative action is audited. This cannot reduce the privilege held by the API process itself. A narrowly allowlisted Docker socket proxy is the preferred production design. Do not publish that proxy, and do not give it container create, exec, delete, image build, volume mutation, or swarm permissions.

## Credentials

- Coolify and Docker credentials exist only in `Flare.Api` environment configuration.
- The Android client stores access and refresh tokens with Android secure storage. Passwords are never retained.
- Refresh tokens use 512 bits of entropy, are SHA-256 hashed in PostgreSQL, rotate on use, and revoke their token family when reuse is detected.
- `FLARE_JWT_SIGNING_KEY`, `FLARE_BOOTSTRAP_TOKEN`, database credentials, and `COOLIFY_API_TOKEN` must be supplied by the deployment secret store.
- Do not enable cleartext Android traffic or TLS validation bypasses. The production Android manifest rejects cleartext HTTP.

## Network boundary

Expose only Coolify's HTTPS route. Do not publish port 8080 directly. `FLARE_TRUST_ALL_FORWARDERS=true` is safe only when untrusted clients cannot reach the application port without passing through the trusted reverse proxy. Rotate signing/Coolify/bootstrap secrets if they are exposed.

## Deliberate exclusions

V1 contains no public registration, default credentials, container deletion, arbitrary exec, host shell, generic host commands, TLS bypass, raw Docker proxy, or raw Coolify proxy.
