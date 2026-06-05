package middleware

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"

	"casadrop/internal/models"
	"casadrop/internal/totp"
	"casadrop/internal/utils"
)

// APIKeyValidator provides API key validation capability
type APIKeyValidator interface {
	GetAPIKeyByHash(keyHash string) (id, name, userID, role string, isActive bool, err error)
	UpdateAPIKeyLastUsed(id string)
}

// LocalUserStore provides per-user local credential lookup so users created in
// the users table (email + password) can log in with their own role — not just
// the single shared admin password.
type LocalUserStore interface {
	GetUserByEmail(email string) (*models.User, error)
}

// AuditEventType represents the type of security audit event
type AuditEventType string

const (
	AuditLoginSuccess   AuditEventType = "LOGIN_SUCCESS"
	AuditLoginFailed    AuditEventType = "LOGIN_FAILED"
	AuditLoginLocked    AuditEventType = "LOGIN_LOCKED"
	AuditLogout         AuditEventType = "LOGOUT"
	AuditSetupComplete  AuditEventType = "SETUP_COMPLETE"
	AuditSessionCreated AuditEventType = "SESSION_CREATED"
	AuditSessionExpired AuditEventType = "SESSION_EXPIRED"
	AuditCSRFViolation  AuditEventType = "CSRF_VIOLATION"
	AuditRateLimitHit   AuditEventType = "RATE_LIMIT_HIT"
)

// LogAuditEvent logs a security-relevant event
func LogAuditEvent(eventType AuditEventType, ip, userAgent, details string) {
	log.Printf("[AUDIT] %s | IP: %s | UA: %.50s | %s", eventType, ip, userAgent, details)
}

// failedAttemptInfo tracks failed login attempts with timestamps for time-based cleanup
type failedAttemptInfo struct {
	count       int
	lastAttempt time.Time
}

// AdminAuth handles admin authentication
type AdminAuth struct {
	envPassword       string                        // Password from environment variable
	dataDir           string                        // Data directory for persistent storage
	sessions          map[string]Session            // Active sessions
	csrfTokens        map[string]time.Time          // CSRF tokens with expiry
	failedAttempts    map[string]*failedAttemptInfo // Failed login attempts per IP (for lockout)
	mu                sync.RWMutex
	rateLimiter       *RateLimiter
	config            *AdminConfig
	oidcEnabled       bool                                 // Whether OIDC is enabled (startup-cached fallback)
	oidcLocalDisabled bool                                 // Whether local auth is disabled when OIDC is enabled (fallback)
	oidcStatusFn      func() (enabled, localDisabled bool) // Live OIDC status (single source of truth)
	apiKeyValidator   APIKeyValidator                      // Optional API key validator
	userStore         LocalUserStore                       // Optional per-user local credential store
	stop              chan struct{}                        // Shutdown signal for background cleanup goroutine
	stopOnce          sync.Once
}

// Session represents an authenticated session
type Session struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expiresAt"`
	IP        string      `json:"ip"`
	UserAgent string      `json:"userAgent"`
	CreatedAt time.Time   `json:"createdAt"`
	UserID    string      `json:"userId,omitempty"`
	UserEmail string      `json:"userEmail,omitempty"`
	UserRole  models.Role `json:"userRole,omitempty"`
}

// Context key for user information
type contextKey string

const userContextKey contextKey = "user"

// SessionUser represents user info stored in context
type SessionUser struct {
	ID    string
	Email string
	Role  models.Role
}

// AdminConfig stores the admin password hash
type AdminConfig struct {
	PasswordHash string `json:"passwordHash"`
	SetupDone    bool   `json:"setupDone"`
	// Optional TOTP second factor for the local admin password login.
	TOTPSecret  string `json:"totpSecret,omitempty"`
	TOTPEnabled bool   `json:"totpEnabled,omitempty"`
	// LastTOTPCounter is the most recently consumed 30s step counter. A code is
	// accepted only if its counter is strictly greater, making each code
	// single-use (anti-replay) within the acceptance window.
	LastTOTPCounter uint64 `json:"lastTotpCounter,omitempty"`
}

// Constants for security settings
const (
	MaxFailedAttempts = 10               // Account lockout after this many failures
	LockoutDuration   = 15 * time.Minute // Lockout duration
	CSRFTokenExpiry   = 1 * time.Hour    // CSRF token validity
	SessionIdleTTL    = 24 * time.Hour   // Rolling idle timeout (extended on activity)
	// SessionAbsoluteTTL caps total session age regardless of activity, so a
	// stolen token can't be kept alive indefinitely by periodic use.
	SessionAbsoluteTTL = 7 * 24 * time.Hour
)

// NewAdminAuth creates a new admin auth middleware
func NewAdminAuth(envPassword string, dataDir string) *AdminAuth {
	aa := &AdminAuth{
		envPassword:    envPassword,
		dataDir:        dataDir,
		sessions:       make(map[string]Session),
		csrfTokens:     make(map[string]time.Time),
		failedAttempts: make(map[string]*failedAttemptInfo),
		rateLimiter:    NewRateLimiter(5, time.Minute), // 5 attempts per minute
		stop:           make(chan struct{}),
	}

	aa.loadConfig()
	aa.loadSessions()

	// Session and token cleanup. Exits on Stop() so the goroutine doesn't
	// leak on graceful shutdown.
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				aa.cleanupSessions()
				aa.cleanupCSRFTokens()
				aa.cleanupFailedAttempts()
			case <-aa.stop:
				return
			}
		}
	}()

	return aa
}

// Stop terminates background cleanup goroutines (including the embedded
// RateLimiter). Safe to call multiple times.
func (aa *AdminAuth) Stop() {
	aa.stopOnce.Do(func() {
		close(aa.stop)
	})
	if aa.rateLimiter != nil {
		aa.rateLimiter.Stop()
	}
}

// SetAPIKeyValidator sets the API key validator for middleware authentication
func (aa *AdminAuth) SetAPIKeyValidator(v APIKeyValidator) {
	aa.apiKeyValidator = v
}

// SetLocalUserStore enables per-user local (email+password) authentication.
func (aa *AdminAuth) SetLocalUserStore(s LocalUserStore) {
	aa.userStore = s
}

// Config file paths
func (aa *AdminAuth) configPath() string {
	return filepath.Join(aa.dataDir, "admin_config.json")
}

func (aa *AdminAuth) sessionsPath() string {
	return filepath.Join(aa.dataDir, "sessions.json")
}

// loadConfig loads admin config from file
func (aa *AdminAuth) loadConfig() {
	data, err := os.ReadFile(aa.configPath())
	if err != nil {
		aa.config = &AdminConfig{SetupDone: false}
		return
	}

	var config AdminConfig
	if err := json.Unmarshal(data, &config); err != nil {
		aa.config = &AdminConfig{SetupDone: false}
		return
	}
	aa.config = &config
}

// saveConfig saves admin config to file
func (aa *AdminAuth) saveConfig() error {
	data, err := json.MarshalIndent(aa.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(aa.configPath(), data, 0600)
}

// loadSessions loads sessions from disk
func (aa *AdminAuth) loadSessions() {
	data, err := os.ReadFile(aa.sessionsPath())
	if err != nil {
		return
	}

	var sessions map[string]Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return
	}

	// Filter expired sessions
	now := time.Now()
	aa.sessions = make(map[string]Session)
	for token, session := range sessions {
		if now.Before(session.ExpiresAt) {
			aa.sessions[token] = session
		}
	}
}

// saveSessions persists sessions to disk
func (aa *AdminAuth) saveSessions() {
	data, err := json.MarshalIndent(aa.sessions, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(aa.sessionsPath(), data, 0600)
}

func (aa *AdminAuth) cleanupSessions() {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	now := time.Now()
	changed := false
	for token, session := range aa.sessions {
		if now.After(session.ExpiresAt) {
			delete(aa.sessions, token)
			changed = true
		}
	}
	if changed {
		aa.saveSessions()
	}
}

func (aa *AdminAuth) cleanupCSRFTokens() {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	now := time.Now()
	for token, expiry := range aa.csrfTokens {
		if now.After(expiry) {
			delete(aa.csrfTokens, token)
		}
	}
}

func (aa *AdminAuth) cleanupFailedAttempts() {
	aa.mu.Lock()
	defer aa.mu.Unlock()
	cutoff := time.Now().Add(-LockoutDuration)
	for ip, info := range aa.failedAttempts {
		if info.lastAttempt.Before(cutoff) {
			delete(aa.failedAttempts, ip)
		}
	}
}

// GenerateCSRFToken creates a new CSRF token
func (aa *AdminAuth) GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(b)

	aa.mu.Lock()
	aa.csrfTokens[token] = time.Now().Add(CSRFTokenExpiry)
	aa.mu.Unlock()

	return token, nil
}

// ValidateCSRFToken checks if a CSRF token is valid
func (aa *AdminAuth) ValidateCSRFToken(token string) bool {
	aa.mu.RLock()
	expiry, exists := aa.csrfTokens[token]
	aa.mu.RUnlock()

	if !exists {
		return false
	}
	return time.Now().Before(expiry)
}

// ConsumeCSRFToken validates and removes a CSRF token (one-time use)
func (aa *AdminAuth) ConsumeCSRFToken(token string) bool {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	expiry, exists := aa.csrfTokens[token]
	if !exists || time.Now().After(expiry) {
		return false
	}
	delete(aa.csrfTokens, token)
	return true
}

// IsLockedOut checks if an IP is locked out
func (aa *AdminAuth) IsLockedOut(ip string) bool {
	aa.mu.RLock()
	defer aa.mu.RUnlock()
	info, exists := aa.failedAttempts[ip]
	if !exists {
		return false
	}
	// Only locked out if attempts are recent (within lockout duration)
	if time.Since(info.lastAttempt) > LockoutDuration {
		return false
	}
	return info.count >= MaxFailedAttempts
}

// RecordFailedAttempt records a failed login attempt
func (aa *AdminAuth) RecordFailedAttempt(ip string) int {
	aa.mu.Lock()
	defer aa.mu.Unlock()
	info, exists := aa.failedAttempts[ip]
	if !exists {
		info = &failedAttemptInfo{}
		aa.failedAttempts[ip] = info
	}
	info.count++
	info.lastAttempt = time.Now()
	return info.count
}

// ResetFailedAttempts resets failed attempts for an IP
func (aa *AdminAuth) ResetFailedAttempts(ip string) {
	aa.mu.Lock()
	defer aa.mu.Unlock()
	delete(aa.failedAttempts, ip)
}

// IsEnabled returns true if authentication is required
func (aa *AdminAuth) IsEnabled() bool {
	return aa.envPassword != "" || (aa.config != nil && aa.config.SetupDone)
}

// NeedsSetup returns true if initial setup is required
func (aa *AdminAuth) NeedsSetup() bool {
	return aa.envPassword == "" && (aa.config == nil || !aa.config.SetupDone)
}

// ValidatePassword checks if the password is correct.
//
// We always perform a bcrypt comparison (against a dummy hash when no real
// credential is configured) so the response time does not leak which auth
// mode is active (env password, stored hash, neither). subtle.ConstantTimeCompare
// handles the env-password byte comparison in constant time, and the dummy
// bcrypt verification below burns the same work as a real check.
func (aa *AdminAuth) ValidatePassword(password string) bool {
	// Pre-generated dummy bcrypt hash (cost 12) over a random secret the
	// caller can never reach. Used solely to equalize timing when no real
	// credential is configured.
	const dummyHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEeOVhBxO4YWj7uR0K5kO5gH7u8H9vT1jia"

	envOK := false
	if aa.envPassword != "" {
		envOK = subtle.ConstantTimeCompare([]byte(password), []byte(aa.envPassword)) == 1
	}

	var storedHash string
	if aa.config != nil && aa.config.SetupDone && aa.config.PasswordHash != "" {
		storedHash = aa.config.PasswordHash
	} else {
		storedHash = dummyHash
	}
	hashOK := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
	// Only count the stored-hash result when a real one exists.
	if storedHash == dummyHash {
		hashOK = false
	}

	return envOK || hashOK
}

// authenticateLocalUser verifies an email+password against the users table.
// Returns the user on success. A bcrypt comparison is always performed (against
// the user's hash or a dummy) so response time does not reveal whether the email
// exists — mirroring the timing-safety of ValidatePassword.
func (aa *AdminAuth) authenticateLocalUser(email, password string) (*models.User, bool) {
	const dummyHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEeOVhBxO4YWj7uR0K5kO5gH7u8H9vT1jia"

	if aa.userStore == nil || email == "" {
		return nil, false
	}
	user, err := aa.userStore.GetUserByEmail(email)
	if err != nil || user == nil || !user.IsActive || user.PasswordHash == "" {
		// Burn equivalent work so a missing/inactive account isn't distinguishable by timing.
		_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
		return nil, false
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, false
	}
	return user, true
}

// resolveLogin authenticates a login attempt and returns the resulting session
// identity. It tries per-user local credentials first (when an email is given),
// then falls back to the single admin password (env var or setup-wizard hash)
// for backward compatibility. The returned role drives RBAC for the session.
func (aa *AdminAuth) resolveLogin(email, password string) (userID, userEmail string, role models.Role, ok bool) {
	if user, found := aa.authenticateLocalUser(email, password); found {
		return user.ID, user.Email, user.Role, true
	}
	if aa.ValidatePassword(password) {
		return "", "", models.RoleAdmin, true
	}
	return "", "", "", false
}

// SetPassword sets a new password (hashed)
func (aa *AdminAuth) SetPassword(password string) error {
	// cost 12: see internal/auth.BcryptCost — slightly above bcrypt.DefaultCost (10)
	// for better resistance against offline cracking
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	// Preserve any existing TOTP config across password changes.
	if aa.config == nil {
		aa.config = &AdminConfig{}
	}
	aa.config.PasswordHash = string(hash)
	aa.config.SetupDone = true
	return aa.saveConfig()
}

// IsTOTPEnabled reports whether admin TOTP 2FA is active.
func (aa *AdminAuth) IsTOTPEnabled() bool {
	return aa.config != nil && aa.config.TOTPEnabled && aa.config.TOTPSecret != ""
}

// verifyTOTP validates a 6-digit code against the configured admin secret and
// enforces single-use (anti-replay): a code is rejected if its 30s step counter
// is at or below the last consumed counter. Returns true when 2FA is not
// enabled (nothing to verify). Takes aa.mu because it persists the counter.
func (aa *AdminAuth) verifyTOTP(code string) bool {
	aa.mu.Lock()
	defer aa.mu.Unlock()
	if aa.config == nil || !aa.config.TOTPEnabled || aa.config.TOTPSecret == "" {
		return true
	}
	counter, ok := totp.ValidateWithCounter(aa.config.TOTPSecret, code)
	if !ok {
		return false
	}
	// Reject reuse of an already-consumed (or older) code within the window.
	if aa.config.LastTOTPCounter != 0 && counter <= aa.config.LastTOTPCounter {
		return false
	}
	aa.config.LastTOTPCounter = counter
	_ = aa.saveConfig()
	return true
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// TOTPStatusHandler reports whether admin 2FA is enabled (admin only).
func (aa *AdminAuth) TOTPStatusHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": aa.IsTOTPEnabled()})
}

// TOTPSetupHandler generates a fresh secret + enrollment QR (admin only). The
// secret is persisted only after confirmation via TOTPEnableHandler.
func (aa *AdminAuth) TOTPSetupHandler(w http.ResponseWriter, r *http.Request) {
	secret, err := totp.GenerateSecret()
	if err != nil {
		http.Error(w, "Failed to generate secret", http.StatusInternalServerError)
		return
	}
	uri := totp.ProvisioningURI(secret, "admin", "CasaDrop")
	png, err := qrcode.Encode(uri, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "Failed to render QR", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"secret": secret,
		"uri":    uri,
		"qr":     "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
	})
}

// TOTPEnableHandler verifies a code against the supplied secret and, on success,
// persists it as the admin second factor (admin only).
func (aa *AdminAuth) TOTPEnableHandler(w http.ResponseWriter, r *http.Request) {
	var req struct{ Secret, Code string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	enrollCounter, ok := totp.ValidateWithCounter(req.Secret, req.Code)
	if req.Secret == "" || !ok {
		http.Error(w, "Invalid or expired code", http.StatusBadRequest)
		return
	}
	aa.mu.Lock()
	if aa.config == nil {
		aa.config = &AdminConfig{SetupDone: true}
	}
	aa.config.TOTPSecret = req.Secret
	aa.config.TOTPEnabled = true
	// Consume the enrollment code so it can't immediately be replayed to log in.
	aa.config.LastTOTPCounter = enrollCounter
	err := aa.saveConfig()
	aa.mu.Unlock()
	if err != nil {
		http.Error(w, "Failed to save", http.StatusInternalServerError)
		return
	}
	LogAuditEvent(AuditSetupComplete, utils.GetClientIP(r), r.Header.Get("User-Agent"), "Admin 2FA enabled")
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": true})
}

// TOTPDisableHandler turns off admin 2FA after verifying a current code (admin only).
func (aa *AdminAuth) TOTPDisableHandler(w http.ResponseWriter, r *http.Request) {
	var req struct{ Code string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if !aa.IsTOTPEnabled() {
		writeJSON(w, http.StatusOK, map[string]bool{"enabled": false})
		return
	}
	if !aa.verifyTOTP(req.Code) {
		http.Error(w, "Invalid code", http.StatusBadRequest)
		return
	}
	aa.mu.Lock()
	aa.config.TOTPEnabled = false
	aa.config.TOTPSecret = ""
	err := aa.saveConfig()
	aa.mu.Unlock()
	if err != nil {
		http.Error(w, "Failed to save", http.StatusInternalServerError)
		return
	}
	LogAuditEvent(AuditSetupComplete, utils.GetClientIP(r), r.Header.Get("User-Agent"), "Admin 2FA disabled")
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": false})
}

// Middleware checks if the request is authenticated
func (aa *AdminAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If no password configured and no setup done, allow access (but show setup)
		if !aa.IsEnabled() {
			// Redirect to setup if needed
			if aa.NeedsSetup() && r.URL.Path != "/setup" && r.URL.Path != "/api/auth/setup" {
				http.Redirect(w, r, "/setup", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		// Check for API key in X-API-Key header
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" && aa.apiKeyValidator != nil {
			hash := sha256.Sum256([]byte(apiKey))
			keyHash := hex.EncodeToString(hash[:])
			id, _, userID, role, isActive, err := aa.apiKeyValidator.GetAPIKeyByHash(keyHash)
			if err == nil && id != "" && isActive {
				go aa.apiKeyValidator.UpdateAPIKeyLastUsed(id)
				user := &SessionUser{
					ID:    userID,
					Email: "api-key:" + id,
					Role:  models.Role(role),
				}
				ctx := context.WithValue(r.Context(), userContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// Invalid API key - fall through to other auth methods
		}

		// Check session cookie
		cookie, err := r.Cookie("casadrop_session")
		if err == nil {
			if session := aa.getSession(cookie.Value); session != nil {
				// Extend session on activity
				aa.extendSession(cookie.Value)
				// Add user to context (nil means legacy session — invalidate it)
				if updatedR := aa.addUserToContext(r, session); updatedR != nil {
					next.ServeHTTP(w, updatedR)
					return
				}
				// Legacy session without role — invalidate and fall through to login
				aa.InvalidateSession(cookie.Value)
			}
		}

		// Check Authorization header (for API clients)
		if token := r.Header.Get("Authorization"); token != "" {
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
				if session := aa.getSession(token); session != nil {
					aa.extendSession(token)
					// Add user to context (nil means legacy session — invalidate it)
					if updatedR := aa.addUserToContext(r, session); updatedR != nil {
						next.ServeHTTP(w, updatedR)
						return
					}
					aa.InvalidateSession(token)
				}
			}
		}

		// Not authenticated - redirect to login or return 401 for API
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		http.Redirect(w, r, "/login", http.StatusFound)
	})
}

// getSession returns a session if it exists and is valid
func (aa *AdminAuth) getSession(token string) *Session {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	session, exists := aa.sessions[token]
	if !exists {
		return nil
	}
	now := time.Now()
	if now.After(session.ExpiresAt) {
		return nil
	}
	// Absolute lifetime cap: ignore sessions older than SessionAbsoluteTTL even
	// if they were kept alive by activity.
	if !session.CreatedAt.IsZero() && now.After(session.CreatedAt.Add(SessionAbsoluteTTL)) {
		return nil
	}
	return &session
}

// addUserToContext adds user info to the request context
func (aa *AdminAuth) addUserToContext(r *http.Request, session *Session) *http.Request {
	// Legacy sessions without a role are invalid — force re-authentication
	if session.UserRole == "" {
		return nil
	}

	user := &SessionUser{
		ID:    session.UserID,
		Email: session.UserEmail,
		Role:  session.UserRole,
	}
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// GetUserFromContext returns the user from the request context
func GetUserFromContext(ctx context.Context) *SessionUser {
	user, ok := ctx.Value(userContextKey).(*SessionUser)
	if !ok {
		return nil
	}
	return user
}

// ContextWithUser returns a copy of ctx carrying the given user, using the same
// key the auth middleware uses. Exposed so route wiring and tests can inject an
// authenticated identity without going through a full login.
func ContextWithUser(ctx context.Context, user *SessionUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// RequireRole middleware checks if user has one of the allowed roles
func (aa *AdminAuth) RequireRole(roles ...models.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Insufficient permissions"})
		})
	}
}

// RequireAdmin middleware ensures user is admin
func (aa *AdminAuth) RequireAdmin() func(http.Handler) http.Handler {
	return aa.RequireRole(models.RoleAdmin)
}

// RequireCanCreateShares middleware ensures user can create shares (admin or user)
func (aa *AdminAuth) RequireCanCreateShares() func(http.Handler) http.Handler {
	return aa.RequireRole(models.RoleAdmin, models.RoleUser)
}

func (aa *AdminAuth) validSession(token string) bool {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	session, exists := aa.sessions[token]
	if !exists {
		return false
	}
	return time.Now().Before(session.ExpiresAt)
}

func (aa *AdminAuth) extendSession(token string) {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	if session, exists := aa.sessions[token]; exists {
		newExpiry := time.Now().Add(SessionIdleTTL)
		// Never extend past the absolute lifetime cap.
		if !session.CreatedAt.IsZero() {
			if cap := session.CreatedAt.Add(SessionAbsoluteTTL); newExpiry.After(cap) {
				newExpiry = cap
			}
		}
		session.ExpiresAt = newExpiry
		aa.sessions[token] = session
		aa.saveSessions()
	}
}

// CreateSession creates a new session (backward compatible - creates admin session)
func (aa *AdminAuth) CreateSession(ip, userAgent string) (string, error) {
	return aa.CreateSessionForUser(ip, userAgent, "", "", models.RoleAdmin)
}

// CreateSessionForUser creates a new session with user information
func (aa *AdminAuth) CreateSessionForUser(ip, userAgent, userID, userEmail string, role models.Role) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(b)

	aa.mu.Lock()
	aa.sessions[token] = Session{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IP:        ip,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
		UserID:    userID,
		UserEmail: userEmail,
		UserRole:  role,
	}
	aa.saveSessions()
	aa.mu.Unlock()

	return token, nil
}

// InvalidateSession removes a session
func (aa *AdminAuth) InvalidateSession(token string) {
	aa.mu.Lock()
	delete(aa.sessions, token)
	aa.saveSessions()
	aa.mu.Unlock()
}

// LoginHandler handles login requests
func (aa *AdminAuth) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Handle AJAX login
	if r.Header.Get("Content-Type") == "application/json" {
		aa.handleJSONLogin(w, r)
		return
	}

	clientIP := utils.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	// NOTE: lockout is intentionally NOT a hard pre-credential gate. Hard-blocking
	// a locked IP before checking credentials lets an attacker lock the (typically
	// single) admin out of their own or a shared-NAT egress IP — a trivial remote
	// DoS. Instead, correct credentials always pass (below); only wrong credentials
	// are counted and throttled (see the !ok branch, which escalates the delay and
	// shows the locked message). The 5/min rate limiter remains the first throttle.

	if r.Method == "GET" {
		// Generate CSRF token for form
		csrfToken, err := aa.GenerateCSRFToken()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		aa.renderLoginPage(w, "", false, csrfToken)
		return
	}

	// POST - Login attempt

	// Local password auth may be disabled (OIDC-only). Enforce on the endpoint,
	// not just by hiding the form, so a direct POST can't use the password path.
	if !aa.IsLocalAuthAllowed() {
		LogAuditEvent(AuditLoginFailed, clientIP, userAgent, "Local auth disabled (OIDC-only)")
		aa.renderLoginPage(w, "Local login is disabled. Please sign in with SSO.", false)
		return
	}

	// Validate CSRF token
	csrfToken := r.FormValue("csrf_token")
	if !aa.ConsumeCSRFToken(csrfToken) {
		LogAuditEvent(AuditCSRFViolation, clientIP, userAgent, "Invalid CSRF token on login")
		newToken, err := aa.GenerateCSRFToken()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		aa.renderLoginPage(w, "Invalid or expired form. Please try again.", false, newToken)
		return
	}

	// Rate limiting
	if !aa.rateLimiter.Allow(clientIP) {
		LogAuditEvent(AuditRateLimitHit, clientIP, userAgent, "Login rate limit exceeded")
		newToken, err := aa.GenerateCSRFToken()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		aa.renderLoginPage(w, "Too many attempts. Please wait.", false, newToken)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	userID, userEmail, role, ok := aa.resolveLogin(email, password)
	if !ok {
		attempts := aa.RecordFailedAttempt(clientIP)
		// Escalate the throttle once over the lockout threshold so wrong-credential
		// guessing is slowed hard, without ever blocking the correct password.
		if attempts >= MaxFailedAttempts {
			time.Sleep(2 * time.Second)
		} else {
			time.Sleep(500 * time.Millisecond)
		}

		msg := "Invalid credentials"
		if attempts >= MaxFailedAttempts-3 {
			msg = "Invalid password. Warning: Account will be locked after more failed attempts."
		}
		if attempts >= MaxFailedAttempts {
			LogAuditEvent(AuditLoginLocked, clientIP, userAgent, "Account locked after max failed attempts")
			msg = "Account locked due to too many failed attempts. Please try again later."
		} else {
			LogAuditEvent(AuditLoginFailed, clientIP, userAgent, fmt.Sprintf("Failed login attempt %d/%d", attempts, MaxFailedAttempts))
		}

		newToken, err := aa.GenerateCSRFToken()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		aa.renderLoginPage(w, msg, false, newToken)
		return
	}

	// Second factor: the shared-admin password login (no userID) requires a
	// valid TOTP code when 2FA is enabled.
	if userID == "" && role == models.RoleAdmin && aa.IsTOTPEnabled() {
		if !aa.verifyTOTP(r.FormValue("totp")) {
			aa.RecordFailedAttempt(clientIP)
			time.Sleep(500 * time.Millisecond)
			LogAuditEvent(AuditLoginFailed, clientIP, userAgent, "Admin 2FA code missing/invalid")
			newToken, err := aa.GenerateCSRFToken()
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			aa.renderLoginPage(w, "Enter your 6-digit 2FA code.", false, newToken)
			return
		}
	}

	// Successful login - reset failed attempts
	aa.ResetFailedAttempts(clientIP)
	LogAuditEvent(AuditLoginSuccess, clientIP, userAgent, fmt.Sprintf("Successful login (role=%s)", role))

	// Create session with the resolved identity/role
	token, err := aa.CreateSessionForUser(clientIP, userAgent, userID, userEmail, role)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "casadrop_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   utils.IsRequestSecure(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 hours
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

// handleJSONLogin handles AJAX login requests
func (aa *AdminAuth) handleJSONLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		TOTP     string `json:"totp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	clientIP := utils.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	// Local password auth may be disabled (OIDC-only); enforce on the endpoint.
	if !aa.IsLocalAuthAllowed() {
		LogAuditEvent(AuditLoginFailed, clientIP, userAgent, "Local auth disabled (OIDC-only)")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Local login is disabled"})
		return
	}

	// Lockout is not a hard pre-credential gate here either — correct credentials
	// must always be able to log in, otherwise an attacker can DoS-lock the admin
	// from a shared IP. Wrong credentials are counted/throttled in the !ok branch.

	if !aa.rateLimiter.Allow(clientIP) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "Too many attempts"})
		return
	}

	userID, userEmail, role, ok := aa.resolveLogin(req.Email, req.Password)
	if !ok {
		attempts := aa.RecordFailedAttempt(clientIP)
		if attempts >= MaxFailedAttempts {
			time.Sleep(2 * time.Second)
		} else {
			time.Sleep(500 * time.Millisecond)
		}

		msg := "Invalid credentials"
		if attempts >= MaxFailedAttempts {
			LogAuditEvent(AuditLoginLocked, clientIP, userAgent, "Account locked after max failed attempts (JSON)")
			msg = "Account locked due to too many failed attempts. Please try again later."
		} else {
			LogAuditEvent(AuditLoginFailed, clientIP, userAgent, fmt.Sprintf("Failed JSON login attempt %d/%d", attempts, MaxFailedAttempts))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return
	}

	// Second factor for the shared-admin password login when 2FA is enabled.
	if userID == "" && role == models.RoleAdmin && aa.IsTOTPEnabled() {
		if !aa.verifyTOTP(req.TOTP) {
			aa.RecordFailedAttempt(clientIP)
			time.Sleep(500 * time.Millisecond)
			LogAuditEvent(AuditLoginFailed, clientIP, userAgent, "Admin 2FA code missing/invalid (JSON)")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "2FA code required", "totpRequired": "true"})
			return
		}
	}

	// Successful login - reset failed attempts
	aa.ResetFailedAttempts(clientIP)
	LogAuditEvent(AuditLoginSuccess, clientIP, userAgent, fmt.Sprintf("Successful JSON login (role=%s)", role))

	token, err := aa.CreateSessionForUser(clientIP, userAgent, userID, userEmail, role)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "casadrop_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   utils.IsRequestSecure(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// LogoutHandler handles logout requests
func (aa *AdminAuth) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := utils.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	cookie, err := r.Cookie("casadrop_session")
	if err == nil {
		aa.InvalidateSession(cookie.Value)
		LogAuditEvent(AuditLogout, clientIP, userAgent, "User logged out")
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "casadrop_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   utils.IsRequestSecure(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	// JSON response for AJAX
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// SetupHandler handles initial password setup
func (aa *AdminAuth) SetupHandler(w http.ResponseWriter, r *http.Request) {
	// Don't allow setup if env password is set
	if aa.envPassword != "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Don't allow setup if already done
	if aa.config != nil && aa.config.SetupDone {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		csrfToken, err := aa.GenerateCSRFToken()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		aa.renderSetupPage(w, "", csrfToken)
		return
	}

	// POST - Setup

	// Validate CSRF token
	csrfToken := r.FormValue("csrf_token")
	if !aa.ConsumeCSRFToken(csrfToken) {
		newToken, err := aa.GenerateCSRFToken()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		aa.renderSetupPage(w, "Invalid or expired form. Please try again.", newToken)
		return
	}

	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if len(password) < 8 {
		newToken, err := aa.GenerateCSRFToken()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		aa.renderSetupPage(w, "Password must be at least 8 characters", newToken)
		return
	}

	if password != confirmPassword {
		newToken, err := aa.GenerateCSRFToken()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		aa.renderSetupPage(w, "Passwords don't match", newToken)
		return
	}

	if err := aa.SetPassword(password); err != nil {
		newToken, err := aa.GenerateCSRFToken()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		aa.renderSetupPage(w, "Failed to save password", newToken)
		return
	}

	// Auto-login after setup
	clientIP := utils.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")
	LogAuditEvent(AuditSetupComplete, clientIP, userAgent, "Initial admin setup completed")

	token, err := aa.CreateSession(clientIP, userAgent)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "casadrop_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   utils.IsRequestSecure(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

// AuthStatusHandler returns current auth status (for frontend)
func (aa *AdminAuth) AuthStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := struct {
		Authenticated bool   `json:"authenticated"`
		SetupRequired bool   `json:"setupRequired"`
		EnvPassword   bool   `json:"envPassword"`
		SessionExpiry string `json:"sessionExpiry,omitempty"`
	}{
		SetupRequired: aa.NeedsSetup(),
		EnvPassword:   aa.envPassword != "",
	}

	if cookie, err := r.Cookie("casadrop_session"); err == nil {
		// Route through getSession so this honors BOTH the idle expiry and the
		// absolute-lifetime cap — otherwise a session past its absolute TTL would
		// be reported authenticated here while the real middleware rejects it.
		if session := aa.getSession(cookie.Value); session != nil {
			status.Authenticated = true
			status.SessionExpiry = session.ExpiresAt.Format(time.RFC3339)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetSessions returns all active sessions
func (aa *AdminAuth) GetSessions() []Session {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	var sessions []Session
	now := time.Now()
	for _, s := range aa.sessions {
		if now.Before(s.ExpiresAt) {
			// Mask token for security
			s.Token = s.Token[:8] + "..."
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// SetOIDCStatus updates the OIDC configuration status (startup-cached fallback).
// Prefer SetOIDCStatusProvider for a live, runtime-accurate source.
func (aa *AdminAuth) SetOIDCStatus(enabled, localDisabled bool) {
	aa.mu.Lock()
	defer aa.mu.Unlock()
	aa.oidcEnabled = enabled
	aa.oidcLocalDisabled = localDisabled
}

// SetOIDCStatusProvider registers a live status source (the OIDC provider) so
// local-auth decisions always reflect the current runtime config rather than a
// value cached once at startup. Without this, enabling OIDC + disable_local_auth
// at runtime via the admin API would leave the local password login path open
// until the next restart (the UI hides the form but the backend still accepts it).
func (aa *AdminAuth) SetOIDCStatusProvider(fn func() (enabled, localDisabled bool)) {
	aa.mu.Lock()
	aa.oidcStatusFn = fn
	aa.mu.Unlock()
}

// oidcStatus returns the current OIDC status, preferring the live provider.
// The status function is invoked without holding aa.mu to avoid lock-order
// inversions with the provider's own lock.
func (aa *AdminAuth) oidcStatus() (enabled, localDisabled bool) {
	aa.mu.RLock()
	fn := aa.oidcStatusFn
	cachedEnabled := aa.oidcEnabled
	cachedDisabled := aa.oidcLocalDisabled
	aa.mu.RUnlock()
	if fn != nil {
		return fn()
	}
	return cachedEnabled, cachedDisabled
}

// IsOIDCEnabled returns whether OIDC authentication is enabled
func (aa *AdminAuth) IsOIDCEnabled() bool {
	enabled, _ := aa.oidcStatus()
	return enabled
}

// IsLocalAuthAllowed returns whether local password authentication is allowed
func (aa *AdminAuth) IsLocalAuthAllowed() bool {
	// Local auth is allowed if OIDC is disabled, or OIDC is enabled but local
	// auth is not explicitly disabled.
	enabled, localDisabled := aa.oidcStatus()
	return !enabled || !localDisabled
}

func (aa *AdminAuth) renderLoginPage(w http.ResponseWriter, errorMsg string, isSetup bool, csrfToken ...string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	errorHTML := ""
	if errorMsg != "" {
		errorHTML = `<div class="error">` + html.EscapeString(errorMsg) + `</div>`
	}

	// Get CSRF token (optional parameter for backwards compatibility)
	csrf := ""
	if len(csrfToken) > 0 {
		csrf = csrfToken[0]
	}

	// Check OIDC status
	oidcEnabled := aa.IsOIDCEnabled()
	localAuthAllowed := aa.IsLocalAuthAllowed()

	// Build SSO button HTML if OIDC is enabled
	ssoButtonHTML := ""
	if oidcEnabled {
		ssoButtonHTML = `
        <a href="/auth/oidc/login" class="button sso-button">
            <svg viewBox="0 0 24 24" width="20" height="20" style="margin-right: 8px; vertical-align: middle;">
                <path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
            </svg>
            Login with SSO
        </a>`
	}

	// Build password form HTML if local auth is allowed
	passwordFormHTML := ""
	if localAuthAllowed {
		dividerHTML := ""
		if oidcEnabled {
			dividerHTML = `<div class="divider"><span>or</span></div>`
		}
		passwordFormHTML = dividerHTML + `
        <form method="POST" action="/login">
            <input type="hidden" name="csrf_token" value="` + csrf + `">
            <div class="form-group">
                <label for="email">Email <span style="opacity:.6;font-weight:400">(leave blank for admin)</span></label>
                <input type="email" id="email" name="email" placeholder="you@example.com" autocomplete="username">
            </div>
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" placeholder="••••••••" autocomplete="current-password" required>
            </div>
            <div class="form-group">
                <label for="totp">2FA Code <span style="opacity:.6;font-weight:400">(if enabled)</span></label>
                <input type="text" id="totp" name="totp" placeholder="123456" inputmode="numeric" autocomplete="one-time-code" pattern="[0-9]*" maxlength="6">
            </div>
            <button type="submit">Login</button>
        </form>`
	}

	// If only OIDC is enabled (local auth disabled), show different message
	subtitle := "Admin Login"
	if oidcEnabled && !localAuthAllowed {
		subtitle = "Single Sign-On"
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login | CasaDrop</title>
    <link rel="stylesheet" href="/static/css/auth.css">
    <style>
        .sso-button {
            display: flex;
            align-items: center;
            justify-content: center;
            background: #4285f4;
            color: white;
            text-decoration: none;
            padding: 12px 24px;
            border-radius: 6px;
            font-weight: 500;
            transition: background 0.2s;
        }
        .sso-button:hover {
            background: #3367d6;
        }
        .divider {
            display: flex;
            align-items: center;
            margin: 20px 0;
            color: #888;
        }
        .divider::before,
        .divider::after {
            content: '';
            flex: 1;
            height: 1px;
            background: #ddd;
        }
        .divider span {
            padding: 0 16px;
            font-size: 14px;
        }
        @media (prefers-color-scheme: dark) {
            .divider::before,
            .divider::after {
                background: #444;
            }
        }
    </style>
</head>
<body>
    <div class="login-card">
        <div class="logo">
            <img src="/static/logo.png" alt="CasaDrop" style="height:180px;width:auto;margin:-30px auto -10px auto;display:block;">
        </div>
        <p class="subtitle">` + subtitle + `</p>
        ` + errorHTML + ssoButtonHTML + passwordFormHTML + `
    </div>
</body>
</html>`

	w.Write([]byte(html))
}

func (aa *AdminAuth) renderSetupPage(w http.ResponseWriter, errorMsg string, csrfToken ...string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	errorHTML := ""
	if errorMsg != "" {
		errorHTML = `<div class="error">` + html.EscapeString(errorMsg) + `</div>`
	}

	csrf := ""
	if len(csrfToken) > 0 {
		csrf = csrfToken[0]
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Setup | CasaDrop</title>
    <link rel="stylesheet" href="/static/css/auth.css">
</head>
<body>
    <div class="setup-card">
        <div class="logo">
            <img src="/static/logo.png" alt="CasaDrop" style="height:180px;width:auto;margin:-30px auto -10px auto;display:block;">
        </div>
        <h1>Welcome to CasaDrop</h1>
        <p class="subtitle">Initial Setup</p>
        <p class="hint">Create an admin password to secure your file sharing.</p>
        ` + errorHTML + `
        <form method="POST" action="/setup">
            <input type="hidden" name="csrf_token" value="` + csrf + `">
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" placeholder="Min. 8 characters" required minlength="8">
            </div>
            <div class="form-group">
                <label for="confirm_password">Confirm Password</label>
                <input type="password" id="confirm_password" name="confirm_password" placeholder="Repeat password" required>
            </div>
            <button type="submit">Create Admin Account</button>
        </form>
    </div>
</body>
</html>`

	w.Write([]byte(html))
}
