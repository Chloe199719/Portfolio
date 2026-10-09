package platform

import (
	"context"
	"encoding/json"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/token/jwt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

func (s *Server) initIdentity(ctx context.Context) error {
	var e error
	s.Keys, e = loadKeys(s.C.DataDir)
	if e != nil {
		return e
	}
	c := &fosite.Config{GlobalSecret: s.Keys.Secret, IDTokenIssuer: s.C.Issuer, AccessTokenLifespan: 5 * time.Minute, RefreshTokenLifespan: 30 * 24 * time.Hour, AuthorizeCodeLifespan: time.Minute, IDTokenLifespan: 5 * time.Minute, EnforcePKCE: true, EnablePKCEPlainChallengeMethod: false, RefreshTokenScopes: []string{"offline_access"}, MinParameterEntropy: 16}
	getter := func(context.Context) (interface{}, error) { return s.Keys.Private(), nil }
	strategy := &compose.CommonStrategy{CoreStrategy: compose.NewOAuth2HMACStrategy(c), OpenIDConnectTokenStrategy: compose.NewOpenIDConnectStrategy(getter, c), Signer: &jwt.DefaultSigner{GetPrivateKey: getter}}
	s.OAuth = compose.Compose(c, &OAuthStore{S: s}, strategy, compose.OAuth2AuthorizeExplicitFactory, compose.OAuth2RefreshTokenGrantFactory, compose.OpenIDConnectExplicitFactory, compose.OpenIDConnectRefreshFactory, compose.OAuth2TokenIntrospectionFactory, compose.OAuth2TokenRevocationFactory, compose.OAuth2PKCEFactory)
	u, _ := url.Parse(s.C.Issuer)
	s.WebAuthn, e = webauthn.New(&webauthn.Config{RPDisplayName: "Chloe ID", RPID: u.Hostname(), RPOrigins: []string{s.C.Issuer}, AuthenticatorSelection: protocol.AuthenticatorSelection{UserVerification: protocol.VerificationRequired, ResidentKey: protocol.ResidentKeyRequirementRequired}})
	return e
}
func (s *Server) discovery(w http.ResponseWriter, r *http.Request) {
	i := s.C.Issuer
	send(w, 200, map[string]any{"issuer": i, "authorization_endpoint": i + "/oauth/authorize", "token_endpoint": i + "/oauth/token", "userinfo_endpoint": i + "/oauth/userinfo", "jwks_uri": i + "/.well-known/jwks.json", "revocation_endpoint": i + "/oauth/revoke", "introspection_endpoint": i + "/oauth/introspect", "end_session_endpoint": i + "/logout", "response_types_supported": []string{"code"}, "grant_types_supported": []string{"authorization_code", "refresh_token"}, "subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"}, "code_challenge_methods_supported": []string{"S256"}, "token_endpoint_auth_methods_supported": []string{"client_secret_basic", "none"}, "scopes_supported": []string{"openid", "profile", "email", "offline_access"}, "claims_supported": []string{"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "name", "email", "email_verified"}})
}
func (s *Server) authorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if e := s.limit(ctx, "authorize:"+s.ip(r), 100, time.Minute); e != nil {
		fail(w, e)
		return
	}
	if r.Method == "POST" && !s.csrf(w, r) {
		return
	}
	ar, e := s.OAuth.NewAuthorizeRequest(ctx, r)
	if e != nil {
		s.OAuth.WriteAuthorizeError(ctx, w, ar, e)
		return
	}
	redirect := ar.GetRequestForm().Get("redirect_uri")
	if !slices.Contains(ar.GetClient().GetRedirectURIs(), redirect) {
		fail(w, bad("Use an exactly registered redirect URI."))
		return
	}
	if ar.GetRequestForm().Get("code_challenge_method") != "S256" {
		s.OAuth.WriteAuthorizeError(ctx, w, ar, fosite.ErrInvalidRequest.WithHint("S256 PKCE is required."))
		return
	}
	u := s.user(r, "identity")
	if r.Method == "GET" {
		prompt := strings.Fields(ar.GetRequestForm().Get("prompt"))
		force := slices.Contains(prompt, "login")
		if value := ar.GetRequestForm().Get("max_age"); value != "" {
			age, err := strconv.ParseInt(value, 10, 64)
			if err != nil || age < 0 {
				s.OAuth.WriteAuthorizeError(ctx, w, ar, fosite.ErrInvalidRequest.WithHint("Invalid max_age."))
				return
			}
			force = force || u != nil && time.Now().Unix()-u.AuthTime.Unix() > age
		}
		if force {
			if slices.Contains(prompt, "none") {
				s.OAuth.WriteAuthorizeError(ctx, w, ar, fosite.ErrLoginRequired)
				return
			}
			continuation := r.URL.Query()
			continuation.Del("max_age")
			remaining := []string{}
			for _, p := range prompt {
				if p != "login" {
					remaining = append(remaining, p)
				}
			}
			continuation.Set("prompt", strings.Join(remaining, " "))
			http.Redirect(w, r, "/login?reauth=true&continue="+url.QueryEscape("/oauth/authorize?"+continuation.Encode()), 303)
			return
		}
	}
	if u == nil || !u.Verified || u.Owner && !u.MFA {
		if ar.GetRequestForm().Get("prompt") == "none" {
			s.OAuth.WriteAuthorizeError(ctx, w, ar, fosite.ErrLoginRequired)
			return
		}
		http.Redirect(w, r, "/login?continue="+url.QueryEscape("/oauth/authorize?"+r.URL.RawQuery), 303)
		return
	}
	if r.Method != "POST" {
		if ar.GetRequestForm().Get("prompt") == "none" {
			s.OAuth.WriteAuthorizeError(ctx, w, ar, fosite.ErrConsentRequired)
			return
		}
		var name string
		_ = s.DB.QueryRow(ctx, "SELECT name FROM oauth_clients WHERE id=$1", ar.GetClient().GetID()).Scan(&name)
		s.page(w, r, "Connect "+name+"?", `<p>This application is requesting access to:</p><ul>{{range .Scopes}}<li>{{.}}</li>{{end}}</ul><p class="muted">Signed in as {{.Email}}. You can revoke access from your account.</p><form method="post">`+hiddenForm+`<div class="actions"><button name="decision" value="allow">Allow access</button><button name="decision" value="deny">Cancel</button></div></form>`, map[string]any{"Scopes": ar.GetRequestedScopes(), "Email": u.Email, "FormTarget": consentTarget(redirect)})
		return
	}
	if r.FormValue("decision") != "allow" {
		s.OAuth.WriteAuthorizeError(ctx, w, ar, fosite.ErrAccessDenied)
		return
	}
	for _, scope := range ar.GetRequestedScopes() {
		ar.GrantScope(scope)
	}
	session := openid.NewDefaultSession()
	session.Subject = u.ID
	session.Username = u.Name
	claims := session.IDTokenClaims()
	claims.Subject = u.ID
	claims.AuthTime = u.AuthTime.UTC()
	claims.Extra = map[string]any{}
	if ar.GetGrantedScopes().Has("profile") {
		claims.Extra["name"] = u.Name
	}
	if ar.GetGrantedScopes().Has("email") {
		claims.Extra["email"] = u.Email
		claims.Extra["email_verified"] = true
	}
	session.IDTokenHeaders().Add("kid", s.Keys.CurrentID())
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		fail(w, e)
		return
	}
	defer tx.Rollback(ctx)
	ctx = context.WithValue(ctx, oauthTxKey{}, tx)
	response, e := s.OAuth.NewAuthorizeResponse(ctx, ar, session)
	if e != nil {
		s.OAuth.WriteAuthorizeError(ctx, w, ar, e)
		return
	}
	scopes, _ := json.Marshal(ar.GetGrantedScopes())
	_, e = tx.Exec(ctx, "INSERT INTO oauth_consents(account_id,client_id,scopes) VALUES($1,$2,$3) ON CONFLICT(account_id,client_id) DO UPDATE SET scopes=EXCLUDED.scopes", u.ID, ar.GetClient().GetID(), scopes)
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		fail(w, e)
		return
	}
	s.OAuth.WriteAuthorizeResponse(ctx, w, ar, response)
}
func (s *Server) oauthAtomic(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if e := s.limit(r.Context(), "oauth:"+s.ip(r), 120, time.Minute); e != nil {
			fail(w, e)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16384)
		tx, e := s.DB.Begin(r.Context())
		if e != nil {
			fail(w, e)
			return
		}
		defer tx.Rollback(r.Context())
		if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(718299)"); e != nil {
			fail(w, e)
			return
		}
		ctx := context.WithValue(r.Context(), oauthTxKey{}, tx)
		rec := httptest.NewRecorder()
		fn(rec, r.WithContext(ctx))
		if rec.Code >= 500 {
			returnError := tx.Rollback(ctx)
			_ = returnError
		} else if e = tx.Commit(ctx); e != nil {
			fail(w, e)
			return
		}
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Code)
		_, _ = w.Write(rec.Body.Bytes())
	}
}
func (s *Server) oauthToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ar, e := s.OAuth.NewAccessRequest(ctx, r, openid.NewDefaultSession())
	if e != nil {
		s.OAuth.WriteAccessError(ctx, w, ar, e)
		return
	}
	resp, e := s.OAuth.NewAccessResponse(ctx, ar)
	if e != nil {
		s.OAuth.WriteAccessError(ctx, w, ar, e)
		return
	}
	s.OAuth.WriteAccessResponse(ctx, w, ar, resp)
}
func (s *Server) oauthRevoke(w http.ResponseWriter, r *http.Request) {
	e := s.OAuth.NewRevocationRequest(r.Context(), r)
	s.OAuth.WriteRevocationResponse(r.Context(), w, e)
}
func (s *Server) oauthIntrospect(w http.ResponseWriter, r *http.Request) {
	ar, e := s.OAuth.NewIntrospectionRequest(r.Context(), r, openid.NewDefaultSession())
	if e != nil {
		s.OAuth.WriteIntrospectionError(r.Context(), w, e)
		return
	}
	s.OAuth.WriteIntrospectionResponse(r.Context(), w, ar)
}
func (s *Server) userInfo(w http.ResponseWriter, r *http.Request) {
	bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	_, req, e := s.OAuth.IntrospectToken(r.Context(), bearer, fosite.AccessToken, openid.NewDefaultSession(), "openid")
	if e != nil {
		fail(w, problem{401, "Invalid or expired access token."})
		return
	}
	var name, email string
	var verified bool
	e = s.DB.QueryRow(r.Context(), "SELECT name,email,verified FROM accounts WHERE id=$1", req.GetSession().GetSubject()).Scan(&name, &email, &verified)
	if e != nil {
		fail(w, problem{401, "Account is unavailable."})
		return
	}
	out := map[string]any{"sub": req.GetSession().GetSubject()}
	if req.GetGrantedScopes().Has("profile") {
		out["name"] = name
	}
	if req.GetGrantedScopes().Has("email") {
		out["email"] = email
		out["email_verified"] = verified
	}
	send(w, 200, out)
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !s.csrf(w, r) {
			return
		}
		if _, e := s.DB.Exec(r.Context(), "DELETE FROM sessions WHERE hash=$1", hash(s.cookieValue(r, "chloe_identity"))); e != nil {
			fail(w, e)
			return
		}
		s.cookie(w, "chloe_identity", "", -1)
		http.Redirect(w, r, s.C.SiteURL, 303)
		return
	}
	s.page(w, r, "Sign out?", `<p>This ends your Chloe ID browser session. Connected app access can be revoked from your account.</p><form method="post">`+hiddenForm+`<button>Sign out</button></form>`, nil)
}

// Browsers apply form-action to redirects too. Only the already validated
// registered application's origin (or native URI scheme) is permitted.
func consentTarget(redirect string) string {
	u, _ := url.Parse(redirect)
	if u.Host != "" && (u.Scheme == "https" || u.Scheme == "http") {
		return u.Scheme + "://" + u.Host
	}
	return u.Scheme + ":"
}
