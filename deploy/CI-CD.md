# GitHub Actions deployment

`Checks` runs on pull requests. `Deploy portfolio` runs the same checks for each push to `main`, then automatically deploys staging when the repository variable `DEPLOYMENTS_ENABLED` is `true`. Use its **Run workflow** action on `main` to deploy production after staging acceptance. Production additionally requires the environment variable `PRODUCTION_READY=true` and its separate prepared stack/key.

Checks cover lint, types, generated API consistency, frontend unit tests/build, Go vet/race integration tests with PostgreSQL 18, and desktop/mobile browser tests against disposable local services. Fork pull requests receive no deployment credentials. Actions are pinned to commit IDs; the Vercel CLI is pinned to a version.

Deployment builds one Go AMD64 image named for the commit, transfers it over verified SSH, backs up the existing portfolio data, and replaces only the portfolio backend container. Health failure restores the previous image. Database migrations must remain backward compatible: automatic image rollback does not undo schema/data changes. The frontend uses Vercel's pull/build/prebuilt deployment flow. Staging receives its stable custom-domain alias through the deployment API, which supports project-scoped tokens without the CLI alias command's account lookup. The workflow validates the deployment JSON and the alias response. Production uses `--prod`. `vercel.json` disables duplicate Git-triggered Vercel builds so they cannot bypass checks or replace production during staging. `.vercelignore` explicitly allows only frontend source; local Git ignores alone do not protect CLI uploads.

## Required GitHub configuration

Repository: `Chloe199719/Portfolio`. Create **staging** and **production** environments under Settings → Environments. Restrict deployment branches to `main`. Add a required reviewer for production if the repository's plan supports it; otherwise retain the explicit release action and `PRODUCTION_READY` gate.

Set these environment variables (GitHub **Variables**, not workflow source):

| Variable            | Staging value                                 |
| ------------------- | --------------------------------------------- |
| `LINODE_HOST`       | `104.105.15.153`                              |
| `LINODE_USER`       | `root` with a dedicated forced-command key    |
| `SITE_URL`          | `https://staging.chloepratas.com`             |
| `API_URL`           | `https://api.staging.chloepratas.com`         |
| `AUTH_URL`          | `https://auth.staging.chloepratas.com`        |
| `VERCEL_ORG_ID`     | Team ID from the Vercel project link/settings |
| `VERCEL_PROJECT_ID` | Project ID for `portfolio`                    |

Set these environment **Secrets**:

| Secret                   | Purpose                                                             |
| ------------------------ | ------------------------------------------------------------------- |
| `LINODE_SSH_PRIVATE_KEY` | The new deployment-only private key, never the personal SSH key     |
| `LINODE_KNOWN_HOSTS`     | Verified server host-key entry copied from a trusted SSH connection |
| `VERCEL_TOKEN`           | Token authorized for the selected Vercel team/project               |

Set the repository variable `DEPLOYMENTS_ENABLED=true` only after all staging secrets, DNS, HTTPS, the Vercel domain, and the receiver are ready. Without it, checks run and deployment is skipped. Production gets separate environment secrets, its own forced-command key, URLs, and stack. Do not point production credentials at staging.

## Restricted Linode receiver

Install `deploy/receive-deployment.sh` as root-owned `/usr/local/sbin/chloe-deploy`, mode 0755. Install `scripts/backup-staging.sh` in `/opt/chloe-staging/backup-staging.sh`. Keep the directory, `.env`, Compose file and scripts root-owned and unwritable by application containers. Add only the dedicated public key to root's `authorized_keys` with:

```text
restrict,command="/usr/local/sbin/chloe-deploy staging" ssh-ed25519 PUBLIC_KEY portfolio-staging-actions
```

The key allows only `deploy FULL_40_CHARACTER_COMMIT_SHA` with a bounded gzip Docker archive on stdin. Shells, PTYs, forwarding, arbitrary commands and other image tags are rejected. The receiver uses fixed portfolio paths and services; it never edits/reloads Caddy or touches Vaultwarden/Seerr. It rejects concurrent deployments using a host lock. Its root authorization still makes this a sensitive credential: store it only in the protected GitHub environment and rotate by replacing that one authorized-key entry.

The Compose configuration explicitly fixes the backend UID/GID, memory limits, mounts, networks and dropped capabilities. CI cannot upload Compose files or deployment scripts. Infrastructure changes are applied separately after review. For production, prepare `/opt/chloe-production` with the same two-service Compose layout and an appropriate `.env`, a separate loopback port, volumes/project name, and capacity settings before installing a production-scoped key. Mail and the shared Caddy are managed separately.

## Release and rollback

Check the Actions run, its deployment URL, API health, login and the existing Linode services after each release. Each successful receiver run records `current-release`; each backup records the prior image and matching database/files/keys/config. Keep previous image tags. For a code-only rollback, restore the previous image value and recreate only `backend`. For a data/schema rollback, restore the matching private backup in a maintenance window following the restore runbook.

The current source tree must be committed and pushed before GitHub can run these workflows. Staging should pass before the first production release; `vercel.json` prevents that push from independently replacing the existing production website.

References: [Vercel's GitHub Actions flow](https://vercel.com/kb/guide/how-can-i-use-github-actions-with-vercel), [GitHub deployment environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/control-deployments), [Vercel Git deployment control](https://vercel.com/docs/project-configuration/git-configuration#git.deploymentenabled).

The staging alias script follows [Vercel's Assign an Alias API](https://vercel.com/docs/rest-api/aliases/assign-an-alias). See [Vercel's upload exclusions](https://vercel.com/docs/deployments/vercel-ignore) before changing the frontend allowlist.
