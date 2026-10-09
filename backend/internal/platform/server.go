package platform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/fosite"
)

type Server struct {
	C        Config
	DB       *pgxpool.Pool
	Files    fs.FS
	OAuth    fosite.OAuth2Provider
	Keys     *KeyRing
	WebAuthn *webauthn.WebAuthn
}
type problem struct {
	Code    int
	Message string
}

func (e problem) Error() string { return e.Message }
func bad(s string) error        { return problem{400, s} }
func fail(w http.ResponseWriter, e error) {
	var p problem
	if errors.As(e, &p) {
		send(w, p.Code, map[string]any{"error": p.Message})
		return
	}
	slog.Error("request failed", "error", e)
	send(w, 500, map[string]any{"error": "The server could not complete this request. Please try again."})
}
func send(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return bad("Invalid request body.")
	}
	if e := d.Decode(new(any)); e != io.EOF {
		return bad("Expected one JSON object.")
	}
	return nil
}
func token() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func hash(v string) string { b := sha256.Sum256([]byte(v)); return hex.EncodeToString(b[:]) }
func equal(a, b string) bool {
	return len(a) > 0 && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func id() string { return uuid.NewString() }
func (s *Server) cookie(w http.ResponseWriter, name, value string, seconds int) {
	if s.C.Production {
		name = "__Host-" + name
	}
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: s.C.Production, SameSite: http.SameSiteLaxMode, MaxAge: seconds})
}
func (s *Server) cookieValue(r *http.Request, name string) string {
	if s.C.Production {
		name = "__Host-" + name
	}
	c, e := r.Cookie(name)
	if e != nil {
		return ""
	}
	return c.Value
}
func (s *Server) csrf(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	allowed := origin == s.C.Issuer || origin == s.C.APIURL
	for _, v := range s.C.Origins {
		allowed = allowed || origin == v
	}
	if !allowed {
		fail(w, problem{403, "This origin is not allowed."})
		return false
	}
	value := r.Header.Get("X-CSRF-Token")
	if value == "" && strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		value = r.FormValue("csrf")
	}
	if !equal(s.cookieValue(r, "chloe_csrf"), value) {
		fail(w, problem{403, "This form expired. Refresh and try again."})
		return false
	}
	return true
}
func (s *Server) csrfToken(w http.ResponseWriter, r *http.Request) string {
	v := s.cookieValue(r, "chloe_csrf")
	if len(v) < 32 {
		v = token()
		s.cookie(w, "chloe_csrf", v, 86400)
	}
	return v
}
func (s *Server) limit(ctx context.Context, key string, max int, window time.Duration) error {
	now := time.Now().UnixMilli()
	var count int
	e := s.DB.QueryRow(ctx, `INSERT INTO rate_limits(key,count,reset_at) VALUES($1,1,$2) ON CONFLICT(key) DO UPDATE SET count=CASE WHEN rate_limits.reset_at<$3 THEN 1 ELSE rate_limits.count+1 END,reset_at=CASE WHEN rate_limits.reset_at<$3 THEN $2 ELSE rate_limits.reset_at END RETURNING count`, hash(key), now+window.Milliseconds(), now).Scan(&count)
	if e != nil {
		return e
	}
	if count > max {
		return problem{429, "Too many attempts. Please try again later."}
	}
	return nil
}
func (s *Server) ip(r *http.Request) string {
	if s.C.Production {
		return r.Header.Get("X-Real-IP")
	}
	h, _, _ := net.SplitHostPort(r.RemoteAddr)
	return h
}
func (s *Server) audit(ctx context.Context, actor, event, target string) {
	if _, e := s.DB.Exec(ctx, "INSERT INTO audit_events(actor,event,target) VALUES($1,$2,$3)", actor, event, target); e != nil {
		slog.Error("audit failed", "error", e)
	}
}
func New(ctx context.Context, c Config, files fs.FS) (*Server, error) {
	db, e := pgxpool.New(ctx, c.DatabaseURL)
	if e != nil {
		return nil, e
	}
	if e = db.Ping(ctx); e != nil {
		db.Close()
		return nil, e
	}
	s := &Server{C: c, DB: db, Files: files}
	if e = s.migrate(ctx); e != nil {
		db.Close()
		return nil, e
	}
	if e = os.MkdirAll(filepath.Join(c.DataDir, "media"), 0700); e != nil {
		db.Close()
		return nil, e
	}
	if e = s.initIdentity(ctx); e != nil {
		db.Close()
		return nil, e
	}
	return s, nil
}
func (s *Server) migrate(ctx context.Context) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(487139); CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY,applied_at TIMESTAMPTZ DEFAULT now())"); e != nil {
		return e
	}
	names, e := fs.Glob(s.Files, "migrations/*.sql")
	if e != nil {
		return e
	}
	for _, name := range names {
		var exists bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", name).Scan(&exists); e != nil {
			return e
		}
		if exists {
			continue
		}
		b, e := fs.ReadFile(s.Files, name)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, string(b)); e != nil {
			return fmt.Errorf("migration %s: %w", name, e)
		}
		if _, e = tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES($1)", name); e != nil {
			return e
		}
	}
	var seeded bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM meta WHERE key='seed-version')").Scan(&seeded); e != nil {
		return e
	}
	if !seeded {
		b, e := fs.ReadFile(s.Files, "seed/content.json")
		if e != nil {
			return e
		}
		var docs []map[string]any
		if e = json.Unmarshal(b, &docs); e != nil {
			return e
		}
		docs = append(docs, map[string]any{"_id": "siteSettings", "_type": "siteSettings", "homepageTitle": "Software. Side quests. Me.", "homepageSubtitle": "Things I build, moments I notice, and interests I keep coming back to."})
		for _, d := range docs {
			raw, _ := json.Marshal(d)
			if _, e = tx.Exec(ctx, "INSERT INTO content VALUES($1,$2,'published',$3,$4,now()) ON CONFLICT DO NOTHING", d["_id"], d["_type"], raw, id()); e != nil {
				return e
			}
		}
		if _, e = tx.Exec(ctx, "INSERT INTO meta VALUES('seed-version','1')"); e != nil {
			return e
		}
	}
	// Backfill explicit references from JSON structure, including migrated drafts.
	rows, e := tx.Query(ctx, "SELECT id,state,document FROM content")
	if e != nil {
		return e
	}
	type row struct {
		id, state string
		doc       map[string]any
	}
	var all []row
	for rows.Next() {
		var v row
		if e = rows.Scan(&v.id, &v.state, &v.doc); e != nil {
			return e
		}
		all = append(all, v)
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return e
	}
	for _, v := range all {
		if e = syncRefs(ctx, tx, v.id, v.state, v.doc); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	s.routes(m)
	s.identityRoutes(m)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				slog.Error("panic", "error", e)
				fail(w, errors.New("internal error"))
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		origin := r.Header.Get("Origin")
		allowed := false
		for _, o := range s.C.Origins {
			allowed = allowed || o == origin
		}
		allowed = allowed || origin == s.C.Issuer || origin == s.C.APIURL
		// Registered OAuth browser origins get token endpoints only, never cookie-based APIs.
		if !allowed && origin != "" && (r.URL.Path == "/oauth/token" || r.URL.Path == "/oauth/revoke" || r.URL.Path == "/oauth/userinfo") {
			_ = s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM oauth_clients WHERE NOT disabled AND origins ? $1)", origin).Scan(&allowed)
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}
		if r.Method == "OPTIONS" {
			if !allowed {
				w.WriteHeader(403)
			} else {
				w.WriteHeader(204)
			}
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v1/") && r.Method != "GET" && r.Method != "HEAD" && !s.csrf(w, r) {
			return
		}
		if s.C.Production {
			expected := strings.TrimPrefix(strings.TrimPrefix(s.C.APIURL, "https://"), "http://")
			issuer := strings.TrimPrefix(strings.TrimPrefix(s.C.Issuer, "https://"), "http://")
			if r.Host != expected && r.Host != issuer {
				http.NotFound(w, r)
				return
			}
			if strings.HasPrefix(r.URL.Path, "/v1/") && r.Host != expected {
				http.NotFound(w, r)
				return
			}
		}
		if !strings.HasPrefix(r.URL.Path, "/v1/") && r.Method == "POST" {
			r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
		}
		m.ServeHTTP(w, r)
	})
}
