#!/usr/bin/env bash
set -euo pipefail
umask 077
# Run from /opt/chloe-staging. This targets only the private staging project.
compose=(docker compose --env-file .env -f staging.compose.yaml --profile mail)
backup_dir="${1:-.backups/$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$backup_dir"
running=false
mail_running=false
if "${compose[@]}" ps --status running --services backend | grep -qx backend; then
  running=true
fi
if "${compose[@]}" ps --status running --services mail | grep -qx mail; then
  mail_running=true
fi
resume() {
  if "$mail_running"; then "${compose[@]}" start mail >/dev/null; fi
  if "$running"; then "${compose[@]}" start backend >/dev/null; fi
}
trap resume EXIT
if "$running"; then "${compose[@]}" stop backend; fi
if "$mail_running"; then "${compose[@]}" stop mail; fi
"${compose[@]}" exec -T db pg_dump -U portfolio -d portfolio -Fc > "$backup_dir/database.dump"
"${compose[@]}" run --rm -T --no-deps --entrypoint tar backend -czf - -C /data . > "$backup_dir/backend-data.tar.gz"
tar -czf "$backup_dir/config.tar.gz" .env staging.compose.yaml
if [[ -d mail-config ]]; then
  "${compose[@]}" run --rm -T --no-deps --user 0 --entrypoint tar mail \
    -czf - -C / var/mail var/mail-state var/log/mail > "$backup_dir/mail-data.tar.gz"
  tar -czf "$backup_dir/mail-config.tar.gz" mail-config staging-mail.Caddyfile
  # Preserve only this portfolio's mail certificate, never unrelated sites' keys.
  tar -czf "$backup_dir/mail-tls.tar.gz" \
    -C /opt/vaultwarden/caddy-data/caddy/certificates/acme-v02.api.letsencrypt.org-directory \
    mail.staging.chloepratas.com
fi
"${compose[@]}" images --format json > "$backup_dir/images.json"
if [[ -f source.tar.gz ]]; then cp source.tar.gz "$backup_dir/source.tar.gz"; fi
(
  cd "$backup_dir"
  sha256sum database.dump backend-data.tar.gz config.tar.gz images.json > SHA256SUMS
  if [[ -f source.tar.gz ]]; then sha256sum source.tar.gz >> SHA256SUMS; fi
  if [[ -f mail-data.tar.gz ]]; then
    sha256sum mail-data.tar.gz mail-config.tar.gz mail-tls.tar.gz >> SHA256SUMS
  fi
)
printf 'Staging backup: %s\n' "$backup_dir"
