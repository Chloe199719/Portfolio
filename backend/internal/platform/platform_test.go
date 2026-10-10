package platform

import (
	"bytes"
	assets "chloe.dev/home/backend"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	josejwt "github.com/go-jose/go-jose/v3/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type harness struct {
	s                    *Server
	http                 *httptest.Server
	client               *http.Client
	csrf, owner, visitor string
}

func setup(t *testing.T) *harness {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a database ending in _test")
	}
	parsed, e := url.Parse(dsn)
	if e != nil || !strings.HasSuffix(parsed.Path, "_test") {
		t.Fatal("integration database name must end in _test")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	schema := "test_" + strings.ReplaceAll(id(), "-", "")
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	q := parsed.Query()
	q.Set("search_path", schema)
	parsed.RawQuery = q.Encode()
	h := httptest.NewUnstartedServer(nil)
	origin := "http://" + strings.Replace(h.Listener.Addr().String(), "127.0.0.1", "localhost", 1)
	c := Config{DatabaseURL: parsed.String(), APIURL: origin, Issuer: origin, SiteURL: "http://localhost:3000", Origins: []string{"http://localhost:3000"}, DataDir: t.TempDir()}
	s, e := New(ctx, c, assets.Assets)
	if e != nil {
		t.Fatal(e)
	}
	h.Config.Handler = s.Handler()
	h.Start()
	h.URL = origin
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	x := &harness{s: s, http: h, client: client, owner: id(), visitor: id()}
	_, e = s.DB.Exec(ctx, "INSERT INTO accounts(id,email,name,verified,owner,totp_confirmed) VALUES($1,'owner@example.test','Owner',true,true,true),($2,'visitor@example.test','Visitor',true,false,false)", x.owner, x.visitor)
	if e != nil {
		t.Fatal(e)
	}
	var data map[string]any
	read(t, x.call(t, "GET", "/v1/session", nil), &data)
	x.csrf = str(data, "csrf")
	t.Cleanup(func() {
		h.Close()
		s.DB.Close()
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	return x
}
func (h *harness) call(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, e := json.Marshal(body)
		if e != nil {
			t.Fatal(e)
		}
		reader = bytes.NewReader(raw)
	}
	req, e := http.NewRequest(method, h.http.URL+path, reader)
	if e != nil {
		t.Fatal(e)
	}
	req.Header.Set("Origin", h.s.C.SiteURL)
	req.Header.Set("X-CSRF-Token", h.csrf)
	req.Header.Set("Content-Type", "application/json")
	res, e := h.client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	return res
}
func read(t *testing.T, res *http.Response, out any) {
	t.Helper()
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		t.Fatalf("HTTP %d: %s", res.StatusCode, b)
	}
	if out != nil {
		if e := json.Unmarshal(b, out); e != nil {
			t.Fatalf("status %d: %s", res.StatusCode, b)
		}
	}
}
func status(t *testing.T, res *http.Response, want int) {
	t.Helper()
	defer res.Body.Close()
	if res.StatusCode != want {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("want %d got %d: %s", want, res.StatusCode, b)
	}
}
func (h *harness) loginSession(t *testing.T, uid, kind string, mfa bool) {
	t.Helper()
	v := token()
	_, e := h.s.DB.Exec(context.Background(), "INSERT INTO sessions(hash,account_id,kind,mfa,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')", hash(v), uid, kind, mfa)
	if e != nil {
		t.Fatal(e)
	}
	u, _ := url.Parse(h.http.URL)
	h.client.Jar.SetCookies(u, []*http.Cookie{{Name: "chloe_" + kind, Value: v, Path: "/"}})
}
func (h *harness) form(t *testing.T, path string, values url.Values) *http.Response {
	t.Helper()
	values.Set("csrf", h.csrf)
	req, _ := http.NewRequest("POST", h.http.URL+path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", h.http.URL)
	res, e := h.client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	return res
}
func TestContentPrivacyRevisionsAndScheduling(t *testing.T) {
	h := setup(t)
	ctx := context.Background()
	status(t, h.call(t, "GET", "/v1/admin/content", nil), 401)
	h.loginSession(t, h.visitor, "api", false)
	status(t, h.call(t, "GET", "/v1/admin/content", nil), 403)
	h.loginSession(t, h.owner, "api", false)
	status(t, h.call(t, "GET", "/v1/admin/content", nil), 403)
	h.loginSession(t, h.owner, "api", true)
	doc := Document{"_type": "note", "title": "Private note", "summary": "A test description", "slug": map[string]any{"current": "private-note"}, "date": "2026-10-08", "body": []any{map[string]any{"_type": "block", "_key": "b1", "style": "normal", "children": []any{map[string]any{"_type": "span", "_key": "s1", "text": "Private test content", "marks": []any{}}}, "markDefs": []any{}}}}
	var result struct {
		Document Document `json:"document"`
	}
	read(t, h.call(t, "POST", "/v1/admin/content/note-test", ContentMutation{Action: "save", Document: doc}), &result)
	rev := str(result.Document, "_rev")
	if rev == "" {
		t.Fatal("save did not return revision")
	}
	var public struct {
		Documents []Document `json:"documents"`
	}
	read(t, h.call(t, "GET", "/v1/content", nil), &public)
	for _, d := range public.Documents {
		if str(d, "_id") == "note-test" {
			t.Fatal("draft leaked")
		}
	}
	status(t, h.call(t, "POST", "/v1/admin/content/note-test", ContentMutation{Action: "save", Revision: "stale", Document: doc}), 409)
	status(t, h.call(t, "POST", "/v1/admin/content/note-test/schedules", map[string]any{"revisionId": rev, "runAt": time.Now().Add(time.Hour)}), 200)
	doc["title"] = "Later unscheduled edits"
	read(t, h.call(t, "POST", "/v1/admin/content/note-test", ContentMutation{Action: "autosave", Revision: rev, Document: doc}), &result)
	later := str(result.Document, "_rev")
	if _, e := h.s.DB.Exec(ctx, "UPDATE schedules SET run_at=now()-interval '1 minute'"); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- h.s.publishDue(ctx) }()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	var title string
	e := h.s.DB.QueryRow(ctx, "SELECT document->>'title' FROM content WHERE id='note-test' AND state='published'").Scan(&title)
	if e != nil || title != "Private note" {
		t.Fatalf("schedule changed revision: %s %v", title, e)
	}
	editable, e := h.s.editable(ctx, "note-test")
	if e != nil || str(editable, "_rev") != later || str(editable, "title") != "Later unscheduled edits" {
		t.Fatal("schedule lost newer draft")
	}
	var count int
	_ = h.s.DB.QueryRow(ctx, "SELECT count(*) FROM schedules WHERE status='published'").Scan(&count)
	if count != 1 {
		t.Fatal("schedule did not finish exactly once")
	}
	status(t, h.call(t, "POST", "/v1/admin/content/note-test/organize", map[string]any{"action": "trash", "revision": later}), 200)
	status(t, h.call(t, "GET", "/v1/admin/content/note-test", nil), 404)
	status(t, h.call(t, "POST", "/v1/admin/content/note-test/organize", map[string]any{"action": "restore"}), 200)
}
func TestCSRFModerationAndContact(t *testing.T) {
	h := setup(t)
	req, _ := http.NewRequest("POST", h.http.URL+"/v1/contact", strings.NewReader(`{}`))
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("X-CSRF-Token", h.csrf)
	res, e := h.client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	status(t, res, 403)
	status(t, h.call(t, "POST", "/v1/contact", map[string]any{"name": "Visitor", "email": "visitor@example.test", "subject": "A question", "message": "A message with enough detail."}), 201)
	status(t, h.call(t, "GET", "/v1/admin/inbox", nil), 401)
	status(t, h.call(t, "POST", "/v1/guestbook", map[string]any{"name": "Visitor", "message": "Hello there"}), 401)
	h.loginSession(t, h.visitor, "api", false)
	status(t, h.call(t, "POST", "/v1/guestbook", map[string]any{"name": "Visitor", "message": "Hello there"}), 201)
	var key string
	_ = h.s.DB.QueryRow(context.Background(), "SELECT id FROM guestbook LIMIT 1").Scan(&key)
	status(t, h.call(t, "PATCH", "/v1/guestbook/"+key, map[string]any{"status": "approved"}), 403)
	h.loginSession(t, h.owner, "api", true)
	status(t, h.call(t, "PATCH", "/v1/guestbook/"+key, map[string]any{"status": "approved"}), 200)
	h.loginSession(t, h.visitor, "api", false)
	status(t, h.call(t, "DELETE", "/v1/guestbook/"+key, nil), 200)
}
func TestLocalAuthenticationAndOwnerMFA(t *testing.T) {
	h := setup(t)
	ctx := context.Background()
	pw, e := passwordHash("a sufficiently long password")
	if e != nil {
		t.Fatal(e)
	}
	key, e := totp.Generate(totp.GenerateOpts{Issuer: "Chloe test", AccountName: "owner@example.test"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = h.s.DB.Exec(ctx, "UPDATE accounts SET password_hash=$1,totp_secret=$2 WHERE id=$3", pw, key.Secret(), h.owner); e != nil {
		t.Fatal(e)
	}
	res := h.call(t, "GET", "/login", nil)
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if strings.Contains(string(body), `name="otp"`) {
		t.Fatal("login exposes MFA before identifying the account")
	}
	res = h.form(t, "/login", url.Values{"email": {"owner@example.test"}, "password": {"a sufficiently long password"}})
	if res.StatusCode != 303 || !strings.HasPrefix(res.Header.Get("Location"), "/step-up") {
		t.Fatal("password did not start the separate MFA step")
	}
	res.Body.Close()
	status(t, h.call(t, "GET", "/account", nil), 303)
	status(t, h.call(t, "GET", "/v1/admin/content", nil), 401)
	var lifetime float64
	if e := h.s.DB.QueryRow(ctx, "SELECT EXTRACT(EPOCH FROM expires_at-created_at) FROM sessions WHERE account_id=$1 AND kind='identity'", h.owner).Scan(&lifetime); e != nil || lifetime > 601 {
		t.Fatal("pending MFA session is not short-lived")
	}
	res = h.call(t, "GET", "/step-up", nil)
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if strings.Contains(string(body), "data:image/png") {
		t.Fatal("confirmed enrollment exposed its key")
	}
	status(t, h.form(t, "/step-up", url.Values{"otp": {"invalid"}}), 200)
	otp, _ := totp.GenerateCode(key.Secret(), time.Now())
	status(t, h.form(t, "/step-up", url.Values{"otp": {otp}}), 303)
	status(t, h.call(t, "GET", "/account", nil), 200)
	if h.s.checkTOTP(ctx, h.owner, key.Secret(), otp) {
		t.Fatal("MFA code was replayable")
	}
	h.loginSession(t, h.owner, "identity", false)
	res = h.call(t, "GET", "/step-up", nil)
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if strings.Contains(string(body), "data:image/png") || strings.Contains(string(body), key.Secret()) || !strings.Contains(string(body), "Two-step verification.") {
		t.Fatal("enrolled factor key was exposed")
	}
	_, _ = h.s.DB.Exec(ctx, "UPDATE sessions SET expires_at=now()-interval '1 minute'")
	status(t, h.call(t, "GET", "/account", nil), 303)
	reset, e := h.s.newToken(ctx, "reset", h.owner, map[string]any{}, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	status(t, h.form(t, "/reset", url.Values{"token": {reset}, "password": {"a different long password"}}), 200)
	if _, _, e = h.s.consume(ctx, "reset", reset); e == nil {
		t.Fatal("reset token reusable")
	}
}
func TestOAuthPKCERotationAndReplay(t *testing.T) {
	h := setup(t)
	h.loginSession(t, h.owner, "api", true)
	var app struct{ ID, Secret string }
	read(t, h.call(t, "POST", "/v1/admin/applications", map[string]any{"name": "Native test", "public": true, "redirectUris": []string{"dev.chloe.example:/callback"}, "origins": []string{}, "scopes": []string{"openid", "profile", "email", "offline_access"}}), &app)
	if app.ID == "" || app.Secret != "" {
		t.Fatal("public client received secret or missing ID")
	}
	h.loginSession(t, h.visitor, "identity", false)
	verifier := token()
	digest := sha256.Sum256([]byte(verifier))
	params := url.Values{"client_id": {app.ID}, "redirect_uri": {"dev.chloe.example:/callback"}, "response_type": {"code"}, "scope": {"openid profile email offline_access"}, "state": {token()}, "nonce": {token()}, "code_challenge": {base64.RawURLEncoding.EncodeToString(digest[:])}, "code_challenge_method": {"S256"}}
	authorize := func() string {
		res := h.form(t, "/oauth/authorize?"+params.Encode(), url.Values{"decision": {"allow"}})
		defer res.Body.Close()
		if res.StatusCode != 303 && res.StatusCode != 302 {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("authorize: %d %s", res.StatusCode, b)
		}
		u, e := url.Parse(res.Header.Get("Location"))
		if e != nil {
			t.Fatal(e)
		}
		if u.Query().Get("error") != "" {
			t.Fatalf("authorize: %s", u.String())
		}
		return u.Query().Get("code")
	}
	exchange := func(v url.Values) map[string]any {
		res := h.form(t, "/oauth/token", v)
		if res.StatusCode != 200 {
			b, _ := io.ReadAll(res.Body)
			res.Body.Close()
			t.Fatalf("token: %d %s", res.StatusCode, b)
		}
		out := map[string]any{}
		read(t, res, &out)
		return out
	}
	code := authorize()
	values := url.Values{"grant_type": {"authorization_code"}, "client_id": {app.ID}, "redirect_uri": {"dev.chloe.example:/callback"}, "code": {code}, "code_verifier": {verifier}}
	tokens := exchange(values)
	if str(tokens, "access_token") == "" || str(tokens, "id_token") == "" || str(tokens, "refresh_token") == "" {
		t.Fatalf("missing tokens: %v", tokens)
	}
	status(t, h.form(t, "/oauth/token", values), 400)
	values.Set("code", authorize())
	tokens = exchange(values)
	refresh := str(tokens, "refresh_token")
	rotated := exchange(url.Values{"grant_type": {"refresh_token"}, "client_id": {app.ID}, "refresh_token": {refresh}})
	if str(rotated, "refresh_token") == refresh {
		t.Fatal("refresh token did not rotate")
	}
	status(t, h.form(t, "/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {app.ID}, "refresh_token": {refresh}}), 400)
	status(t, h.form(t, "/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {app.ID}, "refresh_token": {str(rotated, "refresh_token")}}), 400)
	signed, e := josejwt.ParseSigned(str(tokens, "id_token"))
	if e != nil {
		t.Fatal(e)
	}
	oldID := h.s.Keys.CurrentID()
	if e = h.s.Keys.Rotate(); e != nil {
		t.Fatal(e)
	}
	var claims map[string]any
	matched := false
	for _, k := range h.s.Keys.Public().Keys {
		if k.KeyID == oldID {
			if e = signed.Claims(k.Key, &claims); e != nil {
				t.Fatal(e)
			}
			matched = true
		}
	}
	if !matched || claims["iss"] != h.s.C.Issuer || h.s.Keys.CurrentID() == oldID {
		t.Fatal("rotation failed")
	}
	params.Set("code_challenge_method", "plain")
	res := h.form(t, "/oauth/authorize?"+params.Encode(), url.Values{"decision": {"allow"}})
	if !strings.Contains(res.Header.Get("Location"), "error=") && res.StatusCode < 400 {
		t.Fatal("plain PKCE accepted")
	}
	res.Body.Close()
}
func TestValidationAndRedirects(t *testing.T) {
	for _, v := range []string{"https://example.com/*", "https://example.com/callback#fragment", "javascript:alert(1)", "http://remote.example/callback"} {
		if validRedirect(v, true, true) {
			t.Errorf("accepted %s", v)
		}
	}
	if !validRedirect("dev.chloe.app:/callback", true, true) || validRedirect("dev.chloe.app:/callback", false, true) {
		t.Fatal("native client type mismatch")
	}
	if validateDocument(Document{"_type": "note", "title": "x", "slug": map[string]any{"current": "x"}}, true) == nil {
		t.Fatal("incomplete publication accepted")
	}
	if validateDocument(Document{"_type": "note", "linkToBuild": "javascript:alert(1)"}, false) == nil {
		t.Fatal("unsafe URL")
	}
}

func TestMediaPrivacyNormalizationAndExplicitReferences(t *testing.T) {
	h := setup(t)
	h.loginSession(t, h.owner, "api", true)
	var raw bytes.Buffer
	if e := jpeg.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil); e != nil {
		t.Fatal(e)
	}
	// An APP1 segment simulates embedded location metadata; decoding/re-encoding
	// must remove it, independently of the supplied filename and Content-Type.
	marker := []byte("Exif\x00\x00location-metadata-must-not-survive")
	input := append([]byte{0xff, 0xd8, 0xff, 0xe1, byte((len(marker) + 2) >> 8), byte(len(marker) + 2)}, marker...)
	input = append(input, raw.Bytes()[2:]...)
	var body bytes.Buffer
	multipartBody := multipart.NewWriter(&body)
	part, _ := multipartBody.CreateFormFile("file", "photo.jpg")
	_, _ = part.Write(input)
	_ = multipartBody.WriteField("alt", "A test image")
	_ = multipartBody.Close()
	req, _ := http.NewRequest("POST", h.http.URL+"/v1/admin/media", &body)
	req.Header.Set("Content-Type", multipartBody.FormDataContentType())
	req.Header.Set("Origin", h.s.C.SiteURL)
	req.Header.Set("X-CSRF-Token", h.csrf)
	res, e := h.client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	var uploaded Document
	read(t, res, &uploaded)
	asset := str(uploaded, "id")
	if asset == "" {
		t.Fatalf("upload failed: %v", uploaded)
	}
	res = h.call(t, "GET", "/media/"+asset, nil)
	data, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if bytes.Contains(data, marker) || res.StatusCode != 200 || res.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatal("image was not normalized/private")
	}
	anonymous := &http.Client{}
	get := func() *http.Response {
		r, e := anonymous.Get(h.http.URL + "/media/" + asset)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	status(t, get(), 404)
	doc := Document{"_type": "note", "title": "Asset references", "date": "2026-10-08", "summary": "Text mentions " + asset, "slug": map[string]any{"current": "asset-references"}, "body": []any{map[string]any{"_type": "block", "children": []any{map[string]any{"_type": "span", "text": "A test story"}}}}}
	var saved struct {
		Document Document `json:"document"`
	}
	read(t, h.call(t, "POST", "/v1/admin/content/media-test", ContentMutation{Action: "publish", Document: doc}), &saved)
	if str(saved.Document, "_rev") == "" {
		t.Fatal("publish failed")
	}
	status(t, get(), 404) // A filename in text is never publication authorization.
	doc["image"] = uploaded["image"]
	read(t, h.call(t, "POST", "/v1/admin/content/media-test", ContentMutation{Action: "publish", Revision: str(saved.Document, "_rev"), Document: doc}), &saved)
	status(t, get(), 200)
	read(t, h.call(t, "POST", "/v1/admin/content/media-test", ContentMutation{Action: "unpublish", Revision: str(saved.Document, "_rev"), Document: doc}), &saved)
	status(t, get(), 404)
}

func TestRegistrationRecoveryAndAccountIsolation(t *testing.T) {
	h := setup(t)
	h.s.C.Registration = true
	h.s.C.SMTPUser = "accounts@example.test"
	status(t, h.form(t, "/register", url.Values{"email": {"new@example.test"}, "name": {"New visitor"}, "password": {"a sufficiently long password"}}), 200)
	ctx := context.Background()
	var uid, mailBody string
	var owner, verified bool
	if e := h.s.DB.QueryRow(ctx, "SELECT id,owner,verified FROM accounts WHERE email='new@example.test'").Scan(&uid, &owner, &verified); e != nil {
		t.Fatal(e)
	}
	if owner || verified {
		t.Fatal("registration granted owner/verified access")
	}
	if e := h.s.DB.QueryRow(ctx, "SELECT body FROM email_jobs WHERE recipient='new@example.test'").Scan(&mailBody); e != nil {
		t.Fatal(e)
	}
	link := strings.Split(strings.Split(mailBody, "\n\n")[1], "\n")[0]
	u, e := url.Parse(link)
	if e != nil {
		t.Fatal(e)
	}
	status(t, h.form(t, "/verify", url.Values{"token": {u.Query().Get("token")}}), 200)
	_ = h.s.DB.QueryRow(ctx, "SELECT verified FROM accounts WHERE id=$1", uid).Scan(&verified)
	if !verified {
		t.Fatal("verification failed")
	}
	h.loginSession(t, h.visitor, "api", false)
	status(t, h.call(t, "GET", "/v1/account", nil), 200)
	h.loginSession(t, h.owner, "identity", false)
	res := h.call(t, "GET", "/account", nil)
	if res.StatusCode != 303 || !strings.HasPrefix(res.Header.Get("Location"), "/step-up") {
		t.Fatal("owner account bypassed MFA")
	}
	res.Body.Close()
	h.loginSession(t, h.owner, "api", true)
	var visitorSession string
	_ = h.s.DB.QueryRow(ctx, "SELECT hash FROM sessions WHERE account_id=$1 AND kind='api'", h.visitor).Scan(&visitorSession)
	status(t, h.call(t, "PATCH", "/v1/account", AccountMutation{Action: "revoke-session", ID: visitorSession}), 200)
	var exists bool
	_ = h.s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM sessions WHERE hash=$1)", visitorSession).Scan(&exists)
	if !exists {
		t.Fatal("revoked another account's session")
	}
	status(t, h.call(t, "HEAD", "/v1/health", nil), 200)
}
