package routes_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"casadrop/internal/auth"
	"casadrop/internal/handlers"
	"casadrop/internal/middleware"
	"casadrop/internal/routes"
	"casadrop/internal/storage"
)

// TestAuthFlow is an end-to-end smoke test of the login → session → protected
// endpoint → logout flow. It wires the real router (internal/routes) against
// a real *middleware.AdminAuth and a real *storage.Storage on a tmpdir, then
// drives it through an httptest.Server.
//
// The goal isn't to cover every handler — it's to make sure middleware, router
// wiring, cookie-based sessions, and role checks hang together. If this passes,
// a regression in any of the glue layers would break it immediately.
func TestAuthFlow(t *testing.T) {
	srv, cleanup := newTestServer(t, "integration-test-pass-123!")
	defer cleanup()

	client := newClientWithJar(t, srv)

	// --- 1. /api/auth/status is public and reports setup=false when env password is set
	res := do(t, client, http.MethodGet, srv.URL+"/api/auth/status", nil, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("auth status: want 200, got %d", res.StatusCode)
	}
	status := map[string]any{}
	decode(t, res, &status)
	if status["authenticated"].(bool) {
		t.Errorf("auth status before login: want authenticated=false, got true")
	}

	// --- 2. Protected endpoint without session → redirect or 401
	res = do(t, client, http.MethodGet, srv.URL+"/api/stats", nil, nil)
	if res.StatusCode == http.StatusOK {
		t.Errorf("/api/stats without session: want denial, got 200")
	}

	// --- 3. JSON login with wrong password → 401
	body := bytes.NewBufferString(`{"password":"wrong"}`)
	res = do(t, client, http.MethodPost, srv.URL+"/login", body, map[string]string{
		"Content-Type": "application/json",
	})
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("bad login: want 401, got %d", res.StatusCode)
	}

	// --- 4. JSON login with right password → 200 + session cookie
	body = bytes.NewBufferString(`{"password":"integration-test-pass-123!"}`)
	res = do(t, client, http.MethodPost, srv.URL+"/login", body, map[string]string{
		"Content-Type": "application/json",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("good login: want 200, got %d", res.StatusCode)
	}
	if !hasCookie(client, srv, "casadrop_session") {
		t.Fatal("login succeeded but no session cookie was stored")
	}

	// --- 5. Protected endpoint with session → 200
	res = do(t, client, http.MethodGet, srv.URL+"/api/stats", nil, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("/api/stats with session: want 200, got %d", res.StatusCode)
	}

	// --- 6. Auth status reflects logged-in state
	res = do(t, client, http.MethodGet, srv.URL+"/api/auth/status", nil, nil)
	decode(t, res, &status)
	if !status["authenticated"].(bool) {
		t.Errorf("auth status after login: want authenticated=true, got false")
	}

	// --- 7. Logout (JSON) → session invalidated
	res = do(t, client, http.MethodPost, srv.URL+"/logout", nil, map[string]string{
		"Accept": "application/json",
	})
	if res.StatusCode != http.StatusOK {
		t.Errorf("logout: want 200, got %d", res.StatusCode)
	}

	// --- 8. After logout the protected endpoint is denied again
	res = do(t, client, http.MethodGet, srv.URL+"/api/stats", nil, nil)
	if res.StatusCode == http.StatusOK {
		t.Errorf("/api/stats after logout: want denial, got 200")
	}
}

// ---------------- helpers ----------------

func newTestServer(t *testing.T, envPassword string) (*httptest.Server, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "casadrop-routes-test-*")
	if err != nil {
		t.Fatalf("tmpdir: %v", err)
	}
	// Required subdirs & minimal templates the handlers package loads.
	must(t, os.MkdirAll(filepath.Join(tmpDir, "uploads"), 0755))
	templatesDir := filepath.Join(tmpDir, "templates")
	must(t, os.MkdirAll(templatesDir, 0755))
	for _, name := range []string{"index.html", "share.html", "folder.html", "receive.html", "login.html", "setup.html", "error.html"} {
		must(t, os.WriteFile(filepath.Join(templatesDir, name), []byte(`<html><body>{{.}}</body></html>`), 0644))
	}

	// webhook.Service reads DATA_DIR when constructing the handler
	t.Setenv("DATA_DIR", tmpDir)

	store, err := storage.New(tmpDir)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}

	h, err := handlers.New(store, templatesDir)
	if err != nil {
		store.Close()
		t.Fatalf("handlers: %v", err)
	}

	adminAuth := middleware.NewAdminAuth(envPassword, tmpDir)
	oidcProvider, _ := auth.NewProvider(tmpDir)
	oidcHandlers := auth.NewHandlers(oidcProvider, adminAuth)
	oidcHandlers.SetUserService(auth.NewUserService(store))

	emailHandler := handlers.NewEmailHandler(store)
	h.SetEmailHandler(emailHandler)

	downloadLimiter := middleware.NewRateLimiter(1000, time.Minute)

	router := routes.New(routes.Deps{
		Handler:         h,
		AdminAuth:       adminAuth,
		OIDC:            oidcHandlers,
		EmailHandler:    emailHandler,
		DownloadLimiter: downloadLimiter,
		StaticDir:       tmpDir,
	})

	srv := httptest.NewServer(router)

	cleanup := func() {
		srv.Close()
		adminAuth.Stop()
		downloadLimiter.Stop()
		emailHandler.Stop()
		if oidcProvider != nil {
			oidcProvider.Stop()
		}
		store.Close()
		os.RemoveAll(tmpDir)
	}
	return srv, cleanup
}

func newClientWithJar(t *testing.T, _ *httptest.Server) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	return &http.Client{
		Jar: jar,
		// Don't auto-follow redirects — we want to observe them.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func do(t *testing.T, c *http.Client, method, url string, body io.Reader, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := c.Do(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, url, err)
	}
	return res
}

func decode(t *testing.T, res *http.Response, out any) {
	t.Helper()
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func hasCookie(c *http.Client, srv *httptest.Server, name string) bool {
	u, err := url.Parse(srv.URL)
	if err != nil {
		return false
	}
	for _, ck := range c.Jar.Cookies(u) {
		if ck.Name == name && ck.Value != "" {
			return true
		}
	}
	return false
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
