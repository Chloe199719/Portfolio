package platform

import (
	"encoding/json"
	"github.com/ory/fosite"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/url"
	"strings"
)

func validRedirect(v string, public, production bool) bool {
	u, e := url.Parse(v)
	if e != nil || u.Scheme == "" || u.Fragment != "" || u.User != nil || strings.Contains(v, "*") {
		return false
	}
	if u.Scheme == "https" {
		return u.Host != ""
	}
	if u.Scheme == "http" {
		return !production && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")
	}
	return public && strings.Contains(u.Scheme, ".") && u.Host == "" && strings.HasPrefix(u.Path, "/")
}
func (s *Server) applications(w http.ResponseWriter, r *http.Request) {
	u := s.require(w, r, true)
	if u == nil {
		return
	}
	ctx := r.Context()
	if r.Method == "POST" {
		var in struct {
			Name         string   `json:"name"`
			Public       bool     `json:"public"`
			RedirectURIs []string `json:"redirectUris"`
			Origins      []string `json:"origins"`
			Scopes       []string `json:"scopes"`
		}
		if e := decode(w, r, &in); e != nil {
			fail(w, e)
			return
		}
		if len(strings.TrimSpace(in.Name)) < 1 || len(in.Name) > 100 || len(in.RedirectURIs) < 1 || len(in.RedirectURIs) > 20 || len(in.Origins) > 20 {
			fail(w, bad("Add a name and between 1 and 20 exact redirect URIs."))
			return
		}
		for _, v := range in.RedirectURIs {
			if !validRedirect(v, in.Public, s.C.Production) {
				fail(w, bad("Invalid callback: "+v))
				return
			}
		}
		for _, v := range in.Origins {
			parsed, e := url.Parse(v)
			if e != nil || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || !validRedirect(v, false, s.C.Production) {
				fail(w, bad("Invalid allowed origin."))
				return
			}
		}
		for _, v := range in.Scopes {
			if v != "openid" && v != "profile" && v != "email" && v != "offline_access" {
				fail(w, bad("Unsupported scope."))
				return
			}
		}
		client := &fosite.DefaultOpenIDConnectClient{DefaultClient: &fosite.DefaultClient{ID: id(), Public: in.Public, RedirectURIs: in.RedirectURIs, Scopes: in.Scopes, GrantTypes: []string{"authorization_code", "refresh_token"}, ResponseTypes: []string{"code"}}, TokenEndpointAuthMethod: "none"}
		secret := ""
		if !in.Public {
			secret = token()
			hashed, e := bcrypt.GenerateFromPassword([]byte(secret), 12)
			if e != nil {
				fail(w, e)
				return
			}
			client.Secret = hashed
			client.TokenEndpointAuthMethod = "client_secret_basic"
		}
		raw, _ := json.Marshal(client)
		origins, _ := json.Marshal(in.Origins)
		if _, e := s.DB.Exec(ctx, "INSERT INTO oauth_clients(id,name,document,origins) VALUES($1,$2,$3,$4)", client.ID, in.Name, raw, origins); e != nil {
			fail(w, e)
			return
		}
		s.audit(ctx, u.ID, "application.create", client.ID)
		send(w, 201, map[string]any{"id": client.ID, "secret": secret})
		return
	}
	if r.Method == "PATCH" {
		key := r.PathValue("id")
		var in struct {
			Action string `json:"action"`
		}
		if e := decode(w, r, &in); e != nil {
			fail(w, e)
			return
		}
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			fail(w, e)
			return
		}
		defer tx.Rollback(ctx)
		var c fosite.DefaultOpenIDConnectClient
		var raw []byte
		if e = tx.QueryRow(ctx, "SELECT document FROM oauth_clients WHERE id=$1 FOR UPDATE", key).Scan(&raw); e != nil {
			fail(w, problem{404, "Application not found."})
			return
		}
		if e = json.Unmarshal(raw, &c); e != nil {
			fail(w, e)
			return
		}
		secret := ""
		switch in.Action {
		case "rotate":
			if c.Public {
				fail(w, bad("Public clients do not have secrets."))
				return
			}
			secret = token()
			c.Secret, e = bcrypt.GenerateFromPassword([]byte(secret), 12)
			if e == nil {
				raw, _ = json.Marshal(c)
				_, e = tx.Exec(ctx, "UPDATE oauth_clients SET document=$2 WHERE id=$1", key, raw)
			}
		case "disable", "enable":
			_, e = tx.Exec(ctx, "UPDATE oauth_clients SET disabled=$2 WHERE id=$1", key, in.Action == "disable")
		default:
			e = bad("Unknown application action.")
		}
		if e == nil && in.Action != "enable" {
			_, e = tx.Exec(ctx, "UPDATE oauth_sessions SET active=false WHERE client_id=$1", key)
		}
		if e == nil {
			e = tx.Commit(ctx)
		}
		if e != nil {
			fail(w, e)
			return
		}
		s.audit(ctx, u.ID, "application."+in.Action, key)
		send(w, 200, map[string]any{"ok": true, "secret": secret})
		return
	}
	rows, e := s.DB.Query(ctx, "SELECT id,name,document - 'client_secret' - 'rotated_secrets',origins,disabled FROM oauth_clients ORDER BY created_at DESC")
	if e != nil {
		fail(w, e)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var key, name string
		var d Document
		var origins []string
		var disabled bool
		if e = rows.Scan(&key, &name, &d, &origins, &disabled); e != nil {
			fail(w, e)
			return
		}
		out = append(out, map[string]any{"id": key, "name": name, "client": d, "origins": origins, "disabled": disabled})
	}
	send(w, 200, map[string]any{"applications": out})
}
