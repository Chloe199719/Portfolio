package platform

import (
	"net/http"
	"strings"
	"time"
)

const accountBody = `<p class="muted">{{.User.Email}}</p><h2>Linked accounts</h2>{{range .Providers}}<form method="post" class="record"><input type="hidden" name="csrf" value="{{$.CSRF}}"><input type="hidden" name="provider" value="{{.}}"><span>{{.}}</span> <button name="action" value="unlink">Unlink</button></form>{{end}}<form method="post">` + hiddenForm + `<input type="hidden" name="action" value="link"><div class="actions">{{if .Google}}<button name="provider" value="google">Link Google</button>{{end}}{{if .GitHub}}<button name="provider" value="github">Link GitHub</button>{{end}}</div></form><h2>Password</h2><form method="post">` + hiddenForm + `<input type="hidden" name="action" value="password"><label>Current password <small>Leave empty if you only use social sign-in</small><input name="current" type="password" autocomplete="current-password"></label><label>New password<input name="password" type="password" required minlength="12" maxlength="72" autocomplete="new-password"></label><button>Update password</button></form><h2>Passkeys</h2><div id="passkeys" data-csrf="{{.CSRF}}"><label>Passkey name<input id="passkey-name" value="My device" maxlength="80"></label><button type="button" id="register-passkey">Add a passkey</button><p id="passkey-status" role="status"></p></div>{{range .Passkeys}}<form method="post" class="record"><input type="hidden" name="csrf" value="{{$.CSRF}}"><input type="hidden" name="id" value="{{.id}}">{{.name}} <button name="action" value="remove-passkey">Remove</button></form>{{end}}<h2>Active sessions</h2>{{range .Sessions}}<form method="post" class="record"><input type="hidden" name="csrf" value="{{$.CSRF}}"><input type="hidden" name="id" value="{{.id}}"><p>{{.kind}} · {{.createdAt}}<br><small>{{.agent}}</small></p><button name="action" value="revoke-session">Revoke</button></form>{{end}}<h2>Connected applications</h2>{{range .Apps}}<form method="post" class="record"><input type="hidden" name="csrf" value="{{$.CSRF}}"><input type="hidden" name="id" value="{{.id}}"><p>{{.name}}</p><button name="action" value="revoke-app">Disconnect</button></form>{{else}}<p class="muted">No connected applications.</p>{{end}}<p><a href="/logout">Sign out</a></p><script src="/passkeys.js" defer></script>`

func (s *Server) account(w http.ResponseWriter, r *http.Request) {
	u := s.identityUser(w, r)
	if u == nil {
		return
	}
	if u.Owner && !u.MFA {
		http.Redirect(w, r, "/step-up?continue=/account", 303)
		return
	}
	ctx := r.Context()
	data := map[string]any{"User": u, "Google": s.C.GoogleID != "", "GitHub": s.C.GitHubID != ""}
	if r.Method == "POST" {
		if !s.csrf(w, r) {
			return
		}
		if e := s.limit(ctx, "account:"+u.ID, 30, time.Hour); e != nil {
			fail(w, e)
			return
		}
		action := r.FormValue("action")
		if action == "link" {
			s.beginSocial(w, r, r.FormValue("provider"), u.ID)
			return
		}
		e := s.changeAccount(ctx, u, AccountMutation{Action: action, Current: r.FormValue("current"), Password: r.FormValue("password"), Provider: r.FormValue("provider"), ID: r.FormValue("id")})
		if e != nil {
			data["Error"] = e.Error()
		} else {
			s.audit(ctx, u.ID, "account."+action, "")
			data["Notice"] = "Account updated."
		}
	}
	providers := []string{}
	rows, e := s.DB.Query(ctx, "SELECT provider FROM social_accounts WHERE account_id=$1", u.ID)
	if e != nil {
		fail(w, e)
		return
	}
	for rows.Next() {
		var v string
		if e = rows.Scan(&v); e != nil {
			rows.Close()
			fail(w, e)
			return
		}
		providers = append(providers, v)
	}
	rows.Close()
	data["Providers"] = providers
	for key, q := range map[string]string{"Passkeys": "SELECT id,name FROM passkeys WHERE account_id=$1 ORDER BY created_at DESC", "Apps": "SELECT c.id,c.name FROM oauth_consents o JOIN oauth_clients c ON c.id=o.client_id WHERE o.account_id=$1"} {
		rows, e = s.DB.Query(ctx, q, u.ID)
		if e != nil {
			fail(w, e)
			return
		}
		out := []any{}
		for rows.Next() {
			var key, name string
			if e = rows.Scan(&key, &name); e != nil {
				rows.Close()
				fail(w, e)
				return
			}
			out = append(out, map[string]any{"id": key, "name": name})
		}
		rows.Close()
		data[key] = out
	}
	rows, e = s.DB.Query(ctx, "SELECT hash,kind,agent,created_at FROM sessions WHERE account_id=$1 AND expires_at>now() ORDER BY created_at DESC", u.ID)
	if e != nil {
		fail(w, e)
		return
	}
	sessions := []any{}
	for rows.Next() {
		var key, kind, agent string
		var date time.Time
		if e = rows.Scan(&key, &kind, &agent, &date); e != nil {
			rows.Close()
			fail(w, e)
			return
		}
		sessions = append(sessions, map[string]any{"id": key, "kind": kind, "agent": strings.TrimSpace(agent), "createdAt": date.Format(time.RFC822)})
	}
	rows.Close()
	data["Sessions"] = sessions
	s.page(w, r, "Your account.", accountBody, data)
}
