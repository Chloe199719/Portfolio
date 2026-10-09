#!/usr/bin/env bash
set -euo pipefail
umask 077
: "${DATABASE_URL:?Use an isolated database ending in _test}"
if [[ "${CI:-}" != true ]]; then
  printf 'This runner requires CI=true and unused local test ports.\n' >&2
  exit 1
fi
python3 - <<'PY'
import os,socket
from urllib.parse import urlparse
u=urlparse(os.environ['DATABASE_URL'])
if u.hostname not in ('localhost','127.0.0.1') or not u.path.endswith('_test'):
    raise SystemExit('Use an isolated local *_test database')
for port in (int(os.getenv('E2E_SITE_PORT','3000')),4000,int(os.getenv('E2E_API_PORT','8080'))):
    with socket.socket() as s:
        if s.connect_ex(('127.0.0.1',port))==0:
            raise SystemExit('CI requires an unused port: '+str(port))
PY
run_dir=$(mktemp -d)
pids=()
cleanup() {
  for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; done
  rm -rf "$run_dir"
}
trap cleanup EXIT
export APP_ENV=development
export API_URL="http://localhost:${E2E_API_PORT:-8080}" ISSUER_URL="http://localhost:${E2E_API_PORT:-8080}"
export SITE_URL="http://localhost:${E2E_SITE_PORT:-3000}" FRONTEND_ORIGINS="http://localhost:${E2E_SITE_PORT:-3000}"
export LISTEN_ADDR=":${E2E_API_PORT:-8080}" PLAYWRIGHT_API_URL="$API_URL" PLAYWRIGHT_BASE_URL="$SITE_URL"
export DATA_DIR="$run_dir/data" E2E_FIXTURE_FILE="$run_dir/fixture.json"
export E2E_WEB_EXAMPLE_URL=http://localhost:4000 REGISTRATION_ENABLED=false
export NEXT_PUBLIC_SITE_URL="$SITE_URL" NEXT_PUBLIC_API_URL="$API_URL"
export NEXT_PUBLIC_AUTH_URL="$ISSUER_URL" NEXT_PUBLIC_READ_ONLY=false
mkdir -p test-results
wait_for() {
  for _ in {1..60}; do
    if curl -fsS "$1" >/dev/null 2>&1; then return; fi
    sleep 1
  done
  printf 'Server did not become ready: %s\n' "$1" >&2
  return 1
}
(
  cd backend
  go build -o "$run_dir/server" ./cmd/server
  go build -o "$run_dir/example" ../examples/web/main.go
)
"$run_dir/server" >test-results/backend.log 2>&1 &
pids+=("$!")
wait_for "$API_URL/v1/health"
(cd backend && go run ./cmd/testfixture) > "$E2E_FIXTURE_FILE"
python3 scripts/ci-web-client.py > "$run_dir/client.json"
CLIENT_ID=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["id"])' "$run_dir/client.json")
CLIENT_SECRET=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["secret"])' "$run_dir/client.json")
export CLIENT_ID CLIENT_SECRET
"$run_dir/example" >test-results/example.log 2>&1 &
pids+=("$!")
wait_for "$E2E_WEB_EXAMPLE_URL"
npm run build
node node_modules/next/dist/bin/next start --hostname 127.0.0.1 --port "${E2E_SITE_PORT:-3000}" >test-results/frontend.log 2>&1 &
pids+=("$!")
wait_for "$SITE_URL"
npm run test:e2e
