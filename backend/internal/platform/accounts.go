package platform

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID          string    `json:"uid"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Owner       bool      `json:"owner"`
	MFA         bool      `json:"mfa"`
	SessionHash string    `json:"-"`
	Verified    bool      `json:"verified"`
	Credentials []byte    `json:"-"`
	AuthTime    time.Time `json:"-"`
}

func (s *Server) user(r *http.Request, kind string) *User {
	v := s.cookieValue(r, "chloe_"+kind)
	if v == "" {
		return nil
	}
	u := &User{}
	e := s.DB.QueryRow(r.Context(), `SELECT a.id,a.name,a.email,a.owner,a.verified,s.mfa,s.hash,s.created_at FROM sessions s JOIN accounts a ON a.id=s.account_id WHERE s.hash=$1 AND s.kind=$2 AND s.expires_at>now()`, hash(v), kind).Scan(&u.ID, &u.Name, &u.Email, &u.Owner, &u.Verified, &u.MFA, &u.SessionHash, &u.AuthTime)
	if e != nil {
		return nil
	}
	return u
}
func (s *Server) require(w http.ResponseWriter, r *http.Request, owner bool) *User {
	u := s.user(r, "api")
	if u == nil {
		fail(w, problem{401, "Sign in to continue."})
		return nil
	}
	if !u.Verified {
		fail(w, problem{403, "Verify your email first."})
		return nil
	}
	if owner && (!u.Owner || !u.MFA) {
		fail(w, problem{403, "Owner access requires an additional authentication factor."})
		return nil
	}
	return u
}
func (s *Server) newSession(w http.ResponseWriter, r *http.Request, uid, kind string, mfa bool) error {
	lifetime := 5 * 24 * time.Hour
	if kind == "identity" && !mfa {
		var owner bool
		if e := s.DB.QueryRow(r.Context(), "SELECT owner FROM accounts WHERE id=$1", uid).Scan(&owner); e != nil {
			return e
		}
		if owner {
			lifetime = 10 * time.Minute
		}
	}
	v := token()
	_, e := s.DB.Exec(r.Context(), "INSERT INTO sessions(hash,account_id,kind,mfa,expires_at,agent) VALUES($1,$2,$3,$4,$5,$6)", hash(v), uid, kind, mfa, time.Now().Add(lifetime), r.UserAgent())
	if e != nil {
		return e
	}
	s.cookie(w, "chloe_"+kind, v, int(lifetime.Seconds()))
	return nil
}
func (s *Server) newToken(ctx context.Context, purpose, uid string, payload any, lifetime time.Duration) (string, error) {
	v := token()
	raw, e := json.Marshal(payload)
	if e != nil {
		return "", e
	}
	_, e = s.DB.Exec(ctx, "INSERT INTO one_time_tokens(hash,purpose,account_id,payload,expires_at) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5)", hash(v), purpose, uid, raw, time.Now().Add(lifetime))
	return v, e
}
func (s *Server) consume(ctx context.Context, purpose, value string) (string, Document, error) {
	var uid string
	var data Document
	e := s.DB.QueryRow(ctx, "DELETE FROM one_time_tokens WHERE hash=$1 AND purpose=$2 AND expires_at>now() RETURNING coalesce(account_id::text,''),payload", hash(value), purpose).Scan(&uid, &data)
	if errors.Is(e, pgx.ErrNoRows) {
		e = bad("This link expired or has already been used.")
	}
	return uid, data, e
}
func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	if r.Method == "DELETE" {
		if _, e := s.DB.Exec(r.Context(), "DELETE FROM sessions WHERE hash=$1 AND kind='api'", hash(s.cookieValue(r, "chloe_api"))); e != nil {
			fail(w, e)
			return
		}
		s.cookie(w, "chloe_api", "", -1)
		send(w, 200, map[string]any{"ok": true})
		return
	}
	u := s.user(r, "api")
	if u != nil {
		u.Owner = u.Owner && u.MFA
	}
	send(w, 200, map[string]any{"user": u, "csrf": s.csrfToken(w, r), "configured": true, "authUrl": s.C.Issuer, "registration": s.C.Registration})
}
func (s *Server) sessionStart(w http.ResponseWriter, r *http.Request) {
	returnTo := r.URL.Query().Get("returnTo")
	if returnTo == "" {
		returnTo = s.C.SiteURL + "/admin"
	}
	u, e := url.Parse(returnTo)
	allowed := false
	if e == nil {
		for _, o := range s.C.Origins {
			allowed = allowed || (u.Scheme+"://"+u.Host == o && u.User == nil)
		}
	}
	if !allowed {
		fail(w, bad("Invalid return address."))
		return
	}
	binding := token()
	s.cookie(w, "chloe_flow", binding, 600)
	flow, e := s.newToken(r.Context(), "portfolio_flow", "", map[string]any{"binding": hash(binding), "returnTo": returnTo}, 10*time.Minute)
	if e != nil {
		fail(w, e)
		return
	}
	http.Redirect(w, r, s.C.Issuer+"/login?flow="+url.QueryEscape(flow), 303)
}
func (s *Server) sessionCallback(w http.ResponseWriter, r *http.Request) {
	uid, p, e := s.consume(r.Context(), "portfolio_ticket", r.URL.Query().Get("ticket"))
	if e != nil {
		fail(w, e)
		return
	}
	if !equal(str(p, "binding"), hash(s.cookieValue(r, "chloe_flow"))) {
		fail(w, problem{403, "Sign-in could not be matched to this browser."})
		return
	}
	mfa, _ := p["mfa"].(bool)
	if e = s.newSession(w, r, uid, "api", mfa); e != nil {
		fail(w, e)
		return
	}
	s.cookie(w, "chloe_flow", "", -1)
	http.Redirect(w, r, str(p, "returnTo"), 303)
}
func (s *Server) finishLogin(w http.ResponseWriter, r *http.Request, u *User) {
	if flow := r.FormValue("flow"); flow != "" {
		_, p, e := s.consume(r.Context(), "portfolio_flow", flow)
		if e != nil {
			fail(w, e)
			return
		}
		p["mfa"] = u.MFA
		t, e := s.newToken(r.Context(), "portfolio_ticket", u.ID, p, time.Minute)
		if e != nil {
			fail(w, e)
			return
		}
		http.Redirect(w, r, s.C.APIURL+"/v1/session/callback?ticket="+url.QueryEscape(t), 303)
		return
	}
	next := r.FormValue("continue")
	if !strings.HasPrefix(next, "/oauth/authorize?") {
		next = "/account"
	}
	http.Redirect(w, r, s.C.Issuer+next, 303)
}

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="robots" content="noindex,nofollow"><title>{{.Title}} · Chloe ID</title><link rel="stylesheet" href="/identity.css"></head><body><main><nav><a href="/account">Chloe ID ↗</a><a href="{{.Site}}">Back to website</a></nav><h1>{{.Title}}</h1>{{if .Error}}<p class="error" role="alert">{{.Error}}</p>{{end}}{{if .Notice}}<p class="notice" role="status">{{.Notice}}</p>{{end}}{{template "body" .}}</main></body></html>`))

func (s *Server) page(w http.ResponseWriter, r *http.Request, title, body string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	data["Title"] = title
	data["Site"] = s.C.SiteURL
	data["CSRF"] = s.csrfToken(w, r)
	data["Flow"] = r.FormValue("flow")
	data["Continue"] = r.FormValue("continue")
	data["Reauth"] = r.FormValue("reauth")
	t, e := pageTemplate.Clone()
	if e == nil {
		_, e = t.Parse(`{{define "body"}}` + body + `{{end}}`)
	}
	if e != nil {
		fail(w, e)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	formTargets := "'self' " + s.C.SiteURL + " " + s.C.APIURL
	if target, ok := data["FormTarget"].(string); ok {
		formTargets += " " + target
	}
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; font-src 'self'; script-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action "+formTargets)
	if e = t.Execute(w, data); e != nil {
		return
	}
}

const hiddenForm = `<input type="hidden" name="reauth" value="{{.Reauth}}"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="flow" value="{{.Flow}}"><input type="hidden" name="continue" value="{{.Continue}}">`
const loginForm = `<p class="muted">One account for this site and your connected projects.</p><form method="post">` + hiddenForm + `<label>Email<input type="email" name="email" required autocomplete="username"></label><label>Password<input type="password" name="password" required autocomplete="current-password" maxlength="72"></label><button>Sign in</button></form><div class="actions">{{if .Google}}<a href="/social/google/start?flow={{.Flow}}&continue={{.Continue}}">Continue with Google</a>{{end}}{{if .GitHub}}<a href="/social/github/start?flow={{.Flow}}&continue={{.Continue}}">Continue with GitHub</a>{{end}}</div><div class="actions"><a href="/passkey-login?flow={{.Flow}}&continue={{.Continue}}">Use a passkey</a><a href="/recover">Forgot password?</a>{{if .Registration}}<a href="/register">Create account</a>{{end}}</div>`

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" && r.FormValue("reauth") != "true" {
		if u := s.user(r, "identity"); u != nil && u.Verified && (!u.Owner || u.MFA) {
			s.finishLogin(w, r, u)
			return
		}
	}
	data := map[string]any{"Google": s.C.GoogleID != "", "GitHub": s.C.GitHubID != "", "Registration": s.C.Registration}
	if r.Method == "POST" {
		if !s.csrf(w, r) {
			return
		}
		email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		e := s.limit(r.Context(), "login:"+s.ip(r), 30, 15*time.Minute)
		if e == nil {
			e = s.limit(r.Context(), "login-email:"+email, 10, 15*time.Minute)
		}
		if e != nil {
			data["Error"] = e.Error()
		} else {
			var u User
			var password string
			e = s.DB.QueryRow(r.Context(), "SELECT id,name,email,password_hash,verified,owner FROM accounts WHERE email=$1", email).Scan(&u.ID, &u.Name, &u.Email, &password, &u.Verified, &u.Owner)
			if e != nil {
				password = "$2a$12$hDUz35kJuhgBmVlRJu91cu73NIaPdaOWCiYOQLtOBBHXFHyYWyyze"
			}
			match := bcrypt.CompareHashAndPassword([]byte(password), []byte(r.FormValue("password"))) == nil
			if e != nil || !match || !u.Verified {
				data["Error"] = "Sign-in failed. Check your details and verify your email."
			} else {
				s.audit(r.Context(), u.ID, "login.password", "")
				s.completeIdentity(w, r, &u)
				return
			}
		}
	}
	s.page(w, r, "Welcome back.", loginForm, data)
}
func validEmail(v string) bool {
	a, e := mail.ParseAddress(v)
	return e == nil && a.Address == v && len(v) <= 254 && !strings.ContainsAny(v, "\r\n")
}
func passwordHash(v string) (string, error) {
	if len(v) < 12 || len(v) > 72 {
		return "", bad("Use a password between 12 and 72 bytes.")
	}
	b, e := bcrypt.GenerateFromPassword([]byte(v), 12)
	return string(b), e
}
func (s *Server) queueMail(ctx context.Context, to, subject, body string) error {
	_, e := s.DB.Exec(ctx, "INSERT INTO email_jobs(id,recipient,subject,body) VALUES($1,$2,$3,$4)", id(), to, subject, body)
	return e
}
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if !s.C.Registration {
		s.page(w, r, "Registration is closed.", `<p>Account registration will open once mail delivery is configured.</p>`, nil)
		return
	}
	data := map[string]any{}
	if r.Method == "POST" {
		if !s.csrf(w, r) {
			return
		}
		email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		name := strings.TrimSpace(r.FormValue("name"))
		e := s.limit(r.Context(), "register:"+s.ip(r), 5, time.Hour)
		if e == nil && (!validEmail(email) || len(name) < 1 || len(name) > 100) {
			e = bad("Enter a valid name and email.")
		}
		var pw string
		if e == nil {
			pw, e = passwordHash(r.FormValue("password"))
		}
		if e != nil {
			data["Error"] = e.Error()
		} else {
			tx, e := s.DB.Begin(r.Context())
			if e != nil {
				fail(w, e)
				return
			}
			defer tx.Rollback(r.Context())
			uid := id()
			res, e := tx.Exec(r.Context(), "INSERT INTO accounts(id,email,name,password_hash) VALUES($1,$2,$3,$4) ON CONFLICT(email) DO NOTHING", uid, email, name, pw)
			if e != nil {
				fail(w, e)
				return
			}
			if res.RowsAffected() == 1 {
				v := token()
				_, e = tx.Exec(r.Context(), "INSERT INTO one_time_tokens(hash,purpose,account_id,payload,expires_at) VALUES($1,'verify',$2,'{}',now()+interval '24 hours')", hash(v), uid)
				if e == nil {
					_, e = tx.Exec(r.Context(), "INSERT INTO email_jobs(id,recipient,subject,body) VALUES($1,$2,$3,$4)", id(), email, "Verify your Chloe ID account", "Confirm your email by opening this link:\n\n"+s.C.Issuer+"/verify?token="+url.QueryEscape(v)+"\n\nThis link expires in 24 hours.")
				}
			}
			if e == nil {
				e = tx.Commit(r.Context())
			}
			if e != nil {
				fail(w, e)
				return
			}
			data["Notice"] = "If this address is available, a verification email is on its way."
		}
	}
	s.page(w, r, "Make yourself at home.", `<form method="post">`+hiddenForm+`<label>Name<input name="name" required maxlength="100" autocomplete="name"></label><label>Email<input name="email" type="email" required autocomplete="email"></label><label>Password <small>At least 12 characters</small><input name="password" type="password" required minlength="12" maxlength="72" autocomplete="new-password"></label><button>Create account</button></form><p><a href="/login">Already have an account?</a></p>`, data)
}
func (s *Server) verify(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{"Token": r.FormValue("token")}
	if r.Method == "POST" {
		if !s.csrf(w, r) {
			return
		}
		uid, _, e := s.consume(r.Context(), "verify", r.FormValue("token"))
		if e == nil {
			_, e = s.DB.Exec(r.Context(), "UPDATE accounts SET verified=true WHERE id=$1", uid)
		}
		if e != nil {
			data["Error"] = e.Error()
		} else {
			data["Notice"] = "Email verified. You can now sign in."
		}
	}
	s.page(w, r, "Verify your email.", `<form method="post">`+hiddenForm+`<input type="hidden" name="token" value="{{.Token}}"><button>Verify email</button></form><p><a href="/login">Sign in</a></p>`, data)
}
func (s *Server) recover(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	if r.Method == "POST" {
		if !s.csrf(w, r) {
			return
		}
		email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		e := s.limit(r.Context(), "recover:"+s.ip(r), 5, time.Hour)
		if e == nil {
			e = s.limit(r.Context(), "recover-email:"+email, 3, time.Hour)
		}
		if e != nil {
			data["Error"] = e.Error()
		} else {
			var uid string
			var verified bool
			if s.C.SMTPUser != "" && s.DB.QueryRow(r.Context(), "SELECT id,verified FROM accounts WHERE email=$1", email).Scan(&uid, &verified) == nil {
				purpose, path, subject, duration := "reset", "/reset", "Reset your Chloe ID password", 30*time.Minute
				if !verified {
					purpose, path, subject, duration = "verify", "/verify", "Verify your Chloe ID account", 24*time.Hour
				}
				t, e := s.newToken(r.Context(), purpose, uid, map[string]any{}, duration)
				if e == nil {
					e = s.queueMail(r.Context(), email, subject, "Continue account recovery with this link:\n\n"+s.C.Issuer+path+"?token="+url.QueryEscape(t)+"\n\nIf you did not request this email, you can ignore it.")
				}
				if e != nil {
					fail(w, e)
					return
				}
			}
			data["Notice"] = "If an account can be recovered, instructions will arrive by email."
		}
	}
	s.page(w, r, "Find your way back.", `<form method="post">`+hiddenForm+`<label>Email<input name="email" type="email" required autocomplete="email"></label><button>Send recovery instructions</button></form><p><a href="/login">Back to sign in</a></p>`, data)
}
func (s *Server) reset(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{"Token": r.FormValue("token")}
	if r.Method == "POST" {
		if !s.csrf(w, r) {
			return
		}
		pw, e := passwordHash(r.FormValue("password"))
		if e == nil {
			var uid string
			uid, _, e = s.consume(r.Context(), "reset", r.FormValue("token"))
			if e == nil {
				tx, err := s.DB.Begin(r.Context())
				if err != nil {
					fail(w, err)
					return
				}
				defer tx.Rollback(r.Context())
				_, e = tx.Exec(r.Context(), "UPDATE accounts SET password_hash=$2 WHERE id=$1", uid, pw)
				if e == nil {
					_, e = tx.Exec(r.Context(), "DELETE FROM sessions WHERE account_id=$1", uid)
				}
				if e == nil {
					_, e = tx.Exec(r.Context(), "UPDATE oauth_sessions SET active=false WHERE account_id=$1", uid)
				}
				if e == nil {
					_, e = tx.Exec(r.Context(), "DELETE FROM one_time_tokens WHERE account_id=$1 AND purpose='reset'", uid)
				}
				if e == nil {
					e = tx.Commit(r.Context())
				}
				s.audit(r.Context(), uid, "password.reset", "")
			}
		}
		if e != nil {
			data["Error"] = e.Error()
		} else {
			data["Notice"] = "Password updated. All existing sessions were signed out."
		}
	}
	s.page(w, r, "A fresh start.", `<form method="post">`+hiddenForm+`<input type="hidden" name="token" value="{{.Token}}"><label>New password<input name="password" type="password" required minlength="12" maxlength="72" autocomplete="new-password"></label><button>Reset password</button></form><p><a href="/login">Sign in</a></p>`, data)
}
func (s *Server) identityUser(w http.ResponseWriter, r *http.Request) *User {
	u := s.user(r, "identity")
	if u == nil || !u.Verified {
		http.Redirect(w, r, "/login", 303)
		return nil
	}
	return u
}
func (s *Server) completeIdentity(w http.ResponseWriter, r *http.Request, u *User) {
	if e := s.newSession(w, r, u.ID, "identity", false); e != nil {
		fail(w, e)
		return
	}
	if u.Owner {
		http.Redirect(w, r, "/step-up?flow="+url.QueryEscape(r.FormValue("flow"))+"&continue="+url.QueryEscape(r.FormValue("continue")), 303)
		return
	}
	s.finishLogin(w, r, u)
}
