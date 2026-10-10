package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (s *Server) socialConfig(provider string) (*oauth2.Config, error) {
	c := &oauth2.Config{RedirectURL: s.C.Issuer + "/social/" + provider + "/callback"}
	switch provider {
	case "google":
		c.ClientID = s.C.GoogleID
		c.ClientSecret = s.C.GoogleSecret
		c.Endpoint = google.Endpoint
		c.Scopes = []string{"openid", "email", "profile"}
	case "github":
		c.ClientID = s.C.GitHubID
		c.ClientSecret = s.C.GitHubSecret
		c.Endpoint = github.Endpoint
		c.Scopes = []string{"read:user", "user:email"}
	default:
		return nil, bad("Unknown sign-in provider.")
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		return nil, problem{503, "This sign-in provider has not been configured."}
	}
	return c, nil
}
func (s *Server) socialStart(w http.ResponseWriter, r *http.Request) {
	s.beginSocial(w, r, r.PathValue("provider"), "")
}
func (s *Server) beginSocial(w http.ResponseWriter, r *http.Request, provider, linkUID string) {
	c, e := s.socialConfig(provider)
	if e != nil {
		fail(w, e)
		return
	}
	if e = s.limit(r.Context(), "social:"+s.ip(r), 30, 15*time.Minute); e != nil {
		fail(w, e)
		return
	}
	binding := token()
	verifier := oauth2.GenerateVerifier()
	state, e := s.newToken(r.Context(), "social", "", map[string]any{"provider": provider, "binding": hash(binding), "verifier": verifier, "flow": r.FormValue("flow"), "continue": r.FormValue("continue"), "linkUID": linkUID}, 10*time.Minute)
	if e != nil {
		fail(w, e)
		return
	}
	s.cookie(w, "chloe_social", binding, 600)
	http.Redirect(w, r, c.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)), 303)
}
func fetchSocial(ctx context.Context, c *http.Client, address string, out any) error {
	req, e := http.NewRequestWithContext(ctx, "GET", address, nil)
	if e != nil {
		return e
	}
	req.Header.Set("Accept", "application/json")
	resp, e := c.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("provider returned %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}
func (s *Server) socialCallback(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	_, p, e := s.consume(r.Context(), "social", r.URL.Query().Get("state"))
	if e != nil {
		fail(w, e)
		return
	}
	if str(p, "provider") != provider || !equal(str(p, "binding"), hash(s.cookieValue(r, "chloe_social"))) {
		fail(w, problem{403, "Sign-in did not match this browser."})
		return
	}
	s.cookie(w, "chloe_social", "", -1)
	if r.URL.Query().Get("error") != "" {
		s.page(w, r, "Sign-in cancelled.", `<p>No account changes were made.</p><a href="/login">Try again</a>`, nil)
		return
	}
	c, e := s.socialConfig(provider)
	if e != nil {
		fail(w, e)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	t, e := c.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(str(p, "verifier")))
	if e != nil {
		s.page(w, r, "Sign-in failed.", `<p>The provider could not complete sign-in. <a href="/login">Try again</a>.</p>`, nil)
		return
	}
	client := c.Client(ctx, t)
	var subject, email, name string
	verified := false
	if provider == "google" {
		var info struct {
			Sub      string `json:"sub"`
			Email    string `json:"email"`
			Name     string `json:"name"`
			Verified bool   `json:"email_verified"`
		}
		e = fetchSocial(ctx, client, "https://openidconnect.googleapis.com/v1/userinfo", &info)
		subject, email, name, verified = info.Sub, info.Email, info.Name, info.Verified
	} else {
		var info struct {
			ID    int64  `json:"id"`
			Name  string `json:"name"`
			Login string `json:"login"`
		}
		e = fetchSocial(ctx, client, "https://api.github.com/user", &info)
		if e == nil {
			subject = strconv.FormatInt(info.ID, 10)
			name = info.Name
			if name == "" {
				name = info.Login
			}
			var emails []struct {
				Email    string `json:"email"`
				Verified bool   `json:"verified"`
				Primary  bool   `json:"primary"`
			}
			e = fetchSocial(ctx, client, "https://api.github.com/user/emails", &emails)
			for _, v := range emails {
				if v.Primary && v.Verified {
					email = v.Email
					verified = true
				}
			}
		}
	}
	if e != nil || subject == "" || subject == "0" {
		fail(w, bad("The provider did not return a valid identity."))
		return
	}
	email = strings.ToLower(email)
	if linkUID := str(p, "linkUID"); linkUID != "" {
		u := s.identityUser(w, r)
		if u == nil {
			return
		}
		if u.ID != linkUID || !u.authenticated() {
			fail(w, problem{403, "Sign in to the original account before linking."})
			return
		}
		res, e := s.DB.Exec(ctx, "INSERT INTO social_accounts(provider,subject,account_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING", provider, subject, u.ID)
		if e != nil {
			fail(w, e)
			return
		}
		if res.RowsAffected() == 0 {
			s.page(w, r, "Could not link account.", `<p>This provider is already connected to an account. Sign in with the existing account to manage it.</p><a href="/account">Back to account</a>`, nil)
			return
		}
		s.audit(ctx, u.ID, "provider.link", provider)
		http.Redirect(w, r, "/account", 303)
		return
	}
	var u User
	e = s.DB.QueryRow(ctx, "SELECT a.id,a.name,a.email,a.verified,a.owner FROM accounts a JOIN social_accounts p ON p.account_id=a.id WHERE p.provider=$1 AND p.subject=$2", provider, subject).Scan(&u.ID, &u.Name, &u.Email, &u.Verified, &u.Owner)
	if e != nil {
		if !errors.Is(e, pgx.ErrNoRows) {
			fail(w, e)
			return
		}
		if !s.C.Registration || !verified || !validEmail(email) {
			s.page(w, r, "Account setup required.", `<p>New social accounts require open registration and a verified provider email. If you already have an account, sign in and link this provider from your account page.</p><a href="/login">Back to sign in</a>`, nil)
			return
		}
		tx, err := s.DB.Begin(ctx)
		if err != nil {
			fail(w, err)
			return
		}
		defer tx.Rollback(ctx)
		u = User{ID: id(), Name: name, Email: email, Verified: true}
		res, err := tx.Exec(ctx, "INSERT INTO accounts(id,email,name,verified) VALUES($1,$2,$3,true) ON CONFLICT(email) DO NOTHING", u.ID, email, name)
		if err != nil {
			fail(w, err)
			return
		}
		if res.RowsAffected() == 0 {
			s.page(w, r, "Connect your existing account.", `<p>Sign in using your existing account, then explicitly link this provider from your account page.</p><a href="/login">Sign in</a>`, nil)
			return
		}
		if _, err = tx.Exec(ctx, "INSERT INTO social_accounts VALUES($1,$2,$3)", provider, subject, u.ID); err == nil {
			err = tx.Commit(ctx)
		}
		if err != nil {
			fail(w, err)
			return
		}
	}
	r.Form = url.Values{"flow": {str(p, "flow")}, "continue": {str(p, "continue")}}
	s.completeIdentity(w, r, &u)
}
