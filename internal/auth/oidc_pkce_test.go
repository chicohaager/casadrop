package auth

import (
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

// Verifies the auth-code flow uses PKCE (S256) and that GenerateAuthURL returns
// the state (so the handler can bind it to the browser) with a stored verifier.
func TestGenerateAuthURL_PKCE(t *testing.T) {
	p := &Provider{
		config: &Config{Enabled: true},
		oauth2Config: &oauth2.Config{
			ClientID:    "cid",
			RedirectURL: "https://app.example/auth/oidc/callback",
			Scopes:      []string{"openid", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://idp.example/authorize",
				TokenURL: "https://idp.example/token",
			},
		},
		states: make(map[string]OIDCState),
	}

	authURL, state, err := p.GenerateAuthURL("/dashboard")
	if err != nil {
		t.Fatalf("GenerateAuthURL: %v", err)
	}
	if state == "" {
		t.Fatal("GenerateAuthURL returned an empty state (needed for browser binding)")
	}
	if !strings.Contains(authURL, "code_challenge=") {
		t.Errorf("auth URL missing code_challenge (PKCE): %s", authURL)
	}
	if !strings.Contains(authURL, "code_challenge_method=S256") {
		t.Errorf("auth URL missing code_challenge_method=S256: %s", authURL)
	}

	p.mu.RLock()
	st, ok := p.states[state]
	p.mu.RUnlock()
	if !ok {
		t.Fatal("state was not stored")
	}
	if st.CodeVerifier == "" {
		t.Error("stored state has no PKCE code verifier")
	}
	if st.ReturnURL != "/dashboard" {
		t.Errorf("ReturnURL = %q, want /dashboard", st.ReturnURL)
	}
}
