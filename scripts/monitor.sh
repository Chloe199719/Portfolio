#!/usr/bin/env bash
set -euo pipefail
# Run on the Linode; nonzero exit means the host monitor should alert.
docker compose exec -T backend server check
failed_jobs=$(docker compose exec -T db psql -U portfolio -d portfolio -At -v ON_ERROR_STOP=1 -c "SELECT (SELECT count(*) FROM schedules WHERE status='failed' OR (status='pending' AND run_at<now()-interval '2 minutes'))+(SELECT count(*) FROM email_jobs WHERE status='failed' OR (status='pending' AND run_at<now()-interval '10 minutes'));")
docker compose exec -T db psql -U portfolio -d portfolio -v ON_ERROR_STOP=1 -c "SELECT status,count(*) FROM email_jobs GROUP BY status; SELECT status,count(*) FROM schedules GROUP BY status;"
docker compose exec -T mail postqueue -p
df -h .
disk_percent=$(df -P . | awk 'NR==2 {gsub(/%/,"",$5);print $5}')
failed=0
if ((failed_jobs > 0)); then
  printf '%s failed or overdue jobs need attention.\n' "$failed_jobs" >&2
  failed=1
fi
if ((disk_percent >= 85)); then
  printf 'Disk use is %s%%.\n' "$disk_percent" >&2
  failed=1
fi
if ! find .backups -maxdepth 2 -name SHA256SUMS -mtime -2 -print -quit | rg -q .; then
  printf 'No complete backup from the past two days.\n' >&2
  failed=1
fi
exit "$failed"
