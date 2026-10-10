package platform

import (
	"encoding/base64"
	"encoding/json"
	"github.com/go-webauthn/webauthn/webauthn"
	"io/fs"
	"net/http"
	"net/url"
	"time"
)

type passkeyUser struct {
	User
	Keys []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte                         { return []byte(u.ID) }
func (u *passkeyUser) WebAuthnName() string                       { return u.Email }
func (u *passkeyUser) WebAuthnDisplayName() string                { return u.Name }
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.Keys }
func (s *Server) loadPasskeyUser(r *http.Request, uid string) (*passkeyUser, error) {
	u := &passkeyUser{}
	e := s.DB.QueryRow(r.Context(), "SELECT id,name,email,verified,owner,totp_confirmed FROM accounts WHERE id=$1 AND verified", uid).Scan(&u.ID, &u.Name, &u.Email, &u.Verified, &u.Owner, &u.Authenticator)
	if e != nil {
		return nil, e
	}
	rows, e := s.DB.Query(r.Context(), "SELECT credential FROM passkeys WHERE account_id=$1", uid)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var c webauthn.Credential
		if e = rows.Scan(&c); e != nil {
			return nil, e
		}
		u.Keys = append(u.Keys, c)
	}
	return u, rows.Err()
}
func (s *Server) passkeyPage(w http.ResponseWriter, r *http.Request) {
	s.page(w, r, "Sign in with a passkey.", `<p>Use a passkey saved on your device or security key.</p><div id="passkeys" data-csrf="{{.CSRF}}" data-flow="{{.Flow}}" data-continue="{{.Continue}}"><button id="login-passkey" type="button">Continue with passkey</button><p id="passkey-status" role="status"></p></div><p><a href="/login">Use another method</a></p><script src="/passkeys.js" defer></script>`, nil)
}
func (s *Server) passkeyAction(w http.ResponseWriter, r *http.Request) {
	if !s.csrf(w, r) {
		return
	}
	if e := s.limit(r.Context(), "passkey:"+s.ip(r), 30, 15*time.Minute); e != nil {
		fail(w, e)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	action := r.PathValue("action")
	ctx := r.Context()
	if action == "register-begin" || action == "login-begin" {
		var session *webauthn.SessionData
		var options any
		var e error
		uid := ""
		if action == "register-begin" {
			u := s.identityUser(w, r)
			if u == nil {
				return
			}
			if !u.authenticated() {
				fail(w, problem{403, "Verify your additional factor first."})
				return
			}
			uid = u.ID
			pu, err := s.loadPasskeyUser(r, uid)
			if err != nil {
				fail(w, err)
				return
			}
			options, session, e = s.WebAuthn.BeginRegistration(pu)
		} else {
			options, session, e = s.WebAuthn.BeginDiscoverableLogin()
		}
		if e != nil {
			fail(w, e)
			return
		}
		t, e := s.newToken(ctx, "passkey-"+action, uid, session, 5*time.Minute)
		if e != nil {
			fail(w, e)
			return
		}
		s.cookie(w, "chloe_passkey", t, 300)
		send(w, 200, map[string]any{"options": options})
		return
	}
	if action != "register-finish" && action != "login-finish" {
		fail(w, bad("Unknown passkey action."))
		return
	}
	purpose := "passkey-login-begin"
	if action == "register-finish" {
		purpose = "passkey-register-begin"
	}
	uid, data, e := s.consume(ctx, purpose, s.cookieValue(r, "chloe_passkey"))
	if e != nil {
		fail(w, e)
		return
	}
	s.cookie(w, "chloe_passkey", "", -1)
	raw, _ := json.Marshal(data)
	var session webauthn.SessionData
	if e = json.Unmarshal(raw, &session); e != nil {
		fail(w, e)
		return
	}
	if action == "register-finish" {
		u := s.identityUser(w, r)
		if u == nil {
			return
		}
		if u.ID != uid || !u.authenticated() {
			fail(w, problem{403, "Account changed during passkey registration."})
			return
		}
		pu, e := s.loadPasskeyUser(r, uid)
		if e != nil {
			fail(w, e)
			return
		}
		credential, e := s.WebAuthn.FinishRegistration(pu, session, r)
		if e != nil {
			fail(w, bad("Passkey registration could not be verified."))
			return
		}
		name := r.URL.Query().Get("name")
		if len(name) < 1 || len(name) > 80 {
			name = "My device"
		}
		raw, _ := json.Marshal(credential)
		if _, e = s.DB.Exec(ctx, "INSERT INTO passkeys(id,account_id,credential,name) VALUES($1,$2,$3,$4)", base64.RawURLEncoding.EncodeToString(credential.ID), uid, raw, name); e != nil {
			fail(w, e)
			return
		}
		s.audit(ctx, uid, "passkey.create", "")
		send(w, 200, map[string]any{"ok": true})
		return
	}
	handler := func(rawID, userHandle []byte) (webauthn.User, error) { return s.loadPasskeyUser(r, string(userHandle)) }
	user, credential, e := s.WebAuthn.FinishPasskeyLogin(handler, session, r)
	if e != nil || credential.Authenticator.CloneWarning {
		fail(w, problem{401, "Passkey sign-in could not be verified."})
		return
	}
	u := user.(*passkeyUser)
	raw, _ = json.Marshal(credential)
	if _, e = s.DB.Exec(ctx, "UPDATE passkeys SET credential=$2 WHERE id=$1 AND account_id=$3", base64.RawURLEncoding.EncodeToString(credential.ID), raw, u.ID); e != nil {
		fail(w, e)
		return
	}
	if e = s.newSession(w, r, u.ID, "identity", false); e != nil {
		fail(w, e)
		return
	}
	next := "/login"
	if u.Authenticator {
		next = "/step-up"
	}
	next += "?flow=" + url.QueryEscape(r.URL.Query().Get("flow")) + "&continue=" + url.QueryEscape(r.URL.Query().Get("continue"))
	send(w, 200, map[string]any{"ok": true, "next": next})
}
func (s *Server) passkeyScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript")
	b, e := fs.ReadFile(s.Files, "web/passkeys.js")
	if e != nil {
		http.NotFound(w, r)
		return
	}
	_, _ = w.Write(b)
}
