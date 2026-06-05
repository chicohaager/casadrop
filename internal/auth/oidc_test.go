package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"casadrop/internal/models"
)

func setupTestOIDC(t *testing.T) (*Provider, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "casadrop-oidc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Clear any OIDC env vars for clean test
	os.Unsetenv(EnvOIDCEnabled)
	os.Unsetenv(EnvOIDCIssuerURL)
	os.Unsetenv(EnvOIDCClientID)

	provider, err := NewProvider(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create provider: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return provider, cleanup
}

func TestNewProvider(t *testing.T) {
	provider, cleanup := setupTestOIDC(t)
	defer cleanup()

	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}

	// Should not be enabled without config
	if provider.IsEnabled() {
		t.Error("Provider should not be enabled without config")
	}
}

func TestIsEnvConfigured(t *testing.T) {
	// Clear env first
	os.Unsetenv(EnvOIDCEnabled)
	os.Unsetenv(EnvOIDCIssuerURL)
	os.Unsetenv(EnvOIDCClientID)

	if isEnvConfigured() {
		t.Error("Should not be env configured with no env vars")
	}

	// Set OIDC_ENABLED
	os.Setenv(EnvOIDCEnabled, "true")
	if !isEnvConfigured() {
		t.Error("Should be env configured with OIDC_ENABLED=true")
	}
	os.Unsetenv(EnvOIDCEnabled)

	// Set OIDC_ISSUER_URL
	os.Setenv(EnvOIDCIssuerURL, "https://example.com")
	if !isEnvConfigured() {
		t.Error("Should be env configured with OIDC_ISSUER_URL set")
	}
	os.Unsetenv(EnvOIDCIssuerURL)

	// Set OIDC_CLIENT_ID
	os.Setenv(EnvOIDCClientID, "client123")
	if !isEnvConfigured() {
		t.Error("Should be env configured with OIDC_CLIENT_ID set")
	}
	os.Unsetenv(EnvOIDCClientID)
}

func TestParseScopes(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{"openid", "profile", "email"}},
		{"openid", []string{"openid"}},
		{"profile,email", []string{"openid", "profile", "email"}}, // openid should be prepended
		{"openid,profile,email", []string{"openid", "profile", "email"}},
		{"openid, profile, email", []string{"openid", "profile", "email"}}, // with spaces
	}

	for _, tt := range tests {
		result := parseScopes(tt.input)

		// Check openid is always first
		if len(result) > 0 && result[0] != "openid" {
			t.Errorf("parseScopes(%q): openid should be first, got %v", tt.input, result)
		}

		// Check length
		if len(result) != len(tt.expected) {
			t.Errorf("parseScopes(%q): got %d scopes, want %d", tt.input, len(result), len(tt.expected))
		}
	}
}

func TestGenerateRandomString(t *testing.T) {
	// generateRandomString now returns the full base64-URL-encoded form of `length`
	// random bytes (truncating the encoded output would drop entropy — that was
	// the bug this test used to pin). 32 random bytes encode to 43 unpadded chars.
	const requestedBytes = 32
	const expectedLen = 43

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s, err := generateRandomString(requestedBytes)
		if err != nil {
			t.Fatalf("generateRandomString failed: %v", err)
		}
		if len(s) != expectedLen {
			t.Errorf("Expected length %d, got %d", expectedLen, len(s))
		}
		if seen[s] {
			t.Error("Generated duplicate string")
		}
		seen[s] = true
	}
}

func TestGetConfig(t *testing.T) {
	provider, cleanup := setupTestOIDC(t)
	defer cleanup()

	config := provider.GetConfig()

	// Should return a config even if empty
	if config.Enabled {
		t.Error("Config should not be enabled by default")
	}
}

func TestSaveConfig(t *testing.T) {
	provider, cleanup := setupTestOIDC(t)
	defer cleanup()

	// Note: This will fail to initialize because we don't have a real OIDC server
	// but it should save the config to file
	config := &Config{
		Enabled:      false, // Keep disabled to avoid initialization
		IssuerURL:    "https://auth.example.com",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "https://app.example.com/callback",
		Scopes:       "openid,profile",
	}

	err := provider.SaveConfig(config)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify config was saved
	retrieved := provider.GetConfig()
	if retrieved.IssuerURL != config.IssuerURL {
		t.Errorf("IssuerURL mismatch: got %s, want %s", retrieved.IssuerURL, config.IssuerURL)
	}
	if retrieved.ClientID != config.ClientID {
		t.Errorf("ClientID mismatch: got %s, want %s", retrieved.ClientID, config.ClientID)
	}
}

func TestStateManagement(t *testing.T) {
	provider, cleanup := setupTestOIDC(t)
	defer cleanup()

	// States should be empty initially
	provider.mu.RLock()
	stateCount := len(provider.states)
	provider.mu.RUnlock()

	if stateCount != 0 {
		t.Errorf("Expected 0 states initially, got %d", stateCount)
	}
}

// Mock SessionCreator for testing handlers
type mockSessionCreator struct {
	lastIP        string
	lastUserAgent string
	token         string
}

func (m *mockSessionCreator) CreateSession(ip, userAgent string) (string, error) {
	m.lastIP = ip
	m.lastUserAgent = userAgent
	if m.token == "" {
		m.token = "mock-session-token"
	}
	return m.token, nil
}

func (m *mockSessionCreator) CreateSessionForUser(ip, userAgent, userID, userEmail string, role models.Role) (string, error) {
	m.lastIP = ip
	m.lastUserAgent = userAgent
	if m.token == "" {
		m.token = "mock-session-token"
	}
	return m.token, nil
}

func TestHandlers_StatusHandler(t *testing.T) {
	provider, cleanup := setupTestOIDC(t)
	defer cleanup()

	handlers := NewHandlers(provider, &mockSessionCreator{})

	req := httptest.NewRequest("GET", "/api/auth/oidc/status", nil)
	rec := httptest.NewRecorder()

	handlers.StatusHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"enabled":false`) {
		t.Error("Response should show enabled:false")
	}
	if !strings.Contains(body, `"envConfigured"`) {
		t.Error("Response should include envConfigured field")
	}
}

func TestHandlers_StatusHandler_NilProvider(t *testing.T) {
	handlers := NewHandlers(nil, &mockSessionCreator{})

	req := httptest.NewRequest("GET", "/api/auth/oidc/status", nil)
	rec := httptest.NewRecorder()

	handlers.StatusHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"enabled":false`) {
		t.Error("Response should show enabled:false for nil provider")
	}
}

func TestHandlers_LoginHandler_Disabled(t *testing.T) {
	provider, cleanup := setupTestOIDC(t)
	defer cleanup()

	handlers := NewHandlers(provider, &mockSessionCreator{})

	req := httptest.NewRequest("GET", "/auth/oidc/login", nil)
	rec := httptest.NewRecorder()

	handlers.LoginHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404 when OIDC disabled, got %d", rec.Code)
	}
}

func TestHandlers_CallbackHandler_Disabled(t *testing.T) {
	provider, cleanup := setupTestOIDC(t)
	defer cleanup()

	handlers := NewHandlers(provider, &mockSessionCreator{})

	req := httptest.NewRequest("GET", "/auth/oidc/callback?code=test&state=test", nil)
	rec := httptest.NewRecorder()

	handlers.CallbackHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404 when OIDC disabled, got %d", rec.Code)
	}
}

func TestHandlers_ConfigHandler_EnvProtection(t *testing.T) {
	provider, cleanup := setupTestOIDC(t)
	defer cleanup()

	handlers := NewHandlers(provider, &mockSessionCreator{})

	// Set env var to enable protection
	os.Setenv(EnvOIDCEnabled, "true")
	defer os.Unsetenv(EnvOIDCEnabled)

	req := httptest.NewRequest("POST", "/api/auth/oidc/config", strings.NewReader(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.ConfigHandler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected 403 when env configured, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "environment variables") {
		t.Error("Response should mention environment variables")
	}
}

func TestHandlers_ConfigHandler_GET(t *testing.T) {
	provider, cleanup := setupTestOIDC(t)
	defer cleanup()

	handlers := NewHandlers(provider, &mockSessionCreator{})

	req := httptest.NewRequest("GET", "/api/auth/oidc/config", nil)
	rec := httptest.NewRecorder()

	handlers.ConfigHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"envConfigured"`) {
		t.Error("Response should include envConfigured")
	}
}

func TestHandlers_LogoutHandler(t *testing.T) {
	provider, cleanup := setupTestOIDC(t)
	defer cleanup()

	handlers := NewHandlers(provider, &mockSessionCreator{})

	req := httptest.NewRequest("GET", "/auth/oidc/logout", nil)
	rec := httptest.NewRecorder()

	handlers.LogoutHandler(rec, req)

	// Should redirect to login
	if rec.Code != http.StatusFound {
		t.Errorf("Expected redirect, got %d", rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/login" {
		t.Errorf("Expected redirect to /login, got %s", location)
	}

	// Should clear session cookie
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "casadrop_session" && c.MaxAge < 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Session cookie should be cleared")
	}
}

func TestHandlers_getClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "172.16.0.1"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "172.16.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("getClientIP() = %s, want %s", ip, tt.expected)
			}
		})
	}
}
