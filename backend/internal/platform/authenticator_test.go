package platform

import (
	"context"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestOptionalAuthenticator(t *testing.T) {
	for _, role := range []string{"owner", "visitor"} {
		t.Run(role, func(t *testing.T) {
			h := setup(t)
			ctx := context.Background()
			uid := h.owner
			if role == "visitor" {
				uid = h.visitor
			}
			pw, _ := passwordHash("a sufficiently long password")
			_, e := h.s.DB.Exec(ctx, "UPDATE accounts SET password_hash=$2,totp_confirmed=false,totp_secret='' WHERE id=$1", uid, pw)
			if e != nil {
				t.Fatal(e)
			}
			res := h.form(t, "/login", url.Values{"email": {role + "@example.test"}, "password": {"a sufficiently long password"}})
			if res.StatusCode != 303 || strings.Contains(res.Header.Get("Location"), "step-up") {
				t.Fatal("optional MFA blocked password-only login")
			}
			res.Body.Close()
			h.loginSession(t, uid, "api", false)
			want := 200
			if role == "visitor" {
				want = 403
			}
			status(t, h.call(t, "GET", "/v1/admin/content", nil), want)
			status(t, h.form(t, "/account/authenticator", url.Values{"action": {"start"}, "current": {"wrong"}}), 200)
			var secret string
			_ = h.s.DB.QueryRow(ctx, "SELECT totp_secret FROM accounts WHERE id=$1", uid).Scan(&secret)
			if secret != "" {
				t.Fatal("wrong password started enrollment")
			}
			status(t, h.form(t, "/account/authenticator", url.Values{"action": {"start"}, "current": {"a sufficiently long password"}}), 303)
			res = h.call(t, "GET", "/account/authenticator", nil)
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if !strings.Contains(string(body), "data:image/png;base64,") {
				t.Fatal("enrollment QR missing")
			}
			_ = h.s.DB.QueryRow(ctx, "SELECT totp_secret FROM accounts WHERE id=$1", uid).Scan(&secret)
			status(t, h.form(t, "/account/authenticator", url.Values{"action": {"enable"}, "otp": {"invalid"}}), 200)
			otp, _ := totp.GenerateCode(secret, time.Now())
			status(t, h.form(t, "/account/authenticator", url.Values{"action": {"enable"}, "otp": {otp}}), 303)
			status(t, h.call(t, "GET", "/v1/account", nil), 401) // pre-enrollment session revoked
			res = h.call(t, "GET", "/account/authenticator", nil)
			body, _ = io.ReadAll(res.Body)
			res.Body.Close()
			if strings.Contains(string(body), secret) || !strings.Contains(string(body), "Your authenticator is enabled") {
				t.Fatal("enabled factor leaked its secret")
			}
			status(t, h.form(t, "/account/authenticator", url.Values{"action": {"disable"}, "current": {"a sufficiently long password"}, "otp": {otp}}), 200) // replay denied
			var enabled bool
			_ = h.s.DB.QueryRow(ctx, "SELECT totp_confirmed FROM accounts WHERE id=$1", uid).Scan(&enabled)
			if !enabled {
				t.Fatal("replayed code disabled factor")
			}
			h.loginSession(t, uid, "api", false)
			status(t, h.call(t, "GET", "/v1/account", nil), 403)
			h.loginSession(t, uid, "identity", false)
			status(t, h.call(t, "GET", "/account", nil), 303)
			status(t, h.call(t, "GET", "/account/authenticator", nil), 303)
			h.loginSession(t, uid, "identity", true)
			// Advance the stored replay counter backwards in this isolated fixture to
			// exercise a new-code disable without a wall-clock-dependent 30s wait.
			_, _ = h.s.DB.Exec(ctx, "UPDATE accounts SET totp_last=0 WHERE id=$1", uid)
			otp, _ = totp.GenerateCode(secret, time.Now())
			status(t, h.form(t, "/account/authenticator", url.Values{"action": {"disable"}, "current": {"wrong"}, "otp": {otp}}), 200)
			status(t, h.form(t, "/account/authenticator", url.Values{"action": {"disable"}, "current": {"a sufficiently long password"}, "otp": {otp}}), 303)
			_ = h.s.DB.QueryRow(ctx, "SELECT totp_confirmed,totp_secret FROM accounts WHERE id=$1", uid).Scan(&enabled, &secret)
			if enabled || secret != "" {
				t.Fatal("disable did not erase the factor")
			}
			h.loginSession(t, uid, "api", false)
			status(t, h.call(t, "GET", "/v1/admin/content", nil), want)
			status(t, h.form(t, "/account/authenticator", url.Values{"action": {"start"}, "current": {"a sufficiently long password"}}), 303)
			_, _ = h.s.DB.Exec(ctx, "UPDATE accounts SET totp_pending_at=now()-interval '11 minutes' WHERE id=$1", uid)
			_ = h.s.DB.QueryRow(ctx, "SELECT totp_secret FROM accounts WHERE id=$1", uid).Scan(&secret)
			otp, _ = totp.GenerateCode(secret, time.Now())
			status(t, h.form(t, "/account/authenticator", url.Values{"action": {"enable"}, "otp": {otp}}), 200)
			_ = h.s.DB.QueryRow(ctx, "SELECT totp_confirmed FROM accounts WHERE id=$1", uid).Scan(&enabled)
			if enabled {
				t.Fatal("expired enrollment enabled factor")
			}
		})
	}
}
