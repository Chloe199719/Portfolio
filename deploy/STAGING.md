# Staging on the existing Linode

The shared Linode at `104.105.15.153` already runs Caddy, Vaultwarden, and Seerr. Caddy uses host networking and owns ports 80/443. Do not run the root four-service `compose.yaml` on this host alongside that proxy. Do not stop or recreate the existing stack in `/opt/vaultwarden`.

The portfolio's separate project is `chloe-staging`, in `/opt/chloe-staging`. It uses `staging.compose.yaml` with a private `.env`, independent PostgreSQL/media/key volumes, and a loopback-only backend port `127.0.0.1:18080`. PostgreSQL has no host port. Registration is disabled, and no mail or social credentials are installed. The memory limits are for initial staging on this 1 GB shared host; reassess capacity before production and self-hosted mail.

## Operate only the portfolio

```sh
cd /opt/chloe-staging
docker compose --env-file .env -f staging.compose.yaml ps
docker compose --env-file .env -f staging.compose.yaml logs --tail 50 backend
docker compose --env-file .env -f staging.compose.yaml exec backend server check
curl -fsS -H 'Host: api.staging.chloepratas.com' http://127.0.0.1:18080/v1/health
```

Build images on the development machine, targeting the server architecture, and transfer them over SSH. Both Dockerfiles cross-compile Go using the builder's native architecture.

```sh
docker buildx build --platform linux/amd64 --load -f backend/Dockerfile -t chloe-backend:YOUR_RELEASE .
set -o pipefail
docker save chloe-backend:YOUR_RELEASE | gzip | ssh root@104.105.15.153 docker load
```

Update only `BACKEND_IMAGE` in the remote `.env` after backing up; use the staging Compose file to recreate the backend. Keep old image tags for rollback. Never copy local test accounts, test signing keys, fixture sessions, or local database credentials onto this host.

## Enable HTTPS with the existing Caddy

1. Add DNS-only Cloudflare A records `api.staging` and `auth.staging`, both pointing to `104.105.15.153`.
2. Preserve a private backup of `/opt/vaultwarden/Caddyfile`. Prepare a candidate that keeps its global options and existing blocks byte-for-byte and appends `deploy/staging.Caddyfile`. Inspect and validate the complete candidate using the installed Caddy version before applying it. Do not replace the file with the snippet alone.
3. Apply through Caddy's graceful `reload`, without restarting its container. Check the existing sites before and after, and restore the previous configuration if validation or routing fails. Preserve the bind-mounted file's inode when updating its contents.
4. Check trusted HTTPS certificates, `/v1/health`, OIDC discovery, and host routing. The proxy must replace `X-Real-IP`; the Go port must remain loopback-only. The staging snippet is prepared separately and is not automatically loaded by the Compose stack.

The frontend uses Vercel with `staging.chloepratas.com`. Get the exact DNS target from the selected Vercel project. Set these public environment values on that staging deployment only:

```dotenv
NEXT_PUBLIC_SITE_URL=https://staging.chloepratas.com
NEXT_PUBLIC_API_URL=https://api.staging.chloepratas.com
NEXT_PUBLIC_AUTH_URL=https://auth.staging.chloepratas.com
NEXT_PUBLIC_READ_ONLY=false
```

Other Vercel previews stay read-only. API credentials, database passwords, signing keys, and owner passwords stay off Vercel. Use the owner CLI described in [README.md](README.md) with this staging Compose file after the owner email has been chosen. Complete authenticator enrollment over HTTPS.

## Backup and restore validation

Copy `scripts/backup-staging.sh` into the staging directory and run `bash backup-staging.sh`. It pauses only the staging backend, then records a matching PostgreSQL dump, media/signing keys, private configuration, image IDs, and a source archive when present. Existing services remain running. Copy the resulting private directory off the Linode and verify `SHA256SUMS`.

For a restore drill, use a separate empty database or Compose project, restore with `pg_restore --exit-on-error`, compare content counts/IDs, extract media and signing keys privately, and compare checksums before destroying only the drill resources. Never point a drill at the live staging database or the existing service directories. The full mail-aware production backup workflow remains in [README.md](README.md).

Mail migration is pending: the domain currently has an Amazon SES inbound MX record. Review that setup and DNS ownership before changing MX, SPF, DKIM, DMARC, or starting a mail container.
