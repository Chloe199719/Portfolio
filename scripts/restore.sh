#!/usr/bin/env bash
set -euo pipefail
umask 077
if [[ $# -ne 2 || "$2" != '--confirm' ]]; then
  printf 'Usage: bash scripts/restore.sh BACKUP_DIRECTORY --confirm\nThis replaces application data. Make a current backup first.\n' >&2
  exit 1
fi
backup_dir="$1"
(cd "$backup_dir" && shasum -a 256 -c SHA256SUMS)
docker compose stop caddy backend mail
docker compose exec -T db pg_restore -U portfolio -d portfolio --clean --if-exists --no-owner --exit-on-error < "$backup_dir/database.dump"
# Restore the complete private data directory, including the active signing key and HMAC secret.
docker compose run --rm -T --no-deps --entrypoint sh backend -c 'find /data -mindepth 1 -maxdepth 1 -exec rm -rf {} +; tar -xzf - -C /data' < "$backup_dir/backend-data.tar.gz"
docker compose run --rm -T --no-deps --entrypoint sh mail -c 'find /var/mail /var/mail-state -mindepth 1 -maxdepth 1 -exec rm -rf {} +; tar -xzf - -C /' < "$backup_dir/mail-data.tar.gz"
tar -xzf "$backup_dir/host-config.tar.gz"
docker compose up -d db caddy backend mail
printf 'Restored. Verify API health, issuer discovery, existing token verification, media privacy, and mail before reopening registration.\n'
