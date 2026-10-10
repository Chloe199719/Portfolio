# Local validation — 8 October 2026

Implementation is validated locally. Staging deployment progress is recorded below; the existing production website has not been replaced. Real-domain social login/mail delivery have not been exercised.

| Check                                   | Result                                                                                                                                                                     |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Frontend TypeScript, ESLint, formatting | Passed                                                                                                                                                                     |
| Frontend unit tests                     | 18 passed, including strict API response handling and Berlin summer/winter and daylight-saving transitions                                                                 |
| Next.js production build                | Passed; public content routes render on demand from Go                                                                                                                     |
| Go integration tests and race detector  | 7 passed against isolated PostgreSQL schemas                                                                                                                               |
| `go vet`                                | Passed                                                                                                                                                                     |
| Browser tests                           | 13 passed; 3 duplicate mobile authentication cases intentionally skipped. Final production build also passed both desktop/mobile public-page checks                        |
| Independent web example                 | Browser completed confidential code + PKCE, login, consent, token exchange, UserInfo, and its own session                                                                  |
| Passkeys                                | Chromium virtual authenticator completed creation, sign-out, sign-in, and removal                                                                                          |
| Native example                          | Swift type checking passed; public native-client code + PKCE flow tested against Go                                                                                        |
| Frontend production dependency audit    | Zero reported vulnerabilities                                                                                                                                              |
| Go vulnerability scan                   | Zero reachable vulnerabilities; one advisory remains in an unused imported package/transitive module path                                                                  |
| Docker                                  | Final Go image built; backend, PostgreSQL 18, Caddy, and Docker Mailserver started in an isolated Compose project                                                          |
| Caddy                                   | Production configuration validated; local drill used local certificates and loopback-only ports                                                                            |
| SMTP                                    | Authenticated local submission accepted; unauthenticated relay rejected with 554 on ports 25 and 587                                                                       |
| Backup/restore                          | Actual scripts restored PostgreSQL, media, signing keys, mail data/state, and host configuration; deleted DB/media fixtures recovered and media/key SHA-256 hashes matched |

The backend suite verifies draft privacy, owner/MFA restrictions, revision conflicts, immutable scheduled revisions, concurrent publication, trash restoration, CSRF, guestbook moderation/author deletion, password recovery, expired sessions, registration verification, account isolation, PKCE, code reuse, refresh replay, and signing-key rotation. Media tests verify private/public transitions, metadata removal, and that mentioning an asset filename in text cannot publish it.

Browser tests verify routes, real images, responsive overflow, metadata/canonicals/RSS/sitemap, keyboard and reduced-motion game behavior, contact failure handling, preservation of newer typing during saves, local draft recovery, private previews, publishing/unpublishing, history restoration, and schedule cancellation.

The local restore stack is separate from the frontend preview. No user production database or mail system was replaced. The original content backup remains in the ignored `.backups` directory.

## Remaining staging and production checks

- Configure real staging/custom domains, DNS, certificates, provider credentials, owner enrollment, and Vercel/Linode environments.
- Exercise Google and GitHub success/cancellation/account-conflict/linking using the actual provider applications.
- Confirm Linode outbound SMTP access, external inbox delivery, SPF/DKIM/DMARC/PTR, bounce reception, and certificate renewal.
- Run OAuth/passkeys on a physical mobile device.
- Configure the host monitor and encrypted off-host backups, then preserve the prior deployment before production replacement.

See [the deployment runbook](README.md) for commands and configuration. The local tests do not certify OIDC conformance or substitute for a security review of an independently operated identity provider.

## Staging progress — 9 October 2026

- Separate Go/PostgreSQL containers are healthy on the shared Linode. PostgreSQL is private and the API listens only on loopback port 18080. Registration is disabled. Existing Caddy, Vaultwarden, and Seerr containers were not restarted or reconfigured.
- The restricted deployment key rejected a general shell command and successfully deployed backend commit `373ca03890e7151c8ca99eecf32359fa2a27a821`, with backup and health checks. Failure rollback is implemented but has not yet been exercised against the shared host.
- A target-host restore drill recovered the database into a separate disposable database; content and owner counts matched, as did signing-key checksums. An off-host copy of the initial backup passed checksum verification.
- The additive Caddy candidate validates and preserves the existing site blocks. It has not been applied: `api.staging` and `auth.staging` still need DNS-only records in Cloudflare before HTTPS acceptance.
- Vercel successfully built Node 24 from a clean Linux install after repairing missing optional lockfile dependencies. Production dependencies report zero audit vulnerabilities; development tooling still reports seven high-severity audit findings that need review before production release.
- `https://staging.chloepratas.com` serves a read-only preview from the versioned public content export. API/auth URLs are deliberately unset for this preview. Rebuild with the live staging backend URLs and `NEXT_PUBLIC_READ_ONLY=false` once DNS/HTTPS are ready.
- The project-scoped Vercel token was verified against the project API, configuration pull, preview creation, and staging alias assignment through the API. Its CLI account-inspection/alias commands require broader account access, which was not granted.
- GitHub Actions definitions are on `codex/portfolio-staging`. GitHub CLI authorization/environment secrets and a hosted workflow run are still pending; continuous deployment is not enabled yet.

## GitHub delivery setup — 10 October 2026

- GitHub CLI authorization is complete. The `staging` environment allows deployment only from `main` and contains encrypted `VERCEL_TOKEN`, `LINODE_SSH_PRIVATE_KEY`, and `LINODE_KNOWN_HOSTS` secrets plus the public deployment variables. The temporary local Vercel token copies were removed after verifying GitHub storage.
- [Hosted checks passed](https://github.com/Chloe199719/Portfolio/actions/runs/38002066784) for implementation commit `a7a107924af77df7fd516c5127ff3f1ffda4fe03`: frontend generation/lint/types/tests/build, Go vet/race integration tests, desktop/mobile browser tests, and the deployable backend image build.
- The initial hosted run hit Docker Hub's unauthenticated pull limit before PostgreSQL could start. CI now installs PostgreSQL 18 from its signed Ubuntu repository into a dedicated test cluster, and image builds use the documented Docker Hub cache on disposable runners. Setup scripts reject non-GitHub-hosted environments.
- [Draft PR #1](https://github.com/Chloe199719/Portfolio/pull/1) contains the implementation and workflows. `DEPLOYMENTS_ENABLED=false` remains in place until backend DNS, HTTPS, and staging acceptance are complete. The frontend remains a read-only preview; existing Linode services remain unchanged.

## Staging HTTPS setup — 10 October 2026

- Added only the two staging site blocks to the existing Caddy configuration after backing it up and validating the combined configuration. Applied a graceful reload while preserving the bind-mounted file inode. Vaultwarden, Seerr, and Jellyfin returned their original HTTP statuses before and after; no containers were restarted.
- Caddy obtained trusted Let's Encrypt certificates for `api.staging.chloepratas.com` and `auth.staging.chloepratas.com`. Direct HTTPS checks against Linode passed for API health, published content, OIDC discovery, public signing keys, frontend CORS, secure session cookies, owner-only endpoints, origin rejection, and hostname routing separation.
- The pre-change Caddy backup is `/opt/chloe-staging/.backups/caddy-20261010T082848Z`. The live backend still uses release `373ca03890e7151c8ca99eecf32359fa2a27a821`.
- Built an API-connected Vercel preview successfully at `https://portfolio-awvupnd12-chloe-pratas-projects.vercel.app` with the staging API/auth URLs and `NEXT_PUBLIC_READ_ONLY=false`. It has not replaced the staging alias.
- Public connectivity remains blocked: authoritative DNS over TCP, an independent HTTPS resolver, and Linode still returned Cloudflare proxy addresses after the reported DNS change. Public HTTPS fails at the proxy while direct Linode HTTPS succeeds. Verify that **both** API/auth records have saved **DNS only** status, then recheck public HTTPS and switch the staging alias to the prepared deployment. Existing production deployment and continuous-deployment gates remain unchanged.

## Connected staging acceptance — 10 October 2026

- The DNS proxy blocker above is resolved. Both staging API/auth A records now resolve to `104.105.15.153`, with no authoritative AAAA records. Public HTTPS health and identity discovery return HTTP 200 from the local machine and Linode.
- Switched `https://staging.chloepratas.com` to deployment `dpl_GWDch87kZWfrT37FoJqSHb1JGW41` (`https://portfolio-awvupnd12-chloe-pratas-projects.vercel.app`). The frontend now uses the live Go API and has `NEXT_PUBLIC_READ_ONLY=false`. The previous read-only deployment remains available for rollback.
- Verified 13 public routes, including the dashboard entry page, contact page, feeds, sitemap, and robots file, return HTTP 200. Browser verification showed live database content, an enabled dashboard sign-in button, and a successful API-to-identity redirect to the HTTPS email/password login page.
- Backend, PostgreSQL, and existing host services remain healthy. No existing containers were restarted. Owner authentication and authenticator enrollment require the owner's next sign-in; they were not completed on the owner's behalf. Google/GitHub provider credentials and transactional mail are still pending, and public registration remains disabled.
- Production has not been replaced. `DEPLOYMENTS_ENABLED=false` remains until the remaining staging acceptance and release steps are complete.

## Optional authenticator and verified registration — 10 October 2026

- Owner and visitor authenticators are now opt-in. Confirmed factors remain enforced; unfinished bootstrap enrollment no longer blocks owner login. Account settings support explicit enrollment, cancellation, and password-plus-code disabling, with revocation of other sessions and OAuth grants. The live owner has no confirmed factor and can sign in with email/password.
- Final backend race/integration checks passed, along with Go vet, frontend ESLint, 18 frontend unit tests, the Next.js production build, and 13 browser tests (three duplicate mobile ceremonies skipped). Browser coverage includes optional enrollment/cancellation. The integration test uses fresh TOTP time windows so slow race-detector password checks cannot expire its codes.
- Linode runs backend release `99e2023be1127766a70adaaa766880e071881ef7`. The staging frontend alias now points to `https://portfolio-idz48ua0f-chloe-pratas-projects.vercel.app`. Both previous images and frontend deployments remain available.
- Added six staging-only mail DNS records. The root Amazon SES MX is unchanged. Authoritative PTR now resolves `104.105.15.153` to `mail.staging.chloepratas.com`; its A record resolves back to the same address. The DKIM public key matches the private signing key.
- Docker Mailserver 16.0.1 runs separately with persistent mail/configuration volumes and a read-only mount of its Caddy certificate directory. Caddy was gracefully reloaded; existing containers were not restarted. The mail change detector is running and monitors both mounted certificate files. Actual certificate renewal remains a future operational check.
- Authenticated STARTTLS submission verifies the certificate hostname. Unauthenticated relays are rejected with 554 on both 25 and 587. Gmail accepted the user-approved test email over TLS; the user confirmed receipt in **Spam**. Inbox placement is not yet established.
- `REGISTRATION_ENABLED=true` is live. A temporary account using our staging mail mailbox completed actual Go-queued verification delivery, verification, password-only login, recovery delivery, password reset, old-password rejection, and new-password login. Unverified login was rejected. Both mail jobs succeeded on their first attempt; the test account was removed and the mail queue is empty. No real owner password or factor was changed.
- Mail-aware backup and off-host checksums passed. A disposable PostgreSQL restore recovered all 25 published documents and the owner. Mail queues/mailboxes/logs restored in isolated Linux storage; DKIM and TLS keys/configuration were also restored and validated. The private off-host backup is under `.backups/linode-staging/pre-optional-auth-mail/`.
- With explicit user approval, added inbound IPv4 TCP 25 to UFW and the separate Linode Cloud Firewall `warden` (17086384). Verified its sole attachment is this server’s interface 396752 on Linode 97499122. The existing SSH/ICMP/HTTP/HTTPS rules and inbound Drop/outbound Accept policies are unchanged. External SMTP now accepts the configured bounce recipient over trusted TLS and rejects unrelated relays. The SPF check also correctly rejected a probe that incorrectly claimed the server’s own hostname. SMTP submission stays private on the Docker network.
- Google/GitHub credentials, physical-device OAuth/passkeys, scheduled off-host backups/monitoring, and production replacement remain separate launch work. Continuous deployment remains gated as previously documented.

- Final public checks returned HTTP 200 for all 13 staging routes/feeds; the browser shows the open registration form. Vaultwarden (200), Seerr (307), and Jellyfin (302) retain their original responses, and all staging containers are healthy.
