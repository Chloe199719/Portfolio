#!/usr/bin/env bash
set -euo pipefail
# Only the disposable job runner's daemon is changed. Never run on Linode.
if [[ "${GITHUB_ACTIONS:-}" != true || "${RUNNER_ENVIRONMENT:-}" != github-hosted || "${RUNNER_OS:-}" != Linux ]]; then
  printf 'This setup is only for GitHub-hosted Linux runners.\n' >&2
  exit 1
fi
sudo python3 - <<'PY'
import json
from pathlib import Path
path = Path('/etc/docker/daemon.json')
config = json.loads(path.read_text()) if path.exists() else {}
mirrors = config.setdefault('registry-mirrors', [])
mirror = 'https://mirror.gcr.io'
if mirror not in mirrors:
    mirrors.append(mirror)
path.write_text(json.dumps(config) + '\n')
PY
sudo systemctl restart docker
docker info --format '{{json .RegistryConfig.Mirrors}}'
