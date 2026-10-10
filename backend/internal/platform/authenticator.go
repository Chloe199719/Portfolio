package platform

import (
	"net/http"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const authenticatorBody = `<p>Two-step verification is optional. When enabled, your authenticator code is required after signing in.</p>
{{if .Enabled}}<p class="notice">Your authenticator is enabled.</p><form method="post">` + hiddenForm + `<input type="hidden" name="action" value="disable">{{if .HasPassword}}<label>Current password<input name="current" type="password" required autocomplete="current-password"></label>{{end}}<label>Current authenticator code<input name="otp" required inputmode="numeric" autocomplete="one-time-code" pattern="[0-9]{6}"></label><button>Disable authenticator</button></form>
{{else if .Pending}}<p>Scan this code with your authenticator, then enter a code to enable it. Setup expires after ten minutes.</p><img class="enrollment-qr" src="{{.QR}}" width="272" height="272" alt="QR code to set up your authenticator"><details class="manual-enrollment"><summary>Can’t scan the QR code?</summary><code>{{.Secret}}</code></details><form method="post">` + hiddenForm + `<input type="hidden" name="action" value="enable"><label>Authenticator code<input name="otp" required inputmode="numeric" autocomplete="one-time-code" pattern="[0-9]{6}"></label><button>Enable authenticator</button></form><form method="post">` + hiddenForm + `<button name="action" value="cancel">Cancel setup</button></form>
{{else}}<p>You currently sign in without an authenticator.</p><form method="post">` + hiddenForm + `<input type="hidden" name="action" value="start">{{if .HasPassword}}<label>Current password<input name="current" type="password" required autocomplete="current-password"></label>{{end}}<button>Set up authenticator</button></form>{{end}}<p><a href="/account">Back to account</a></p>`

func (s *Server) authenticator(w http.ResponseWriter, r *http.Request) {
	u := s.identityUser(w, r)
	if u == nil {
		return
	}
	if !u.authenticated() {
		http.Redirect(w, r, "/step-up", http.StatusSeeOther)
		return
	}
	data := map[string]any{}
	if r.Method == http.MethodPost {
		if !s.csrf(w, r) {
			return
		}
		if e := s.limit(r.Context(), "authenticator:"+u.ID, 12, 15*time.Minute); e != nil {
			fail(w, e)
			return
		}
		if e := s.updateAuthenticator(r, u); e != nil {
			data["Error"] = e.Error()
		} else {
			s.audit(r.Context(), u.ID, "authenticator."+r.FormValue("action"), "")
			http.Redirect(w, r, "/account/authenticator", http.StatusSeeOther)
			return
		}
	}
	var secret string
	var enabled, hasPassword bool
	var pending *time.Time
	if e := s.DB.QueryRow(r.Context(), "SELECT totp_secret,totp_confirmed,totp_pending_at,password_hash!='' FROM accounts WHERE id=$1", u.ID).Scan(&secret, &enabled, &pending, &hasPassword); e != nil {
		fail(w, e)
		return
	}
	data["Enabled"], data["HasPassword"] = enabled, hasPassword
	if !enabled && secret != "" && pending != nil && time.Since(*pending) < 10*time.Minute {
		data["Pending"] = true
		if e := enrollmentData(u.Email, secret, data); e != nil {
			fail(w, e)
			return
		}
	}
	s.page(w, r, "Your authenticator.", authenticatorBody, data)
}

func (s *Server) updateAuthenticator(r *http.Request, u *User) error {
	ctx := r.Context()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(718299)"); e != nil {
		return e
	}
	var secret, password string
	var enabled bool
	var last int64
	var pending *time.Time
	if e = tx.QueryRow(ctx, "SELECT totp_secret,totp_confirmed,totp_last,totp_pending_at,password_hash FROM accounts WHERE id=$1 FOR UPDATE", u.ID).Scan(&secret, &enabled, &last, &pending, &password); e != nil {
		return e
	}
	// Recheck against locked state so concurrent enrollment cannot be bypassed.
	if enabled && !u.MFA {
		return bad("Verify your authenticator first.")
	}
	action := r.FormValue("action")
	if action == "start" || action == "disable" {
		if password != "" {
			if bcrypt.CompareHashAndPassword([]byte(password), []byte(r.FormValue("current"))) != nil {
				return bad("Current password is incorrect.")
			}
		} else if time.Since(u.AuthTime) > 10*time.Minute {
			return bad("Sign out and sign in again before changing your authenticator.")
		}
	}
	switch action {
	case "start":
		if enabled {
			return bad("Your authenticator is already enabled.")
		}
		key, err := totp.Generate(totp.GenerateOpts{Issuer: "Chloe ID", AccountName: u.Email})
		if err != nil {
			return err
		}
		_, e = tx.Exec(ctx, "UPDATE accounts SET totp_secret=$2,totp_last=0,totp_pending_at=now() WHERE id=$1", u.ID, key.Secret())
	case "enable":
		if enabled || pending == nil || time.Since(*pending) >= 10*time.Minute {
			return bad("Setup expired. Start authenticator setup again.")
		}
		if !validCurrentTOTP(r.FormValue("otp"), secret) {
			return bad("That code did not match. Use the current code from your app.")
		}
		_, e = tx.Exec(ctx, "UPDATE accounts SET totp_confirmed=true,totp_pending_at=NULL,totp_last=$2 WHERE id=$1", u.ID, time.Now().Unix()/30)
	case "disable":
		if !enabled {
			return bad("Your authenticator is already disabled.")
		}
		if !validCurrentTOTP(r.FormValue("otp"), secret) || time.Now().Unix()/30 <= last {
			return bad("Use a new, unused code from your authenticator.")
		}
		_, e = tx.Exec(ctx, "UPDATE accounts SET totp_confirmed=false,totp_secret='',totp_last=0,totp_pending_at=NULL WHERE id=$1", u.ID)
	case "cancel":
		if enabled {
			return bad("An enabled authenticator cannot be cancelled.")
		}
		_, e = tx.Exec(ctx, "UPDATE accounts SET totp_secret='',totp_last=0,totp_pending_at=NULL WHERE id=$1", u.ID)
	default:
		return bad("Unknown authenticator action.")
	}
	if e != nil {
		return e
	}
	if action == "enable" || action == "disable" {
		// Revoke pre-change API sessions, login tickets and refresh grants atomically.
		if _, e = tx.Exec(ctx, "DELETE FROM sessions WHERE account_id=$1 AND hash!=$2", u.ID, u.SessionHash); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE sessions SET mfa=$2 WHERE hash=$1", u.SessionHash, action == "enable"); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "DELETE FROM one_time_tokens WHERE account_id=$1", u.ID); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE oauth_sessions SET active=false WHERE account_id=$1", u.ID); e != nil {
			return e
		}
		if s.C.SMTPUser != "" {
			if _, e = tx.Exec(ctx, "INSERT INTO email_jobs(id,recipient,subject,body) VALUES($1,$2,$3,$4)", id(), u.Email, "Chloe ID authenticator updated", "Your authenticator setting changed: "+action+". Review your account at "+s.C.Issuer+"/account."); e != nil {
				return e
			}
		}
	}
	return tx.Commit(ctx)
}
