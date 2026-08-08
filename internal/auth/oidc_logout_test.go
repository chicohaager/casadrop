package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestOIDCLogoutInvalidatesServerSession proves GET /auth/oidc/logout revokes
// the server-side session, not just the cookie. Before the fix the handler
// only cleared the cookie: the bearer token stayed valid until its TTL, so a
// captured token was replayable after an "OIDC logout".
func TestOIDCLogoutInvalidatesServerSession(t *testing.T) {
	sc := &mockSessionCreator{}
	h := NewHandlers(nil, sc) // provider nil: logout must work regardless

	req := httptest.NewRequest("GET", "/auth/oidc/logout", nil)
	req.AddCookie(&http.Cookie{Name: "casadrop_session", Value: "raw-session-token"})
	rec := httptest.NewRecorder()

	h.LogoutHandler(rec, req)

	if sc.invalidatedWith != "raw-session-token" {
		t.Fatalf("server-side session was not invalidated: InvalidateSession got %q, want %q",
			sc.invalidatedWith, "raw-session-token")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect after logout, got %d", rec.Code)
	}
}

// TestOIDCLogoutNilProviderNoPanic covers the boot-failure path: when the
// provider failed to initialize (unreachable issuer), h.provider is nil and
// the old code panicked on h.provider.IsEnabled().
func TestOIDCLogoutNilProviderNoPanic(t *testing.T) {
	h := NewHandlers(nil, &mockSessionCreator{})

	req := httptest.NewRequest("GET", "/auth/oidc/logout", nil)
	rec := httptest.NewRecorder()

	// Must not panic and must fall back to /login.
	h.LogoutHandler(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 fallback redirect, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("expected redirect to /login, got %q", loc)
	}
}

// TestSaveConfigPreservesMaskedSecret: GET masks the client secret as
// "********"; a POST that round-trips the mask (or omits the secret) must keep
// the stored secret instead of overwriting the credential with the mask.
func TestSaveConfigPreservesMaskedSecret(t *testing.T) {
	dir := t.TempDir()
	p := &Provider{
		config:  &Config{ClientSecret: "real-secret"},
		dataDir: dir,
		states:  make(map[string]OIDCState),
		stop:    make(chan struct{}),
	}

	for _, posted := range []string{maskedSecret, ""} {
		cfg := &Config{
			Enabled:      false, // avoid provider discovery in SaveConfig
			IssuerURL:    "https://idp.example",
			ClientID:     "cid",
			ClientSecret: posted,
			RedirectURL:  "https://app.example/cb",
		}
		if err := p.SaveConfig(cfg); err != nil {
			t.Fatalf("SaveConfig(%q) failed: %v", posted, err)
		}
		p.mu.RLock()
		got := p.config.ClientSecret
		p.mu.RUnlock()
		if got != "real-secret" {
			t.Fatalf("posted secret %q: stored secret = %q, want the preserved real secret", posted, got)
		}
	}

	// A genuinely new secret must still be stored.
	cfg := &Config{IssuerURL: "https://idp.example", ClientID: "cid", ClientSecret: "new-secret", RedirectURL: "https://app.example/cb"}
	if err := p.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig(new) failed: %v", err)
	}
	p.mu.RLock()
	got := p.config.ClientSecret
	p.mu.RUnlock()
	if got != "new-secret" {
		t.Fatalf("new secret not stored: got %q", got)
	}
}
