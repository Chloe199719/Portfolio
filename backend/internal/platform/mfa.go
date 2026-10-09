package platform

import (
	"bytes"
	"context"
	"encoding/base64"
	"html/template"
	"image/png"
	"net/http"
	"net/url"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func validCurrentTOTP(code, secret string) bool {
	// Accept only the current time step, so a used code cannot become valid
	// again under the next step's replay counter.
	valid, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{Period: 30, Skew: 0, Digits: 6, Algorithm: otp.AlgorithmSHA1})
	return err == nil && valid
}
func (s *Server) checkTOTP(ctx context.Context, uid, secret, code string) bool {
	if secret == "" || !validCurrentTOTP(code, secret) {
		return false
	}
	step := time.Now().Unix() / 30
	res, e := s.DB.Exec(ctx, "UPDATE accounts SET totp_last=$2,totp_confirmed=true WHERE id=$1 AND totp_last<$2 AND totp_secret=$3", uid, step, secret)
	return e == nil && res.RowsAffected() == 1
}

const mfaForm = `<p class="auth-step">Step 2 of 2</p>{{if .Setup}}<p>Scan this QR code with your authenticator app, then enter its six-digit code to finish setting up your account.</p><img class="enrollment-qr" src="{{.QR}}" width="272" height="272" alt="QR code to set up your authenticator"><details class="manual-enrollment"><summary>Can’t scan the QR code?</summary><p>Add a time-based account named <strong>{{.Email}}</strong> with this setup key:</p><code>{{.Secret}}</code></details>{{else}}<p class="muted">Enter the six-digit code from your authenticator to finish signing in as {{.Email}}.</p>{{end}}<form method="post">` + hiddenForm + `<label>Authenticator code<input class="verification-code" name="otp" inputmode="numeric" autocomplete="one-time-code" pattern="[0-9]{6}" minlength="6" maxlength="6" required {{if not .Setup}}autofocus{{end}} aria-describedby="code-help"></label><p id="code-help" class="muted">Use the current code shown in your app.</p><button>{{if .Setup}}Finish setup{{else}}Verify &amp; sign in{{end}}</button></form><p><a href="/login?reauth=true&amp;flow={{.Flow}}&amp;continue={{.Continue}}">Use a different account</a></p>`

func (s *Server) stepUp(w http.ResponseWriter, r *http.Request) {
	u := s.identityUser(w, r)
	if u == nil {
		return
	}
	if !u.Owner || u.MFA {
		s.finishLogin(w, r, u)
		return
	}
	var secret string
	var confirmed bool
	if e := s.DB.QueryRow(r.Context(), "SELECT totp_secret,totp_confirmed FROM accounts WHERE id=$1", u.ID).Scan(&secret, &confirmed); e != nil {
		fail(w, e)
		return
	}
	if secret == "" {
		fail(w, problem{409, "Authenticator setup is unavailable. Ask the server administrator to complete owner enrollment."})
		return
	}
	data := map[string]any{"Email": u.Email, "Setup": !confirmed}
	if r.Method == "POST" {
		if !s.csrf(w, r) {
			return
		}
		if e := s.limit(r.Context(), "mfa:"+u.ID, 8, 15*time.Minute); e != nil {
			data["Error"] = e.Error()
		} else if s.checkTOTP(r.Context(), u.ID, secret, r.FormValue("otp")) {
			if e := s.newSession(w, r, u.ID, "identity", true); e != nil {
				fail(w, e)
				return
			}
			if _, e := s.DB.Exec(r.Context(), "DELETE FROM sessions WHERE hash=$1", u.SessionHash); e != nil {
				fail(w, e)
				return
			}
			s.audit(r.Context(), u.ID, "login.mfa", "")
			u.MFA = true
			s.finishLogin(w, r, u)
			return
		} else {
			data["Error"] = "That code did not match. Enter the current six-digit code from your app."
		}
	}
	title := "Two-step verification."
	if !confirmed {
		enrollment := url.URL{Scheme: "otpauth", Host: "totp", Path: "/Chloe ID:" + u.Email, RawQuery: url.Values{"secret": {secret}, "issuer": {"Chloe ID"}, "algorithm": {"SHA1"}, "digits": {"6"}, "period": {"30"}}.Encode()}
		key, e := otp.NewKeyFromURL(enrollment.String())
		if e != nil {
			fail(w, e)
			return
		}
		qr, e := key.Image(224, 224)
		if e != nil {
			fail(w, e)
			return
		}
		var encoded bytes.Buffer
		if e = png.Encode(&encoded, qr); e != nil {
			fail(w, e)
			return
		}
		// Only a locally generated QR image is marked safe for the data URL context.
		data["QR"] = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes()))
		data["Secret"] = secret
		title = "Secure your account."
	}
	s.page(w, r, title, mfaForm, data)
}
