# Staging on the existing Linode

The shared Linode at `104.105.15.153` already runs Caddy, Vaultwarden, and Seerr. Caddy uses host networking and owns ports 80/443. Do not run the root four-service `compose.yaml` on this host alongside that proxy. Do not stop or recreate the existing stack in `/opt/vaultwarden`.

The portfolio's separate project is `chloe-staging`, in `/opt/chloe-staging`. It uses `staging.compose.yaml` with a private `.env`, independent PostgreSQL/media/key volumes, and a loopback-only backend port `127.0.0.1:18080`. PostgreSQL has no host port. Docker Mailserver runs in the optional `mail` profile with a dedicated private submission network. Social credentials are not installed. The memory limits are for this 1 GB shared host; monitor available memory and reassess capacity before production. Keep registration disabled until verified delivery passes.

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

Other Vercel previews stay read-only. API credentials, database passwords, signing keys, and owner passwords stay off Vercel. Use the owner CLI described in [README.md](README.md) with this staging Compose file after the owner email has been chosen. Authenticator enrollment is optional and available from the account page over HTTPS. Existing enrolled factors remain enforced until their owner disables them.

## Backup and restore validation

Copy `scripts/backup-staging.sh` into the staging directory and run `bash backup-staging.sh`. It pauses only the staging backend and staging mail container, then records PostgreSQL, media/signing keys, private configuration, image IDs, mail queues/mailboxes/logs, mail configuration, and the staging mail TLS certificate. A source archive is included when present. Existing services remain running. Copy the resulting private directory off the Linode and verify `SHA256SUMS`.

For a restore drill, use a separate empty database or Compose project, restore with `pg_restore --exit-on-error`, compare content counts/IDs, extract media and signing keys privately, and compare checksums before destroying only the drill resources. Never point a drill at the live staging database or the existing service directories. The full mail-aware production backup workflow remains in [README.md](README.md).

## Staging transactional mail

The root domain’s Amazon SES MX remains unchanged. `accounts@notify.staging.chloepratas.com` sends account emails through `mail.staging.chloepratas.com`. The six isolated staging DNS records are recorded in [staging-mail-dns.txt](staging-mail-dns.txt); the BIND import is [staging-mail.zone](staging-mail.zone). The mail A record must be DNS-only. Set the server IPv4 PTR to `mail.staging.chloepratas.com` in Linode and verify it against authoritative DNS.

Append [staging-mail.Caddyfile](staging-mail.Caddyfile) to the existing Caddy configuration using the same backup, validation, inode-preserving write, and graceful reload procedure above. The mail container mounts only that hostname’s certificate directory read-only. Pinned Docker Mailserver 16.0.1 monitors these manual certificate files and reloads Postfix/Dovecot after a change; check `supervisorctl status changedetector` and the SMTP certificate after renewal.

Store the mailbox password in the root-only `.env`, with `SMTP_USER` and `MAIL_FROM` set to the sender above. Provision the mailbox in `mail-config/postfix-accounts.cf` using the mailserver setup command. Never commit the mailbox password or private DKIM key. Start only this service with `docker compose --env-file .env -f staging.compose.yaml --profile mail up -d --no-deps mail`.

Only inbound port 25 is public. Authenticated STARTTLS submission on 587 and mailbox services remain on the private Docker network. `PERMIT_DOCKER=none` requires authentication even from the backend. SPF, DKIM, and DMARC are scoped to the staging mail subdomains. Postmaster, DMARC reports, and bounces deliver into the sender mailbox. Inspect it over SSH with `docker compose --env-file .env -f staging.compose.yaml exec mail doveadm search -u accounts@notify.staging.chloepratas.com ALL`.

The shared 1 GB host uses OpenDKIM/OpenDMARC/SPF validation without the heavier Rspamd/ClamAV/Amavis daemons. This setup is intended for transactional mail, not public mailbox hosting. Monitor memory, disk, failed jobs and `postqueue -p`. Enable `REGISTRATION_ENABLED=true` only after verified delivery, relay rejection, and backups pass, then recreate only the backend. Verification remains mandatory; spam-folder delivery must be clearly communicated until inbox placement improves.
