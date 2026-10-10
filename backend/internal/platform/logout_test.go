package platform

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/publicsuffix"
)

type logoutTransport struct{ handler http.Handler }

// Exercise production host routing and Secure, host-only cookie jars without
// depending on public DNS or a local TLS certificate for the two test hosts.
func (tr logoutTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	tr.handler.ServeHTTP(rec, r)
	res := rec.Result()
	res.Request = r
	return res, nil
}

func logoutBrowser(t *testing.T, h *harness) *http.Client {
	t.Helper()
	jar, e := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if e != nil {
		t.Fatal(e)
	}
	return &http.Client{Jar: jar, Transport: logoutTransport{h.s.Handler()}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func logoutRequest(t *testing.T, client *http.Client, method, target, origin string, form url.Values) *http.Response {
	t.Helper()
	r, e := http.NewRequest(method, target, strings.NewReader(form.Encode()))
	if e != nil {
		t.Fatal(e)
	}
	if method == "POST" {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", origin)
	}
	res, e := client.Do(r)
	if e != nil {
		t.Fatal(e)
	}
	return res
}

func logoutRedirect(t *testing.T, res *http.Response) string {
	t.Helper()
	target := res.Header.Get("Location")
	if res.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("logout redirect can be cached")
	}
	status(t, res, http.StatusSeeOther)
	if target == "" {
		t.Fatal("missing logout redirect")
	}
	return target
}

func setupLogout(t *testing.T) *harness {
	t.Helper()
	h := setup(t)
	h.s.C.Production = true
	h.s.C.APIURL = "https://api.example.test"
	h.s.C.Issuer = "https://auth.example.test"
	h.s.C.SiteURL = "https://www.example.test"
	h.s.C.Origins = []string{h.s.C.SiteURL}
	return h
}

func seedLogoutSession(t *testing.T, h *harness, client *http.Client, kind string) *http.Cookie {
	t.Helper()
	origin := h.s.C.APIURL
	if kind == "identity" {
		origin = h.s.C.Issuer
	}
	r := httptest.NewRequest("GET", origin, nil)
	w := httptest.NewRecorder()
	if e := h.s.newSession(w, r, h.owner, kind, true); e != nil {
		t.Fatal(e)
	}
	u, _ := url.Parse(origin)
	cookies := w.Result().Cookies()
	client.Jar.SetCookies(u, cookies)
	if len(cookies) != 1 || !cookies[0].Secure || cookies[0].Domain != "" {
		t.Fatal("expected one Secure host-only session cookie")
	}
	return cookies[0]
}

func logoutForm(t *testing.T, h *harness, client *http.Client) (string, url.Values) {
	t.Helper()
	start := logoutRedirect(t, logoutRequest(t, client, "GET", h.s.C.Issuer+"/logout", "", nil))
	if start != h.s.C.APIURL+"/v1/session/logout/start" {
		t.Fatal("logout did not establish the API-host binding")
	}
	target := logoutRedirect(t, logoutRequest(t, client, "GET", start, "", nil))
	u, _ := url.Parse(target)
	res := logoutRequest(t, client, "GET", target, "", nil)
	if res.StatusCode != 200 {
		t.Fatalf("logout confirmation status %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	match := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindSubmatch(body)
	if len(match) != 2 || u.Query().Get("flow") == "" {
		t.Fatal("missing confirmation tokens")
	}
	return target, url.Values{"csrf": {string(match[1])}, "flow": {u.Query().Get("flow")}}
}

func TestChloeIDLogoutAcrossHosts(t *testing.T) {
	h := setupLogout(t)
	browser, otherDevice := logoutBrowser(t, h), logoutBrowser(t, h)
	apiCookie := seedLogoutSession(t, h, browser, "api")
	idCookie := seedLogoutSession(t, h, browser, "identity")
	seedLogoutSession(t, h, otherDevice, "api")
	seedLogoutSession(t, h, otherDevice, "identity")
	apiURL, _ := url.Parse(h.s.C.APIURL)
	for _, cookie := range browser.Jar.Cookies(apiURL) {
		if cookie.Name == idCookie.Name {
			t.Fatal("identity cookie leaked to the API host")
		}
	}
	target, form := logoutForm(t, h, browser)
	// Opening the confirmation is read-only, including for the website session.
	status(t, logoutRequest(t, browser, "GET", h.s.C.APIURL+"/v1/admin/content", "", nil), 200)
	status(t, logoutRequest(t, browser, "GET", h.s.C.Issuer+"/account", "", nil), 200)
	callback := logoutRedirect(t, logoutRequest(t, browser, "POST", target, h.s.C.Issuer, form))
	if !strings.HasPrefix(callback, h.s.C.APIURL+"/v1/session/logout/callback?ticket=") {
		t.Fatal("identity logout did not return to the API host")
	}
	// A copied ticket cannot log another browser out or consume its owner's ticket.
	status(t, logoutRequest(t, otherDevice, "GET", callback, "", nil), 403)
	if logoutRedirect(t, logoutRequest(t, browser, "GET", callback, "", nil)) != h.s.C.SiteURL {
		t.Fatal("logout did not finish at the website")
	}
	status(t, logoutRequest(t, browser, "GET", callback, "", nil), 400)
	status(t, logoutRequest(t, browser, "POST", target, h.s.C.Issuer, form), 400)
	status(t, logoutRequest(t, browser, "GET", h.s.C.APIURL+"/v1/admin/content", "", nil), 401)
	status(t, logoutRequest(t, browser, "GET", h.s.C.Issuer+"/login", "", nil), 200)
	identityURL, _ := url.Parse(h.s.C.Issuer)
	for _, pair := range []struct {
		url  *url.URL
		name string
	}{{apiURL, apiCookie.Name}, {identityURL, idCookie.Name}} {
		for _, cookie := range browser.Jar.Cookies(pair.url) {
			if cookie.Name == pair.name {
				t.Fatal("session cookie survived logout")
			}
		}
	}
	// Revocation must survive a reload and replay of the old cookies.
	browser.Jar.SetCookies(apiURL, []*http.Cookie{apiCookie})
	browser.Jar.SetCookies(identityURL, []*http.Cookie{idCookie})
	status(t, logoutRequest(t, browser, "GET", h.s.C.APIURL+"/v1/admin/content", "", nil), 401)
	status(t, logoutRequest(t, browser, "GET", h.s.C.Issuer+"/login", "", nil), 200)
	status(t, logoutRequest(t, otherDevice, "GET", h.s.C.APIURL+"/v1/admin/content", "", nil), 200)
	status(t, logoutRequest(t, otherDevice, "GET", h.s.C.Issuer+"/account", "", nil), 200)
}

func TestChloeIDLogoutRequiresConfirmation(t *testing.T) {
	h := setupLogout(t)
	browser := logoutBrowser(t, h)
	seedLogoutSession(t, h, browser, "api")
	seedLogoutSession(t, h, browser, "identity")
	target, form := logoutForm(t, h, browser)
	status(t, logoutRequest(t, browser, "POST", target, "https://evil.example", form), 403)
	status(t, logoutRequest(t, browser, "POST", target, h.s.C.Issuer, url.Values{"flow": form["flow"]}), 403)
	_, e := h.s.DB.Exec(t.Context(), "UPDATE one_time_tokens SET expires_at=now()-interval '1 second' WHERE purpose='logout_flow'")
	if e != nil {
		t.Fatal(e)
	}
	status(t, logoutRequest(t, browser, "POST", target, h.s.C.Issuer, form), 400)
	status(t, logoutRequest(t, browser, "GET", h.s.C.APIURL+"/v1/admin/content", "", nil), 200)
	status(t, logoutRequest(t, browser, "GET", h.s.C.Issuer+"/account", "", nil), 200)
	// An expired identity session must not prevent ending an existing website session.
	_, e = h.s.DB.Exec(t.Context(), "DELETE FROM sessions WHERE kind='identity'")
	if e != nil {
		t.Fatal(e)
	}
	target, form = logoutForm(t, h, browser)
	callback := logoutRedirect(t, logoutRequest(t, browser, "POST", target, h.s.C.Issuer, form))
	logoutRedirect(t, logoutRequest(t, browser, "GET", callback, "", nil))
	status(t, logoutRequest(t, browser, "GET", h.s.C.APIURL+"/v1/admin/content", "", nil), 401)
}
