package platform

import (
	"net/http"
	"strings"
	"time"
)

func (s *Server) routes(m *http.ServeMux) {
	m.HandleFunc("GET /v1/health", func(w http.ResponseWriter, r *http.Request) {
		if e := s.DB.Ping(r.Context()); e != nil {
			fail(w, e)
			return
		}
		send(w, 200, map[string]any{"ok": true})
	})
	m.HandleFunc("GET /v1/content", s.published)
	m.HandleFunc("GET /v1/account", s.accountAPI)
	m.HandleFunc("PATCH /v1/account", s.accountAPI)
	m.HandleFunc("GET /v1/session", s.session)
	m.HandleFunc("DELETE /v1/session", s.session)
	m.HandleFunc("GET /v1/session/start", s.sessionStart)
	m.HandleFunc("GET /v1/session/callback", s.sessionCallback)
	m.HandleFunc("GET /v1/admin/content", s.contentIndex)
	m.HandleFunc("GET /v1/admin/content/{id}", s.contentItem)
	m.HandleFunc("POST /v1/admin/content/{id}", s.contentItem)
	for _, method := range []string{"GET", "POST"} {
		m.HandleFunc(method+" /v1/admin/content/{id}/revisions", s.revisions)
	}
	for _, method := range []string{"GET", "POST", "DELETE"} {
		m.HandleFunc(method+" /v1/admin/content/{id}/schedules", s.schedules)
	}
	m.HandleFunc("POST /v1/admin/content/{id}/organize", s.organize)
	m.HandleFunc("GET /v1/admin/trash", s.trash)
	m.HandleFunc("POST /v1/contact", s.contact)
	m.HandleFunc("GET /v1/guestbook", s.guestbook)
	m.HandleFunc("POST /v1/guestbook", s.guestbook)
	m.HandleFunc("DELETE /v1/guestbook/{id}", s.guestbookAction)
	m.HandleFunc("PATCH /v1/guestbook/{id}", s.guestbookAction)
	m.HandleFunc("GET /v1/admin/guestbook", s.guestbook)
	m.HandleFunc("GET /v1/admin/inbox", s.inbox)
	m.HandleFunc("PATCH /v1/admin/inbox", s.inbox)
	m.HandleFunc("GET /v1/admin/media", s.mediaIndex)
	m.HandleFunc("POST /v1/admin/media", s.upload)
	m.HandleFunc("PATCH /v1/admin/media/{id}", s.mediaEdit)
	m.HandleFunc("GET /media/{filename}", s.mediaFile)
	m.HandleFunc("GET /v1/admin/applications", s.applications)
	m.HandleFunc("POST /v1/admin/applications", s.applications)
	m.HandleFunc("PATCH /v1/admin/applications/{id}", s.applications)
	m.HandleFunc("GET /v1/admin/status", s.status)
}
func (s *Server) identityRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /fonts/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name != "manrope.woff2" && name != "space-grotesk.woff2" {
			http.NotFound(w, r)
			return
		}
		f, e := s.Files.Open("web/" + name)
		if e != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "font/woff2")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeContent(w, r, name, time.Time{}, mustReadSeeker(f))
	})
	m.HandleFunc("GET /identity.css", func(w http.ResponseWriter, r *http.Request) {
		b, e := s.Files.Open("web/identity.css")
		if e != nil {
			http.NotFound(w, r)
			return
		}
		defer b.Close()
		w.Header().Set("Content-Type", "text/css")
		http.ServeContent(w, r, "identity.css", time.Time{}, mustReadSeeker(b))
	})
	m.HandleFunc("GET /passkeys.js", s.passkeyScript)
	for path, handler := range map[string]http.HandlerFunc{"/login": s.login, "/register": s.register, "/verify": s.verify, "/recover": s.recover, "/reset": s.reset, "/step-up": s.stepUp, "/account/authenticator": s.authenticator, "/account": s.account, "/logout": s.logout, "/oauth/authorize": s.authorize} {
		m.HandleFunc("GET "+path, handler)
		m.HandleFunc("POST "+path, handler)
	}
	m.HandleFunc("GET /social/{provider}/start", s.socialStart)
	m.HandleFunc("GET /social/{provider}/callback", s.socialCallback)
	m.HandleFunc("GET /passkey-login", s.passkeyPage)
	m.HandleFunc("POST /passkeys/{action}", s.passkeyAction)
	m.HandleFunc("GET /.well-known/openid-configuration", s.discovery)
	m.HandleFunc("GET /.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) { send(w, 200, s.Keys.Public()) })
	m.HandleFunc("POST /oauth/token", s.oauthAtomic(s.oauthToken))
	m.HandleFunc("POST /oauth/revoke", s.oauthAtomic(s.oauthRevoke))
	m.HandleFunc("POST /oauth/introspect", s.oauthAtomic(s.oauthIntrospect))
	m.HandleFunc("GET /oauth/userinfo", s.userInfo)
}
func (s *Server) contact(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Subject string `json:"subject"`
		Message string `json:"message"`
		Website string `json:"website"`
	}
	if e := decode(w, r, &in); e != nil {
		fail(w, e)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.TrimSpace(in.Email)
	if len(in.Name) < 1 || len(in.Name) > 100 || !validEmail(in.Email) || len(in.Subject) < 1 || len(in.Subject) > 200 || len(in.Message) < 10 || len(in.Message) > 5000 || in.Website != "" {
		fail(w, bad("Enter a name, valid email, subject, and message of 10–5000 characters."))
		return
	}
	if e := s.limit(r.Context(), "contact:"+s.ip(r), 5, time.Hour); e != nil {
		fail(w, e)
		return
	}
	if _, e := s.DB.Exec(r.Context(), "INSERT INTO messages VALUES($1,$2,$3,$4,$5,now(),false)", id(), in.Name, in.Email, in.Subject, in.Message); e != nil {
		fail(w, e)
		return
	}
	send(w, 201, map[string]any{"ok": true})
}
func (s *Server) guestbook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u := s.user(r, "api")
	if r.Method == "POST" {
		u = s.require(w, r, false)
		if u == nil {
			return
		}
		var in struct {
			Name    string `json:"name"`
			Message string `json:"message"`
		}
		if e := decode(w, r, &in); e != nil {
			fail(w, e)
			return
		}
		in.Name = strings.TrimSpace(in.Name)
		in.Message = strings.TrimSpace(in.Message)
		if len(in.Name) < 1 || len(in.Name) > 60 || len(in.Message) < 2 || len(in.Message) > 500 {
			fail(w, bad("Use a name up to 60 characters and a message of 2–500 characters."))
			return
		}
		if e := s.limit(ctx, "guestbook:"+u.ID, 5, time.Hour); e != nil {
			fail(w, e)
			return
		}
		if _, e := s.DB.Exec(ctx, "INSERT INTO guestbook VALUES($1,$2,$3,$4,'pending',now())", id(), u.ID, in.Name, in.Message); e != nil {
			fail(w, e)
			return
		}
		send(w, 201, map[string]any{"ok": true})
		return
	}
	owner := strings.HasPrefix(r.URL.Path, "/v1/admin/")
	if owner {
		s.require(w, r, true)
		if !u.authenticated() || !u.Owner {
			return
		}
	}
	uid := ""
	if u != nil {
		uid = u.ID
	}
	rows, e := s.DB.Query(ctx, "SELECT id,name,message,status,created_at,author_uid=$1 FROM guestbook WHERE $2 OR status='approved' OR author_uid=$1 ORDER BY created_at DESC LIMIT 200", uid, owner)
	if e != nil {
		fail(w, e)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var key, name, message, status string
		var date time.Time
		var own bool
		if e = rows.Scan(&key, &name, &message, &status, &date, &own); e != nil {
			fail(w, e)
			return
		}
		out = append(out, map[string]any{"id": key, "name": name, "message": message, "status": status, "createdAt": date, "own": own})
	}
	send(w, 200, map[string]any{"entries": out})
}
func (s *Server) guestbookAction(w http.ResponseWriter, r *http.Request) {
	u := s.require(w, r, r.Method == "PATCH")
	if u == nil {
		return
	}
	var err error
	key := r.PathValue("id")
	if r.Method == "DELETE" {
		res, e := s.DB.Exec(r.Context(), "DELETE FROM guestbook WHERE id::text=$1 AND (author_uid=$2 OR $3)", key, u.ID, u.Owner && u.authenticated())
		err = e
		if e == nil && res.RowsAffected() == 0 {
			err = problem{404, "Entry not found."}
		}
	} else {
		var in struct {
			Status string `json:"status"`
		}
		if e := decode(w, r, &in); e != nil {
			fail(w, e)
			return
		}
		if in.Status != "pending" && in.Status != "approved" && in.Status != "rejected" {
			fail(w, bad("Invalid moderation state."))
			return
		}
		_, err = s.DB.Exec(r.Context(), "UPDATE guestbook SET status=$2 WHERE id::text=$1", key, in.Status)
	}
	if err != nil {
		fail(w, err)
		return
	}
	send(w, 200, map[string]any{"ok": true})
}
func (s *Server) inbox(w http.ResponseWriter, r *http.Request) {
	if s.require(w, r, true) == nil {
		return
	}
	if r.Method == "PATCH" {
		var in struct {
			ID   string `json:"id"`
			Read bool   `json:"read"`
		}
		if e := decode(w, r, &in); e != nil {
			fail(w, e)
			return
		}
		if _, e := s.DB.Exec(r.Context(), "UPDATE messages SET is_read=$2 WHERE id::text=$1", in.ID, in.Read); e != nil {
			fail(w, e)
			return
		}
		send(w, 200, map[string]any{"ok": true})
		return
	}
	rows, e := s.DB.Query(r.Context(), "SELECT id,name,email,subject,message,created_at,is_read FROM messages ORDER BY created_at DESC LIMIT 500")
	if e != nil {
		fail(w, e)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var key, name, email, subject, message string
		var date time.Time
		var read bool
		if e = rows.Scan(&key, &name, &email, &subject, &message, &date, &read); e != nil {
			fail(w, e)
			return
		}
		out = append(out, map[string]any{"id": key, "name": name, "email": email, "subject": subject, "message": message, "created_at": date, "is_read": read})
	}
	send(w, 200, map[string]any{"messages": out})
}
func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	if s.require(w, r, true) == nil {
		return
	}
	out := map[string]any{}
	for name, q := range map[string]string{"overdueSchedules": "SELECT count(*) FROM schedules WHERE status='pending' AND run_at<now()-interval '2 minutes'", "failedSchedules": "SELECT count(*) FROM schedules WHERE status='failed'", "pendingMail": "SELECT count(*) FROM email_jobs WHERE status='pending'", "failedMail": "SELECT count(*) FROM email_jobs WHERE status='failed'"} {
		var n int
		if e := s.DB.QueryRow(r.Context(), q).Scan(&n); e != nil {
			fail(w, e)
			return
		}
		out[name] = n
	}
	send(w, 200, out)
}
