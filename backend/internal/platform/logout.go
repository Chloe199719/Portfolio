package platform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
)

// Each host owns its own __Host- cookie. Establish an API-host browser binding
// before confirmation, then return a one-use ticket after the CSRF-checked POST.
func (s *Server) logoutStart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if e := s.limit(r.Context(), "logout:"+s.ip(r), 30, 15*time.Minute); e != nil {
		fail(w, e)
		return
	}
	binding := token()
	flow, e := s.newToken(r.Context(), "logout_flow", "", Document{"binding": hash(binding)}, 10*time.Minute)
	if e != nil {
		fail(w, e)
		return
	}
	s.cookie(w, "chloe_logout", binding, 600)
	http.Redirect(w, r, s.C.Issuer+"/logout?flow="+url.QueryEscape(flow), http.StatusSeeOther)
}

func consumeLogoutToken(ctx context.Context, tx pgx.Tx, purpose, value string) (Document, error) {
	var payload Document
	e := tx.QueryRow(ctx, "DELETE FROM one_time_tokens WHERE hash=$1 AND purpose=$2 AND expires_at>now() RETURNING payload", hash(value), purpose).Scan(&payload)
	if errors.Is(e, pgx.ErrNoRows) {
		e = bad("This sign-out request expired or was already used. Start sign-out again.")
	}
	return payload, e
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "POST" {
		if r.FormValue("flow") == "" {
			http.Redirect(w, r, s.C.APIURL+"/v1/session/logout/start", http.StatusSeeOther)
			return
		}
		s.page(w, r, "Sign out?", `<p>This signs you out of Chloe ID and this website in this browser. Other devices and connected applications keep their own sessions.</p><form method="post">`+hiddenForm+`<button>Sign out</button></form>`, nil)
		return
	}
	if !s.csrf(w, r) {
		return
	}
	ctx := r.Context()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		fail(w, e)
		return
	}
	defer tx.Rollback(ctx)
	payload, e := consumeLogoutToken(ctx, tx, "logout_flow", r.FormValue("flow"))
	if e != nil {
		fail(w, e)
		return
	}
	ticket := token()
	raw, e := json.Marshal(payload)
	if e == nil {
		_, e = tx.Exec(ctx, "INSERT INTO one_time_tokens(hash,purpose,payload,expires_at) VALUES($1,'logout_ticket',$2,$3)", hash(ticket), raw, time.Now().Add(time.Minute))
	}
	if e == nil {
		_, e = tx.Exec(ctx, "DELETE FROM sessions WHERE hash=$1 AND kind='identity'", hash(s.cookieValue(r, "chloe_identity")))
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		fail(w, e)
		return
	}
	s.cookie(w, "chloe_identity", "", -1)
	http.Redirect(w, r, s.C.APIURL+"/v1/session/logout/callback?ticket="+url.QueryEscape(ticket), http.StatusSeeOther)
}

func (s *Server) logoutCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ctx := r.Context()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		fail(w, e)
		return
	}
	defer tx.Rollback(ctx)
	payload, e := consumeLogoutToken(ctx, tx, "logout_ticket", r.URL.Query().Get("ticket"))
	if e != nil {
		fail(w, e)
		return
	}
	if !equal(str(payload, "binding"), hash(s.cookieValue(r, "chloe_logout"))) {
		fail(w, problem{403, "Sign-out could not be matched to this browser. Start sign-out again."})
		return
	}
	if _, e = tx.Exec(ctx, "DELETE FROM sessions WHERE hash=$1 AND kind='api'", hash(s.cookieValue(r, "chloe_api"))); e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		fail(w, e)
		return
	}
	s.cookie(w, "chloe_api", "", -1)
	s.cookie(w, "chloe_logout", "", -1)
	http.Redirect(w, r, s.C.SiteURL, http.StatusSeeOther)
}
