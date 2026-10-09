#!/usr/bin/env bash
set -euo pipefail
umask 077
# Package installation is restricted to disposable GitHub-hosted Linux runners.
if [[ "${GITHUB_ACTIONS:-}" != true || "${RUNNER_ENVIRONMENT:-}" != github-hosted || "${RUNNER_OS:-}" != Linux ]]; then
  printf 'This setup is only for GitHub-hosted Linux runners.\n' >&2
  exit 1
fi
: "${RUNNER_TEMP:?Missing runner temporary directory}"
pg_bin=/usr/lib/postgresql/18/bin
if [[ "${1:-}" == stop ]]; then
  if [[ -n "${PG18_TEST_DATA_DIR:-}" ]]; then
    [[ "$PG18_TEST_DATA_DIR" == "$RUNNER_TEMP"/portfolio-pg.* ]]
    if "$pg_bin/pg_ctl" -D "$PG18_TEST_DATA_DIR" status >/dev/null 2>&1; then
      "$pg_bin/pg_ctl" -D "$PG18_TEST_DATA_DIR" -m fast -w stop
    fi
  fi
  exit 0
fi
database="${1:-}"
[[ "$database" =~ ^portfolio_([a-z0-9]+_)*test$ ]] || { printf 'Expected an isolated portfolio_*test database.\n' >&2; exit 1; }
: "${GITHUB_ENV:?Missing runner environment file}"
if [[ ! -x "$pg_bin/initdb" ]]; then
  # PostgreSQL's signed Ubuntu repository, independent of container registries.
  source /etc/os-release
  curl --fail --silent --show-error --retry 3 \
    https://www.postgresql.org/media/keys/ACCC4CF8.asc -o "$RUNNER_TEMP/portfolio-pgdg.asc"
  sudo install -m 644 "$RUNNER_TEMP/portfolio-pgdg.asc" /usr/share/keyrings/portfolio-pgdg.asc
  printf 'deb [signed-by=/usr/share/keyrings/portfolio-pgdg.asc] https://apt.postgresql.org/pub/repos/apt %s-pgdg main\n' "$VERSION_CODENAME" \
    | sudo tee /etc/apt/sources.list.d/portfolio-pgdg.list >/dev/null
  sudo apt-get update -o Acquire::Retries=3
  sudo env DEBIAN_FRONTEND=noninteractive apt-get install --yes --no-install-recommends postgresql-18
fi
data=$(mktemp -d "$RUNNER_TEMP/portfolio-pg.XXXXXX")
password_file=$(mktemp "$RUNNER_TEMP/portfolio-pg-password.XXXXXX")
trap 'rm -f "$password_file"' EXIT
printf '%s\n' ci-test-only > "$password_file"
"$pg_bin/initdb" -D "$data" -U portfolio --auth-local=trust --auth-host=scram-sha-256 --pwfile="$password_file" >/dev/null
printf 'PG18_TEST_DATA_DIR=%s\n' "$data" >> "$GITHUB_ENV"
"$pg_bin/pg_ctl" -D "$data" -l "$RUNNER_TEMP/portfolio-postgres.log" \
  -o "-h 127.0.0.1 -p 55432 -k $data" -w start
PGPASSWORD=ci-test-only "$pg_bin/createdb" -h 127.0.0.1 -p 55432 -U portfolio "$database"
"$pg_bin/pg_isready" -h 127.0.0.1 -p 55432 -U portfolio -d "$database"
