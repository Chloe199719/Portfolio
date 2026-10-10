# Native system-browser example

Create a SwiftUI iOS 17+ app in Xcode and replace its generated App file with `ChloeIDExample.swift`. Add the entries from `Info.plist` to the target configuration. Register a **public** client in the dashboard, set its exact callback to `dev.chloe.mobile:/callback`, and permit `openid profile email offline_access`. Paste its client ID into `ChloeClientID`; use the HTTPS staging issuer for `ChloeIssuer`. No client secret belongs in this app.

The example uses `ASWebAuthenticationSession`, browser-bound state, S256 PKCE, authenticated UserInfo, and device-only Keychain storage for rotating refresh tokens. It never treats an unverified JWT payload as identity. The source also builds on macOS for a desktop demonstration.

For production apps, prefer an app-claimed HTTPS universal link and configure the corresponding associated-domain file. The custom scheme here is explicitly registered to keep this minimal example runnable without owning another domain. A device/simulator sign-in must be exercised against your HTTPS staging issuer before shipping.
