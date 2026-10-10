#!/usr/bin/env bash
# Install root-owned at /usr/local/sbin/chloe-deploy, mode 0755.
# Authorized key: restrict,command="/usr/local/sbin/chloe-deploy staging" ssh-ed25519 ...
set -euo pipefail
umask 077
export PATH=/usr/sbin:/usr/bin:/sbin:/bin
case "${1:-}" in
  staging) root=/opt/chloe-staging ;;
  production) root=/opt/chloe-production ;;
  *) printf 'Unknown environment.\n' >&2; exit 64 ;;
esac
if [[ ! "${SSH_ORIGINAL_COMMAND:-}" =~ ^deploy\ ([0-9a-f]{40})$ ]]; then
  printf 'Only deploy followed by a full commit SHA is accepted.\n' >&2
  exit 64
fi
revision="${BASH_REMATCH[1]}"
image="chloe-backend:sha-$revision"
cd "$root"
exec 9>.deploy.lock
flock -n 9 || { printf 'Another deployment is running.\n' >&2; exit 75; }
compose=(docker compose --env-file .env -f staging.compose.yaml)
"${compose[@]}" config --quiet
temporary=$(mktemp -d "$root/.deploy-XXXXXXXX")
trap 'rm -rf "$temporary"' EXIT
# Accept only a bounded Docker archive for exactly this portfolio release.
python3 -c '
import gzip,sys,tarfile,json
path,expected=sys.argv[1:]
total=0
with gzip.GzipFile(fileobj=sys.stdin.buffer) as source, open(path,"xb") as target:
    while True:
        chunk=source.read(1024*1024)
        if not chunk: break
        total+=len(chunk)
        if total>512*1024*1024: raise SystemExit("Image exceeds 512 MiB")
        target.write(chunk)
with tarfile.open(path) as archive:
    member=archive.getmember("manifest.json")
    if member.size>1024*1024: raise SystemExit("Invalid manifest size")
    manifests=json.load(archive.extractfile(member))
    if len(manifests)!=1 or manifests[0].get("RepoTags")!=[expected]:
        raise SystemExit("Only the requested portfolio image tag may be loaded")
' "$temporary/image.tar" "$image"
docker load -i "$temporary/image.tar"
[[ $(docker image inspect "$image" --format '{{.Os}}/{{.Architecture}}') == linux/amd64 ]]
# This script and Compose configuration are maintained separately from CI input.
bash backup-staging.sh
cp .env "$temporary/previous.env"
previous=$(python3 -c 'from pathlib import Path; print(next(x.split("=",1)[1] for x in Path(".env").read_text().splitlines() if x.startswith("BACKEND_IMAGE=")))')
python3 - "$image" <<'PY'
from pathlib import Path
import os,sys
p=Path('.env')
lines=p.read_text().splitlines()
assert sum(line.startswith('BACKEND_IMAGE=') for line in lines)==1
new='\n'.join('BACKEND_IMAGE='+sys.argv[1] if line.startswith('BACKEND_IMAGE=') else line for line in lines)+'\n'
temp=Path('.env.next')
temp.write_text(new)
os.chmod(temp,0o600)
temp.replace(p)
PY
rollback() {
  cp "$temporary/previous.env" .env
  printf 'Deployment failed; restoring previous backend image %s.\n' "$previous" >&2
  "${compose[@]}" up -d --no-deps --wait --wait-timeout 120 backend || true
}
if ! "${compose[@]}" up -d --no-deps --wait --wait-timeout 120 backend; then
  rollback
  exit 1
fi
if ! "${compose[@]}" exec -T backend server check; then
  rollback
  exit 1
fi
printf '%s\n' "$revision" > current-release
printf 'Deployed %s; previous image %s.\n' "$revision" "$previous"
