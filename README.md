# Chloe’s personal home

A Next.js 16 / React 19 / TypeScript frontend for **Vercel**, backed by a separate **Go service and PostgreSQL on Linode**. The Go service owns content, account screens, sessions, OAuth/OIDC, uploads, moderation, and durable jobs. There is no Firebase, Sanity, or database connection in the frontend.

The public site includes project stories, notes and RSS, photography, About, Now, a memory game, a moderated guestbook, and a private contact inbox. The dark design uses warm charcoal, warm white, cherry red, Space Grotesk, and Manrope. Empty homepage collections stay hidden. Only preserved content and supplied images are published by default.

## Local development

Use Node 24 LTS, Go 1.26.8, and PostgreSQL 18. The module pins Go’s toolchain. The API contract is [backend/openapi.json](backend/openapi.json); run `npm run api:generate` after changing it.

```sh
npm ci
# Terminal 1: export your PostgreSQL connection and start Go.
cd backend
export DATABASE_URL=postgresql://portfolio:local-development-only@localhost:5432/portfolio
export SITE_URL=http://localhost:3000 FRONTEND_ORIGINS=http://localhost:3000
export API_URL=http://localhost:8080 ISSUER_URL=http://localhost:8080
export DATA_DIR=./data
go run ./cmd/server
```

In another terminal at the repository root:

```sh
NEXT_PUBLIC_API_URL=http://localhost:8080 \
NEXT_PUBLIC_AUTH_URL=http://localhost:8080 \
NEXT_PUBLIC_SITE_URL=http://localhost:3000 npm run dev
```

Use **localhost** consistently. Passkeys require a domain name; do not configure the issuer as an IP address. For production use HTTPS and the custom `api.`/`auth.` hostnames. The frontend fetches public content without caching, so publication updates pages, feeds, and the sitemap on the next request. A preview without an API URL uses only the committed public content export; a configured API failure is shown as an error.

Create the owner with the backend CLI. Put a long password in a private temporary file, then run:

```sh
go run ./cmd/server owner owner@example.com 'Chloe' /secure/path/password.txt
```

Remove the temporary password file. At first sign-in, scan the QR code with your authenticator and confirm its six-digit code to complete owner setup. Later sign-ins ask for that code only after the password step. Public registration cannot grant owner access. Owner APIs require both the owner account and completed additional-factor authentication.

## Publishing

Visit `/admin`, sign in with Chloe ID, and complete the separate authenticator step. All authorization is enforced again by Go.

- Drafts autosave after a short pause. Local browser recovery keeps unsent edits across reloads. A failed response never clears edits. Saves are serialized and merge only returned revision metadata into newer typing.
- **Save draft** and **Publish** create immutable revisions. Autosaves update the working copy. Restoring history creates another draft.
- Scheduling saves and pins a revision, displays Europe/Berlin time, and stores UTC. Later edits stay separate. PostgreSQL locks make publication atomic across restarts and concurrent workers.
- Media supports bulk upload, search, reuse, captions, alt text, and usage records. JPEG/PNG/WebP uploads are decoded, resized, and re-encoded without embedded metadata. Private media requires an owner session; all uploaded media bypasses Vercel image optimization and uses `private, no-store` responses so unpublishing takes effect immediately.
- Filter and paginate content, duplicate into drafts, or move entries to recoverable trash. Restores remain private. Published slugs remain fixed.
- Homepage settings order and hide modules; each entry controls its selection and display order. Publishing validates descriptions, alt text, safe URLs, and internal references.

## Identity

`auth.` serves independently of Vercel: local registration, email verification, login, recovery, Google/GitHub login, explicit linking, passkeys, account management, login sessions, app consent, and revocation. Social identities are keyed by provider subject. Matching email addresses never merge accounts.

The Fosite provider exposes discovery, JWKS, authorization, token, UserInfo, introspection, revocation, and an interactive logout endpoint. Authorization Code + S256 PKCE is mandatory for all clients. Tokens use five-minute lifetimes; refresh tokens rotate with replay detection. Signing keys and the opaque-token HMAC secret persist on disk. Old public keys remain available after rotation.

Owner-managed applications are in the dashboard. Register exact callbacks, allowed browser origins, and scopes. Public browser/native clients have no secret; confidential web clients use `client_secret_basic`. Wildcards, fragments, and unsafe redirects are rejected. OAuth protocol requests use PKCE/state and client authentication; website mutations use host-scoped HTTP-only cookies, exact origin checks, and CSRF tokens. Browser application tokens are never written to local storage.

Examples: [a second web application](examples/web/README.md) and [a Swift native application](examples/mobile/README.md). Their profile identity comes from authenticated UserInfo; neither example trusts an unverified JWT payload.

## Validation

```sh
npm run api:generate
npm run typecheck
npm run lint
npm test
npm run build
cd backend
go vet ./...
go test ./...
TEST_DATABASE_URL=postgresql://user:password@localhost/portfolio_test go test -race ./...
```

Integration tests create and destroy their own schemas and refuse database names without `_test`. They cover content privacy, revision conflicts, concurrent scheduling, CSRF, moderation, owner MFA, expired sessions, password recovery, PKCE, code replay, refresh replay, signing-key rotation, registration verification, account isolation, media privacy, and metadata removal.

Browser tests need the frontend and backend running against a separate `*_test` database. Create test-only fixtures (this command is not included in the production image):

```sh
cd backend
DATABASE_URL=postgresql://user:password@localhost/portfolio_browser_test \
  go run ./cmd/testfixture > /tmp/chloe-browser-fixture.json
cd ..
E2E_FIXTURE_FILE=/tmp/chloe-browser-fixture.json npx playwright test
```

Set `E2E_WEB_EXAMPLE_URL=http://localhost:4000` while the registered web example runs to include its consent/sign-in test. The browser suite covers desktop/mobile routes, images, metadata, normal keyboard/reduced-motion behavior, contact failures, save races, local draft recovery, private previews, publication, history, scheduling, and a virtual passkey. Test fixture credentials are only for disposable local databases.

Deployment, mail DNS, backups, restore checks, monitoring, and the production checklist are in [deploy/README.md](deploy/README.md). Live Google/GitHub login and real email delivery require your provider credentials and configured DNS; local protocol tests do not claim those external checks passed.
