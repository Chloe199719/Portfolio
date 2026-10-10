import SwiftUI
import AuthenticationServices
import CryptoKit
import Security
#if canImport(UIKit)
import UIKit
#else
import AppKit
#endif

// Register a PUBLIC client with the exact callback dev.chloe.mobile:/callback.
// Supply your HTTPS issuer and client ID in the example's Info.plist.
@main struct ChloeIDExample: App {
    @StateObject private var identity = IdentityModel()
    var body: some Scene {
        WindowGroup {
            VStack(alignment: .leading, spacing: 24) {
                Text("One identity.\nAnother app.").font(.largeTitle.bold())
                Text(identity.status).accessibilityIdentifier("status")
                Button("Sign in with Chloe ID") { identity.signIn() }.disabled(identity.busy)
                Button("Refresh session") { Task { await identity.refresh() } }.disabled(identity.busy)
                Button("Forget this device") { identity.forget() }
                Text("Sign-in opens your system browser. Your password stays with Chloe ID.").font(.footnote)
            }.padding(32).frame(minWidth: 280).tint(.green)
        }
    }
}
@MainActor final class IdentityModel: NSObject, ObservableObject, ASWebAuthenticationPresentationContextProviding {
    @Published var status = "Not signed in"
    @Published var busy = false
    private var browser: ASWebAuthenticationSession?
    private let callback = "dev.chloe.mobile:/callback"
    private var issuer: String { (Bundle.main.object(forInfoDictionaryKey: "ChloeIssuer") as? String ?? "").trimmingCharacters(in: CharacterSet(charactersIn: "/")) }
    private var clientID: String { Bundle.main.object(forInfoDictionaryKey: "ChloeClientID") as? String ?? "" }
    func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
        #if canImport(UIKit)
        return UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }.flatMap(\.windows).first(where: \.isKeyWindow) ?? ASPresentationAnchor()
        #else
        return NSApplication.shared.windows.first ?? ASPresentationAnchor()
        #endif
    }
    private func random() throws -> String {
        var bytes = [UInt8](repeating: 0, count: 32)
        guard SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes) == errSecSuccess else { throw IdentityError("Secure randomness unavailable") }
        return Data(bytes).base64URL
    }
    func signIn() {
        guard issuer.hasPrefix("https://"), !clientID.isEmpty else { status = "Set ChloeIssuer and ChloeClientID in Info.plist."; return }
        do {
            let verifier = try random(), state = try random(), nonce = try random()
            let challenge = Data(SHA256.hash(data: Data(verifier.utf8))).base64URL
            var url = URLComponents(string: issuer + "/oauth/authorize")!
            url.queryItems = ["client_id": clientID, "redirect_uri": callback, "response_type": "code", "scope": "openid profile email offline_access", "state": state, "nonce": nonce, "code_challenge": challenge, "code_challenge_method": "S256"].map { URLQueryItem(name: $0.key, value: $0.value) }
            busy = true
            browser = ASWebAuthenticationSession(url: url.url!, callbackURLScheme: "dev.chloe.mobile") { [weak self] result, error in
                Task { @MainActor in
                    guard let self else { return }
                    guard error == nil, let result else { self.busy = false; self.status = "Sign-in cancelled. You can try again."; return }
                    let parameters = URLComponents(url: result, resolvingAgainstBaseURL: false)?.queryItems ?? []
                    let value: (String) -> String? = { key in parameters.first(where: { $0.name == key })?.value }
                    guard result.scheme == "dev.chloe.mobile", result.path == "/callback", value("state") == state, let code = value("code"), value("error") == nil else { self.busy = false; self.status = "Sign-in response did not match this request."; return }
                    await self.exchange(["grant_type": "authorization_code", "client_id": self.clientID, "code": code, "redirect_uri": self.callback, "code_verifier": verifier])
                }
            }
            browser?.presentationContextProvider = self
            browser?.prefersEphemeralWebBrowserSession = false // Allow SSO with other projects.
            if browser?.start() != true { busy = false; status = "The browser could not open sign-in." }
        } catch { status = error.localizedDescription }
    }
    func refresh() async {
        guard let token = Keychain.read() else { status = "Sign in first."; return }
        busy = true
        await exchange(["grant_type": "refresh_token", "client_id": clientID, "refresh_token": token])
    }
    private func exchange(_ values: [String: String]) async {
        defer { busy = false }
        do {
            guard let address = URL(string: issuer + "/oauth/token") else { throw IdentityError("Invalid issuer") }
            var request = URLRequest(url: address)
            request.httpMethod = "POST"
            request.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")
            var encoded = URLComponents(); encoded.queryItems = values.map { URLQueryItem(name: $0.key, value: $0.value) }
            request.httpBody = Data((encoded.percentEncodedQuery ?? "").replacingOccurrences(of: "+", with: "%2B").utf8)
            let (data, response) = try await URLSession.shared.data(for: request)
            guard (response as? HTTPURLResponse)?.statusCode == 200 else { throw IdentityError("Session expired or exchange rejected. Sign in again.") }
            let tokens = try JSONDecoder().decode(TokenResponse.self, from: data)
            if let refresh = tokens.refresh_token { try Keychain.save(refresh) }
            var profile = URLRequest(url: URL(string: issuer + "/oauth/userinfo")!)
            profile.setValue("Bearer " + tokens.access_token, forHTTPHeaderField: "Authorization")
            let (profileData, profileResponse) = try await URLSession.shared.data(for: profile)
            guard (profileResponse as? HTTPURLResponse)?.statusCode == 200 else { throw IdentityError("Could not load your account") }
            let user = try JSONDecoder().decode(Profile.self, from: profileData)
            status = "Signed in as \(user.name ?? user.sub)"
        } catch { status = error.localizedDescription }
    }
    func forget() { Keychain.remove(); status = "This device forgot its session. Revoke connected app access in your Chloe ID account." }
}
private struct TokenResponse: Decodable { let access_token: String; let refresh_token: String? }
private struct Profile: Decodable { let sub: String; let name: String? }
private struct IdentityError: LocalizedError { let message: String; init(_ message: String) { self.message = message }; var errorDescription: String? { message } }
private extension Data { var base64URL: String { base64EncodedString().replacingOccurrences(of: "+", with: "-").replacingOccurrences(of: "/", with: "_").replacingOccurrences(of: "=", with: "") } }
private enum Keychain {
    static let query: [String: Any] = [kSecClass as String: kSecClassGenericPassword, kSecAttrService as String: "dev.chloe.mobile", kSecAttrAccount as String: "refresh-token"]
    static func save(_ value: String) throws {
        remove(); var item = query; item[kSecValueData as String] = Data(value.utf8); item[kSecAttrAccessible as String] = kSecAttrAccessibleWhenUnlockedThisDeviceOnly
        guard SecItemAdd(item as CFDictionary, nil) == errSecSuccess else { throw IdentityError("Could not securely save the session") }
    }
    static func read() -> String? { var item = query; item[kSecReturnData as String] = true; var result: CFTypeRef?; guard SecItemCopyMatching(item as CFDictionary, &result) == errSecSuccess, let data = result as? Data else { return nil }; return String(data: data, encoding: .utf8) }
    static func remove() { SecItemDelete(query as CFDictionary) }
}
