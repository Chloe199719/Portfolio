# A second web application

Register a **confidential web client** in the dashboard with callback `http://localhost:4000/callback` and scopes `openid`, `profile`, `email`. Local HTTP callbacks are accepted only by a development issuer. Run this independent server from the repository root:

```sh
ISSUER_URL=http://localhost:8080 CLIENT_ID=registered-id CLIENT_SECRET=registered-secret go run examples/web/main.go
```

Open `http://localhost:4000`. A previous Chloe ID login is reused by the system browser; consent is still explicit. This example uses a server-held PKCE verifier, validates the browser-bound state, authenticates confidential token requests, retrieves identity from the authenticated UserInfo endpoint, and creates its own HTTP-only session. It does not trust decoded JWT payloads. No tokens enter browser storage.

This is a local demonstration: in-memory example sessions disappear on restart. For production use HTTPS, Secure cookies, persistent application sessions, session cleanup, and your own logout route. Never place this confidential client's secret in a browser bundle or mobile app.
