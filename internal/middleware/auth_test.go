package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"casadrop/internal/utils"
)

func setupTestAdminAuth(t *testing.T) (*AdminAuth, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "casadrop-auth-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	auth := NewAdminAuth("", tmpDir)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return auth, cleanup
}

func TestNewAdminAuth(t *testing.T) {
	auth, cleanup := setupTestAdminAuth(t)
	defer cleanup()

	if auth == nil {
		t.Fatal("Expected non-nil AdminAuth")
	}
}

func TestAdminAuthNoPassword(t *testing.T) {
	auth, cleanup := setupTestAdminAuth(t)
	defer cleanup()

	// Without password, auth should not be enabled
	if auth.IsEnabled() {
		t.Error("Auth should not be enabled without password")
	}

	// Should need setup
	if !auth.NeedsSetup() {
		t.Error("Should need setup without password")
	}
}

func TestAdminAuthWithEnvPassword(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "casadrop-auth-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	auth := NewAdminAuth("test-password", tmpDir)

	if !auth.IsEnabled() {
		t.Error("Auth should be enabled with password")
	}

	if auth.NeedsSetup() {
		t.Error("Should not need setup with env password")
	}
}

func TestValidatePassword(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "casadrop-auth-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	auth := NewAdminAuth("correct-password", tmpDir)

	tests := []struct {
		password string
		valid    bool
	}{
		{"correct-password", true},
		{"wrong-password", false},
		{"", false},
		{"CORRECT-PASSWORD", false}, // Case sensitive
	}

	for _, tt := range tests {
		result := auth.ValidatePassword(tt.password)
		if result != tt.valid {
			t.Errorf("ValidatePassword(%q) = %v, want %v", tt.password, result, tt.valid)
		}
	}
}

func TestSetPassword(t *testing.T) {
	auth, cleanup := setupTestAdminAuth(t)
	defer cleanup()

	// Set a password
	err := auth.SetPassword("new-password")
	if err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}

	// Auth should now be enabled
	if !auth.IsEnabled() {
		t.Error("Auth should be enabled after SetPassword")
	}

	// Password should validate
	if !auth.ValidatePassword("new-password") {
		t.Error("New password should validate")
	}

	// Wrong password should not validate
	if auth.ValidatePassword("wrong") {
		t.Error("Wrong password should not validate")
	}
}

func TestSessionManagement(t *testing.T) {
	auth, cleanup := setupTestAdminAuth(t)
	defer cleanup()

	// Create a session
	token, err := auth.CreateSession("127.0.0.1", "Test-Agent")
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if token == "" {
		t.Fatal("CreateSession returned empty token")
	}

	// Session should be valid
	if !auth.validSession(token) {
		t.Error("New session should be valid")
	}

	// Get sessions
	sessions := auth.GetSessions()
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	// Invalidate session
	auth.InvalidateSession(token)
	if auth.validSession(token) {
		t.Error("Session should be invalid after invalidation")
	}
}

func TestCSRFToken(t *testing.T) {
	auth, cleanup := setupTestAdminAuth(t)
	defer cleanup()

	// Generate CSRF token
	token, err := auth.GenerateCSRFToken()
	if err != nil {
		t.Fatalf("GenerateCSRFToken returned error: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateCSRFToken returned empty token")
	}

	// Token should validate
	if !auth.ValidateCSRFToken(token) {
		t.Error("New CSRF token should validate")
	}

	// Consume token
	if !auth.ConsumeCSRFToken(token) {
		t.Error("ConsumeCSRFToken should return true for valid token")
	}

	// Token should no longer validate (consumed)
	if auth.ValidateCSRFToken(token) {
		t.Error("Consumed CSRF token should not validate")
	}
}

func TestRateLimiting(t *testing.T) {
	auth, cleanup := setupTestAdminAuth(t)
	defer cleanup()

	ip := "192.168.1.100"

	// Should allow initial requests
	for i := 0; i < 5; i++ {
		if !auth.rateLimiter.Allow(ip) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should be rate limited after burst
	if auth.rateLimiter.Allow(ip) {
		t.Error("Should be rate limited after burst")
	}
}

func TestAccountLockout(t *testing.T) {
	auth, cleanup := setupTestAdminAuth(t)
	defer cleanup()

	ip := "192.168.1.200"

	// Should not be locked out initially
	if auth.IsLockedOut(ip) {
		t.Error("Should not be locked out initially")
	}

	// Record failed attempts
	for i := 0; i < MaxFailedAttempts; i++ {
		auth.RecordFailedAttempt(ip)
	}

	// Should be locked out after max attempts
	if !auth.IsLockedOut(ip) {
		t.Error("Should be locked out after max failed attempts")
	}

	// Reset failed attempts
	auth.ResetFailedAttempts(ip)
	if auth.IsLockedOut(ip) {
		t.Error("Should not be locked out after reset")
	}
}

func TestMiddleware(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "casadrop-auth-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	auth := NewAdminAuth("test-password", tmpDir)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with middleware
	protected := auth.Middleware(testHandler)

	// Test without session (should redirect for non-API)
	t.Run("NoSession_Redirect", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		if rec.Code != http.StatusFound {
			t.Errorf("Expected redirect, got %d", rec.Code)
		}
	})

	// Test without session (should 401 for API)
	t.Run("NoSession_API_401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/shares", nil)
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", rec.Code)
		}
	})

	// Test with valid session cookie (must have role set to pass addUserToContext)
	t.Run("ValidSession", func(t *testing.T) {
		token, err := auth.CreateSessionForUser("127.0.0.1", "Test", "admin-id", "admin@test.com", "admin")
		if err != nil {
			t.Fatalf("CreateSessionForUser failed: %v", err)
		}
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: "casadrop_session", Value: token})
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rec.Code)
		}
	})

	// Test with Authorization header
	t.Run("AuthorizationHeader", func(t *testing.T) {
		token, err := auth.CreateSessionForUser("127.0.0.1", "Test", "admin-id", "admin@test.com", "admin")
		if err != nil {
			t.Fatalf("CreateSessionForUser failed: %v", err)
		}
		req := httptest.NewRequest("GET", "/api/shares", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rec.Code)
		}
	})
}

func TestLoginHandler(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "casadrop-auth-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	auth := NewAdminAuth("test-password", tmpDir)

	// Test GET (should render login page)
	t.Run("GET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/login", nil)
		rec := httptest.NewRecorder()

		auth.LoginHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "CasaDrop") {
			t.Error("Response should contain CasaDrop")
		}
	})

	// Test POST with correct password
	t.Run("POST_CorrectPassword", func(t *testing.T) {
		// First get CSRF token
		getReq := httptest.NewRequest("GET", "/login", nil)
		getRec := httptest.NewRecorder()
		auth.LoginHandler(getRec, getReq)

		// Extract CSRF token (simplified - in real test would parse HTML)
		csrf, err := auth.GenerateCSRFToken()
		if err != nil {
			t.Fatalf("GenerateCSRFToken failed: %v", err)
		}

		req := httptest.NewRequest("POST", "/login", strings.NewReader("password=test-password&csrf_token="+csrf))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		auth.LoginHandler(rec, req)

		// Should redirect on success
		if rec.Code != http.StatusFound {
			t.Errorf("Expected redirect, got %d", rec.Code)
		}
	})
}

func TestLogoutHandler(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "casadrop-auth-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	auth := NewAdminAuth("test-password", tmpDir)

	// Create a session
	token, err := auth.CreateSession("127.0.0.1", "Test")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Logout
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "casadrop_session", Value: token})
	rec := httptest.NewRecorder()

	auth.LogoutHandler(rec, req)

	// Should redirect
	if rec.Code != http.StatusFound {
		t.Errorf("Expected redirect, got %d", rec.Code)
	}

	// Session should be invalid
	if auth.validSession(token) {
		t.Error("Session should be invalid after logout")
	}
}

func TestAuthStatusHandler(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "casadrop-auth-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	auth := NewAdminAuth("test-password", tmpDir)

	// Test without session
	t.Run("NotAuthenticated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/status", nil)
		rec := httptest.NewRecorder()

		auth.AuthStatusHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"authenticated":false`) {
			t.Error("Should show not authenticated")
		}
	})

	// Test with session
	t.Run("Authenticated", func(t *testing.T) {
		token, _ := auth.CreateSession("127.0.0.1", "Test")
		req := httptest.NewRequest("GET", "/api/auth/status", nil)
		req.AddCookie(&http.Cookie{Name: "casadrop_session", Value: token})
		rec := httptest.NewRecorder()

		auth.AuthStatusHandler(rec, req)

		if !strings.Contains(rec.Body.String(), `"authenticated":true`) {
			t.Error("Should show authenticated")
		}
	})
}

func TestOIDCStatusTracking(t *testing.T) {
	auth, cleanup := setupTestAdminAuth(t)
	defer cleanup()

	// Initially OIDC should be disabled
	if auth.IsOIDCEnabled() {
		t.Error("OIDC should be disabled initially")
	}

	// Set OIDC status
	auth.SetOIDCStatus(true, false)
	if !auth.IsOIDCEnabled() {
		t.Error("OIDC should be enabled after SetOIDCStatus")
	}
	if !auth.IsLocalAuthAllowed() {
		t.Error("Local auth should be allowed when OIDC local disabled is false")
	}

	// Disable local auth
	auth.SetOIDCStatus(true, true)
	if auth.IsLocalAuthAllowed() {
		t.Error("Local auth should not be allowed when OIDC local disabled is true")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(3, time.Second)

	ip := "10.0.0.1"

	// Should allow first 3 requests
	for i := 0; i < 3; i++ {
		if !limiter.Allow(ip) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be blocked
	if limiter.Allow(ip) {
		t.Error("4th request should be blocked")
	}

	// Different IP should have its own limit
	if !limiter.Allow("10.0.0.2") {
		t.Error("Different IP should be allowed")
	}

	// Wait for rate limit to reset
	time.Sleep(time.Second + 100*time.Millisecond)

	// Should allow again
	if !limiter.Allow(ip) {
		t.Error("Should allow after rate limit window")
	}
}

func TestGetClientIPViaUtils(t *testing.T) {
	// Fail-closed contract: with no TRUSTED_PROXY configured (the test default),
	// forwarded headers are NOT trusted and the real socket peer is used, so a
	// direct client can't spoof its IP to evade rate-limit/lockout. Honoring of
	// X-Forwarded-* from configured proxies is covered in utils' own tests.
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
			name:       "X-Forwarded-For ignored from untrusted peer",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For chain ignored from untrusted peer",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2, 10.0.0.3"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Real-IP ignored from untrusted peer",
			headers:    map[string]string{"X-Real-IP": "172.16.0.1"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := utils.GetClientIP(req)
			if ip != tt.expected {
				t.Errorf("GetClientIP() = %s, want %s", ip, tt.expected)
			}
		})
	}
}
