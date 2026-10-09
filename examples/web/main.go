// A separate confidential web application using Chloe ID. Tokens stay server-side.
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type flow struct {
	Verifier string
	Expires  time.Time
}
type account struct {
	Name, Email string
	Expires     time.Time
}

func random() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func main() {
	issuer := strings.TrimSuffix(os.Getenv("ISSUER_URL"), "/")
	clientID, secret := os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET")
	if issuer == "" || clientID == "" || secret == "" {
		log.Fatal("Set ISSUER_URL, CLIENT_ID and CLIENT_SECRET")
	}
	parsed, e := url.Parse(issuer)
	if e != nil || (parsed.Scheme != "https" && parsed.Hostname() != "localhost") {
		log.Fatal("Issuer must use HTTPS or localhost")
	}
	callback := "http://localhost:4000/callback"
	var mu sync.Mutex
	flows := map[string]flow{}
	sessions := map[string]account{}
	client := &http.Client{Timeout: 10 * time.Second}
	page := template.Must(template.New("home").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Another project · Chloe ID example</title><body style="background:#171917;color:#f1f2e9;font:18px system-ui;max-width:640px;margin:10vh auto;padding:24px"><h1>Another project.</h1>{{if .Name}}<p>Signed in as {{.Name}} ({{.Email}}).</p><p>This application has its own server-side session. Chloe ID provided the sign-in.</p>{{else}}<p>A second website using your identity platform.</p><a style="color:#c5ef78" href="/login">Sign in with Chloe ID</a>{{end}}</body></html>`))
	m := http.NewServeMux()
	m.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		var a account
		if c, e := r.Cookie("example_session"); e == nil {
			mu.Lock()
			a = sessions[c.Value]
			if a.Expires.Before(time.Now()) {
				delete(sessions, c.Value)
				a = account{}
			}
			mu.Unlock()
		}
		w.Header().Set("Cache-Control", "no-store")
		_ = page.Execute(w, a)
	})
	m.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		state, verifier := random(), random()
		mu.Lock()
		for k, v := range flows {
			if v.Expires.Before(time.Now()) {
				delete(flows, k)
			}
		}
		flows[state] = flow{verifier, time.Now().Add(5 * time.Minute)}
		mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "example_flow", Value: state, HttpOnly: true, Path: "/", SameSite: http.SameSiteLaxMode, MaxAge: 300})
		sum := sha256.Sum256([]byte(verifier))
		q := url.Values{"client_id": {clientID}, "redirect_uri": {callback}, "response_type": {"code"}, "scope": {"openid profile email"}, "state": {state}, "nonce": {random()}, "code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"}}
		http.Redirect(w, r, issuer+"/oauth/authorize?"+q.Encode(), 303)
	})
	m.HandleFunc("GET /callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		cookie, e := r.Cookie("example_flow")
		if e != nil || state == "" || cookie.Value != state {
			http.Error(w, "Sign-in did not match this browser", 400)
			return
		}
		mu.Lock()
		f, ok := flows[state]
		delete(flows, state)
		mu.Unlock()
		if !ok || f.Expires.Before(time.Now()) {
			http.Error(w, "Sign-in expired", 400)
			return
		}
		if r.URL.Query().Get("error") != "" {
			http.Error(w, "Sign-in was cancelled. Return to the application and try again.", 400)
			return
		}
		v := url.Values{"grant_type": {"authorization_code"}, "code": {r.URL.Query().Get("code")}, "redirect_uri": {callback}, "code_verifier": {f.Verifier}}
		req, _ := http.NewRequest("POST", issuer+"/oauth/token", strings.NewReader(v.Encode()))
		req.SetBasicAuth(clientID, secret)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		res, e := client.Do(req)
		if e != nil {
			http.Error(w, "Identity server unavailable", 502)
			return
		}
		defer res.Body.Close()
		var tokens struct {
			AccessToken string `json:"access_token"`
		}
		if res.StatusCode != 200 || json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&tokens) != nil || tokens.AccessToken == "" {
			http.Error(w, "Token exchange failed", 400)
			return
		}
		req, _ = http.NewRequest("GET", issuer+"/oauth/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		res, e = client.Do(req)
		if e != nil {
			http.Error(w, "Profile unavailable", 502)
			return
		}
		defer res.Body.Close()
		var user struct{ Sub, Name, Email string }
		if res.StatusCode != 200 || json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&user) != nil || user.Sub == "" {
			http.Error(w, "Profile could not be verified", 400)
			return
		}
		session := random()
		mu.Lock()
		sessions[session] = account{user.Name, user.Email, time.Now().Add(time.Hour)}
		mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "example_session", Value: session, HttpOnly: true, Path: "/", SameSite: http.SameSiteLaxMode, MaxAge: 3600})
		http.SetCookie(w, &http.Cookie{Name: "example_flow", Value: "", HttpOnly: true, Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/", 303)
	})
	fmt.Println("Web example: http://localhost:4000")
	log.Fatal((&http.Server{Addr: "127.0.0.1:4000", Handler: m, ReadHeaderTimeout: 5 * time.Second}).ListenAndServe())
}
