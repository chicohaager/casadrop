package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Regression test for the setup-wizard takeover guard (C1): the unauthenticated
// /setup POST must require the one-time setup token printed to the logs, so an
// internet-exposed, not-yet-configured instance can't be claimed by whoever
// reaches /setup first.
func TestSetupRequiresToken(t *testing.T) {
	aa, cleanup := setupTestAdminAuth(t)
	defer cleanup()

	if aa.setupToken == "" {
		t.Fatal("expected a setup token to be generated when no ADMIN_PASSWORD and setup not done")
	}

	post := func(setupTok string) *httptest.ResponseRecorder {
		csrf, err := aa.GenerateCSRFToken()
		if err != nil {
			t.Fatalf("csrf: %v", err)
		}
		form := url.Values{
			"csrf_token":       {csrf},
			"setup_token":      {setupTok},
			"password":         {"supersecret1"},
			"confirm_password": {"supersecret1"},
		}
		req := httptest.NewRequest("POST", "/setup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		aa.SetupHandler(rec, req)
		return rec
	}

	// Wrong token: setup must be refused and no admin created.
	rec := post("not-the-token")
	if aa.IsEnabled() {
		t.Fatal("setup succeeded with a wrong token — takeover guard ineffective")
	}
	if rec.Code == http.StatusFound {
		t.Fatalf("wrong token should not redirect to success, got %d", rec.Code)
	}

	// Correct token: setup completes and auth becomes enabled.
	good := aa.setupToken
	rec = post(good)
	if !aa.IsEnabled() {
		t.Fatalf("setup with the correct token should enable auth (status %d)", rec.Code)
	}
}
