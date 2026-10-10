#!/usr/bin/env bash
set -euo pipefail
umask 077
backup_dir="${1:-.backups/$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$backup_dir"
# Briefly stop writers for a matching database, key, file, and mail-queue snapshot.
resume_services=()
for service in backend mail; do
  if docker compose ps --status running --services "$service" | rg -q "^${service}$"; then
    resume_services+=("$service")
  fi
done
resume() { if ((${#resume_services[@]})); then docker compose start "${resume_services[@]}" >/dev/null; fi; }
trap resume EXIT
if ((${#resume_services[@]})); then docker compose stop "${resume_services[@]}"; fi
docker compose exec -T db pg_dump -U portfolio -d portfolio -Fc > "$backup_dir/database.dump"
docker compose run --rm -T --no-deps --entrypoint tar backend -czf - -C /data . > "$backup_dir/backend-data.tar.gz"
docker compose run --rm -T --no-deps --entrypoint tar mail -czf - /var/mail /var/mail-state > "$backup_dir/mail-data.tar.gz"
tar -czf "$backup_dir/host-config.tar.gz" .env data/mail-config data/caddy deploy/Caddyfile compose.yaml
cp backend/go.mod "$backup_dir/go.mod"
git rev-parse HEAD > "$backup_dir/revision.txt"
docker compose images --format json > "$backup_dir/images.json"
(cd "$backup_dir" && shasum -a 256 database.dump backend-data.tar.gz mail-data.tar.gz host-config.tar.gz > SHA256SUMS)
printf 'Complete backup saved to %s. Copy it to a separate encrypted backup destination.\n' "$backup_dir"
