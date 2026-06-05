package auth

import (
	"encoding/json"
	"html"
	"log"
	"net/http"
	"os"
	"strings"

	"casadrop/internal/models"
	"casadrop/internal/utils"
)

// SessionCreator is an interface for creating authenticated sessions.
// This allows the OIDC handlers to integrate with the existing AdminAuth.
type SessionCreator interface {
	CreateSession(ip, userAgent string) (string, error)
	CreateSessionForUser(ip, userAgent, userID, userEmail string, role models.Role) (string, error)
}

// Handlers provides HTTP handlers for OIDC authentication.
type Handlers struct {
	provider       *Provider
	sessionCreator SessionCreator
	userService    *UserService
}

// NewHandlers creates new OIDC HTTP handlers.
func NewHandlers(provider *Provider, sessionCreator SessionCreator) *Handlers {
	return &Handlers{
		provider:       provider,
		sessionCreator: sessionCreator,
	}
}

// SetUserService sets the user service for OIDC user provisioning
func (h *Handlers) SetUserService(userService *UserService) {
	h.userService = userService
}

// LoginHandler initiates the OIDC login flow.
// GET /auth/oidc/login
func (h *Handlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if h.provider == nil || !h.provider.IsEnabled() {
		http.Error(w, "OIDC authentication is not enabled", http.StatusNotFound)
		return
	}

	// Get return URL from query param or default to /
	// Validate to prevent open redirect attacks
	returnURL := r.URL.Query().Get("return")
	if returnURL == "" {
		returnURL = "/"
	} else {
		// Only allow relative URLs starting with /
		if !strings.HasPrefix(returnURL, "/") || strings.HasPrefix(returnURL, "//") {
			returnURL = "/"
		}
		// Block any URL with scheme (http://, https://, javascript:, etc.)
		if strings.Contains(returnURL, ":") {
			returnURL = "/"
		}
	}

	// Generate authorization URL
	authURL, err := h.provider.GenerateAuthURL(returnURL)
	if err != nil {
		log.Printf("OIDC: Failed to generate auth URL: %v", err)
		http.Error(w, "Failed to initiate OIDC login", http.StatusInternalServerError)
		return
	}

	// Redirect to identity provider
	http.Redirect(w, r, authURL, http.StatusFound)
}

// CallbackHandler handles the OIDC callback after authentication.
// GET /auth/oidc/callback
func (h *Handlers) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	if h.provider == nil || !h.provider.IsEnabled() {
		http.Error(w, "OIDC authentication is not enabled", http.StatusNotFound)
		return
	}

	// Check for error from provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		log.Printf("OIDC: Provider returned error: %s - %s", errParam, errDesc)
		h.renderError(w, "Authentication failed: "+errDesc)
		return
	}

	// Get authorization code and state
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		h.renderError(w, "Missing code or state parameter")
		return
	}

	// Exchange code for tokens and validate
	userInfo, returnURL, err := h.provider.ExchangeCode(r.Context(), code, state)
	if err != nil {
		log.Printf("OIDC: Failed to exchange code: %v", err)
		h.renderError(w, "Failed to complete authentication")
		return
	}

	// Log successful authentication
	clientIP := getClientIP(r)
	log.Printf("OIDC: Successful login for %s (%s) from %s",
		userInfo.Email, userInfo.Subject, clientIP)

	// Look up or create user via UserService
	var token string
	var sessionErr error
	if h.userService != nil {
		user, err := h.userService.FindOrCreateOIDCUser(userInfo)
		if err != nil {
			log.Printf("OIDC: Failed to find/create user: %v", err)
			h.renderError(w, "Failed to provision user account")
			return
		}
		if user == nil {
			log.Printf("OIDC: Auto-provisioning disabled and user not found: %s", userInfo.Email)
			h.renderError(w, "User account not found. Please contact an administrator.")
			return
		}
		if !user.IsActive {
			log.Printf("OIDC: User account is disabled: %s", userInfo.Email)
			h.renderError(w, "Your account has been disabled. Please contact an administrator.")
			return
		}

		// Create session with user information
		token, sessionErr = h.sessionCreator.CreateSessionForUser(
			clientIP,
			r.Header.Get("User-Agent"),
			user.ID,
			user.Email,
			user.Role,
		)
		if sessionErr != nil {
			log.Printf("OIDC: Failed to create session: %v", sessionErr)
			h.renderError(w, "Failed to create session")
			return
		}
		log.Printf("OIDC: Created session for user %s (role: %s)", user.Email, user.Role)
	} else {
		// Fallback: create admin session (backward compatibility)
		token, sessionErr = h.sessionCreator.CreateSession(clientIP, r.Header.Get("User-Agent"))
		if sessionErr != nil {
			log.Printf("OIDC: Failed to create session: %v", sessionErr)
			h.renderError(w, "Failed to create session")
			return
		}
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "casadrop_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteLaxMode, // Lax for OIDC redirect flow
		MaxAge:   86400,                // 24 hours
	})

	// Redirect to return URL or home
	if returnURL == "" {
		returnURL = "/"
	}
	http.Redirect(w, r, returnURL, http.StatusFound)
}

// LogoutHandler handles OIDC logout.
// GET /auth/oidc/logout
func (h *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	// Clear session cookie first (match creation flags so browsers honour it)
	http.SetCookie(w, &http.Cookie{
		Name:     "casadrop_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	// If OIDC is enabled, redirect to provider's logout endpoint
	if h.provider.IsEnabled() {
		// Build post-logout redirect URI.
		//
		// We deliberately use r.Host (the server's Host header) and NOT
		// X-Forwarded-Host. A client can freely inject X-Forwarded-Host,
		// so trusting it here would give them an open redirect: they could
		// set X-Forwarded-Host: evil.com and have the IdP bounce victims to
		// their domain after logout. Reverse proxies that terminate TLS
		// should rewrite r.Host to the public hostname, which is what most
		// proxies (nginx, Traefik, Caddy) do by default.
		scheme := "http"
		if isHTTPS {
			scheme = "https"
		}
		postLogoutURI := scheme + "://" + r.Host + "/login"

		logoutURL := h.provider.GetLogoutURL(postLogoutURI)
		if logoutURL != "" {
			http.Redirect(w, r, logoutURL, http.StatusFound)
			return
		}
	}

	// Fallback: redirect to login
	http.Redirect(w, r, "/login", http.StatusFound)
}

// StatusHandler returns the OIDC configuration status.
// GET /api/auth/oidc/status
func (h *Handlers) StatusHandler(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Enabled          bool   `json:"enabled"`
		IssuerURL        string `json:"issuerUrl,omitempty"`
		ClientID         string `json:"clientId,omitempty"`
		DisableLocalAuth bool   `json:"disableLocalAuth"`
		EnvConfigured    bool   `json:"envConfigured"`
	}{
		Enabled:       false,
		EnvConfigured: isEnvConfigured(),
	}

	// Only populate if provider is available
	if h.provider != nil {
		response.Enabled = h.provider.IsEnabled()
		response.DisableLocalAuth = h.provider.IsLocalAuthDisabled()

		if response.Enabled {
			config := h.provider.GetConfig()
			response.IssuerURL = config.IssuerURL
			response.ClientID = config.ClientID
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ConfigHandler handles OIDC configuration (admin only).
// GET /api/auth/oidc/config - Get current config
// POST /api/auth/oidc/config - Update config
func (h *Handlers) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getConfig(w, r)
	case http.MethodPost:
		h.updateConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// isEnvConfigured returns true if OIDC is configured via environment variables
func isEnvConfigured() bool {
	return os.Getenv(EnvOIDCEnabled) == "true" ||
		os.Getenv(EnvOIDCIssuerURL) != "" ||
		os.Getenv(EnvOIDCClientID) != ""
}

func (h *Handlers) getConfig(w http.ResponseWriter, r *http.Request) {
	// Add flag to indicate if config is from env (readonly)
	response := struct {
		Config
		EnvConfigured bool `json:"envConfigured"`
	}{
		EnvConfigured: isEnvConfigured(),
	}

	if h.provider != nil {
		response.Config = h.provider.GetConfig()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handlers) updateConfig(w http.ResponseWriter, r *http.Request) {
	// Block updates when configured via environment variables
	if isEnvConfigured() {
		h.jsonError(w, "OIDC is configured via environment variables and cannot be changed via API", http.StatusForbidden)
		return
	}

	// Check if provider is available
	if h.provider == nil {
		h.jsonError(w, "OIDC provider not initialized", http.StatusServiceUnavailable)
		return
	}

	var config Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Basic validation
	if config.Enabled {
		if config.IssuerURL == "" {
			h.jsonError(w, "Issuer URL is required", http.StatusBadRequest)
			return
		}
		if config.ClientID == "" {
			h.jsonError(w, "Client ID is required", http.StatusBadRequest)
			return
		}
		if config.RedirectURL == "" {
			h.jsonError(w, "Redirect URL is required", http.StatusBadRequest)
			return
		}
	}

	if err := h.provider.SaveConfig(&config); err != nil {
		log.Printf("OIDC: Failed to save config: %v", err)
		h.jsonError(w, "Failed to save configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"enabled": h.provider.IsEnabled(),
	})
}

// renderError renders an error page for OIDC failures.
func (h *Handlers) renderError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication Error | CasaDrop</title>
    <link rel="stylesheet" href="/static/css/auth.css">
</head>
<body>
    <div class="login-card">
        <div class="logo error">
            <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
            </svg>
        </div>
        <h1>Authentication Error</h1>
        <div class="error">` + html.EscapeString(message) + `</div>
        <a href="/login" class="button">Back to Login</a>
    </div>
</body>
</html>`

	w.Write([]byte(html))
}

// jsonError sends a JSON error response.
func (h *Handlers) jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// getClientIP extracts the client IP from the request, honoring the shared
// TRUSTED_PROXY-aware logic so spoofed X-Forwarded-For can't poison audit logs.
func getClientIP(r *http.Request) string {
	return utils.GetClientIP(r)
}
