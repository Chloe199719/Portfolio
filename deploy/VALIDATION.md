# Local validation — 8 October 2026

Implementation is validated locally. It has **not** been deployed to Vercel or Linode, and real-domain social login/mail delivery have not been exercised.

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
- Run OAuth/passkeys on a physical mobile device and complete a restore drill on the target Linode.
- Configure the host monitor and encrypted off-host backups, then preserve the prior deployment before production replacement.

See [the deployment runbook](README.md) for commands and configuration. The local tests do not certify OIDC conformance or substitute for a security review of an independently operated identity provider.
