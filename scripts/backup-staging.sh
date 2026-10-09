#!/usr/bin/env bash
set -euo pipefail
umask 077
# Run from /opt/chloe-staging. This targets only the private staging project.
compose=(docker compose --env-file .env -f staging.compose.yaml)
backup_dir="${1:-.backups/$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$backup_dir"
running=false
if "${compose[@]}" ps --status running --services backend | grep -qx backend; then
  running=true
fi
resume() { if "$running"; then "${compose[@]}" start backend >/dev/null; fi; }
trap resume EXIT
if "$running"; then "${compose[@]}" stop backend; fi
"${compose[@]}" exec -T db pg_dump -U portfolio -d portfolio -Fc > "$backup_dir/database.dump"
"${compose[@]}" run --rm -T --no-deps --entrypoint tar backend -czf - -C /data . > "$backup_dir/backend-data.tar.gz"
tar -czf "$backup_dir/config.tar.gz" .env staging.compose.yaml
"${compose[@]}" images --format json > "$backup_dir/images.json"
if [[ -f source.tar.gz ]]; then cp source.tar.gz "$backup_dir/source.tar.gz"; fi
(
  cd "$backup_dir"
  sha256sum database.dump backend-data.tar.gz config.tar.gz images.json > SHA256SUMS
  if [[ -f source.tar.gz ]]; then sha256sum source.tar.gz >> SHA256SUMS; fi
)
printf 'Staging backup: %s\n' "$backup_dir"
