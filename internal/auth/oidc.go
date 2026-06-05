// Package auth provides OIDC/OAuth2 authentication for CasaDrop.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Config holds OIDC configuration.
type Config struct {
	Enabled          bool   `json:"enabled"`
	IssuerURL        string `json:"issuerUrl"`
	ClientID         string `json:"clientId"`
	ClientSecret     string `json:"clientSecret"`
	RedirectURL      string `json:"redirectUrl"`
	Scopes           string `json:"scopes"`
	DisableLocalAuth bool   `json:"disableLocalAuth"`
}

// OIDCState represents a pending OIDC authentication state.
type OIDCState struct {
	State        string    `json:"state"`
	Nonce        string    `json:"nonce"`
	CodeVerifier string    `json:"codeVerifier"` // PKCE verifier bound to this auth request
	ExpiresAt    time.Time `json:"expiresAt"`
	ReturnURL    string    `json:"returnUrl"`
}

// UserInfo contains information from the OIDC ID token.
type UserInfo struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Issuer        string `json:"iss,omitempty"` // Issuer URL for user provisioning
}

// Provider manages OIDC authentication.
type Provider struct {
	config       *Config
	dataDir      string
	provider     *oidc.Provider
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	states       map[string]OIDCState
	mu           sync.RWMutex
	stop         chan struct{}
	stopOnce     sync.Once
}

// Environment variable names for OIDC configuration.
const (
	EnvOIDCEnabled          = "OIDC_ENABLED"
	EnvOIDCIssuerURL        = "OIDC_ISSUER_URL"
	EnvOIDCClientID         = "OIDC_CLIENT_ID"
	EnvOIDCClientSecret     = "OIDC_CLIENT_SECRET"
	EnvOIDCRedirectURL      = "OIDC_REDIRECT_URL"
	EnvOIDCScopes           = "OIDC_SCOPES"
	EnvOIDCDisableLocalAuth = "OIDC_DISABLE_LOCAL_AUTH"
)

// Default values.
const (
	DefaultScopes   = "openid,profile,email"
	StateExpiry     = 10 * time.Minute
	CleanupInterval = 5 * time.Minute
)

// NewProvider creates a new OIDC provider from environment variables.
func NewProvider(dataDir string) (*Provider, error) {
	config := loadConfigFromEnv()

	p := &Provider{
		config:  config,
		dataDir: dataDir,
		states:  make(map[string]OIDCState),
		stop:    make(chan struct{}),
	}

	// Load persisted config (overrides env if not set)
	if err := p.loadConfig(); err != nil {
		log.Printf("OIDC: Could not load config from file: %v", err)
	}

	// Initialize if enabled
	if p.config.Enabled {
		if err := p.initialize(); err != nil {
			return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
		}
	}

	// Start cleanup goroutine
	go p.cleanupLoop()

	return p, nil
}

// loadConfigFromEnv loads OIDC configuration from environment variables.
func loadConfigFromEnv() *Config {
	return &Config{
		Enabled:          os.Getenv(EnvOIDCEnabled) == "true",
		IssuerURL:        os.Getenv(EnvOIDCIssuerURL),
		ClientID:         os.Getenv(EnvOIDCClientID),
		ClientSecret:     os.Getenv(EnvOIDCClientSecret),
		RedirectURL:      os.Getenv(EnvOIDCRedirectURL),
		Scopes:           getEnvOrDefault(EnvOIDCScopes, DefaultScopes),
		DisableLocalAuth: os.Getenv(EnvOIDCDisableLocalAuth) == "true",
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// configPath returns the path to the OIDC config file.
func (p *Provider) configPath() string {
	return filepath.Join(p.dataDir, "oidc_config.json")
}

// loadConfig loads configuration from file.
func (p *Provider) loadConfig() error {
	data, err := os.ReadFile(p.configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var fileConfig Config
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return err
	}

	// File config supplements env config (env takes precedence)
	if p.config.IssuerURL == "" {
		p.config.IssuerURL = fileConfig.IssuerURL
	}
	if p.config.ClientID == "" {
		p.config.ClientID = fileConfig.ClientID
	}
	if p.config.ClientSecret == "" {
		p.config.ClientSecret = fileConfig.ClientSecret
	}
	if p.config.RedirectURL == "" {
		p.config.RedirectURL = fileConfig.RedirectURL
	}
	if !p.config.Enabled && fileConfig.Enabled {
		p.config.Enabled = true
	}

	return nil
}

// SaveConfig persists the OIDC configuration to file.
func (p *Provider) SaveConfig(config *Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = config

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(p.configPath(), data, 0600); err != nil {
		return err
	}

	// Reinitialize if enabled
	if config.Enabled {
		return p.initialize()
	}

	return nil
}

// initialize sets up the OIDC provider and OAuth2 config.
func (p *Provider) initialize() error {
	if p.config.IssuerURL == "" {
		return errors.New("OIDC issuer URL is required")
	}
	if p.config.ClientID == "" {
		return errors.New("OIDC client ID is required")
	}
	if p.config.ClientSecret == "" {
		return errors.New("OIDC client secret is required")
	}
	if p.config.RedirectURL == "" {
		return errors.New("OIDC redirect URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Discover OIDC provider
	provider, err := oidc.NewProvider(ctx, p.config.IssuerURL)
	if err != nil {
		return fmt.Errorf("failed to discover OIDC provider: %w", err)
	}
	p.provider = provider

	// Parse scopes
	scopes := parseScopes(p.config.Scopes)

	// Configure OAuth2
	p.oauth2Config = &oauth2.Config{
		ClientID:     p.config.ClientID,
		ClientSecret: p.config.ClientSecret,
		RedirectURL:  p.config.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	// Configure ID token verifier
	p.verifier = provider.Verifier(&oidc.Config{
		ClientID: p.config.ClientID,
	})

	log.Printf("OIDC: Provider initialized (issuer: %s)", p.config.IssuerURL)
	return nil
}

// parseScopes converts comma-separated scopes to a slice.
func parseScopes(scopeStr string) []string {
	if scopeStr == "" {
		scopeStr = DefaultScopes
	}

	var scopes []string
	for _, s := range strings.Split(scopeStr, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			scopes = append(scopes, s)
		}
	}

	// Ensure openid is always present
	hasOpenID := false
	for _, s := range scopes {
		if s == oidc.ScopeOpenID {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID {
		scopes = append([]string{oidc.ScopeOpenID}, scopes...)
	}

	return scopes
}

// isEnabledLocked checks if OIDC is enabled without acquiring the lock.
// Caller must hold at least RLock.
func (p *Provider) isEnabledLocked() bool {
	return p.config.Enabled && p.provider != nil
}

// IsEnabled returns true if OIDC authentication is enabled.
func (p *Provider) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isEnabledLocked()
}

// IsLocalAuthDisabled returns true if local password auth should be disabled.
func (p *Provider) IsLocalAuthDisabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config.DisableLocalAuth && p.isEnabledLocked()
}

// GetConfig returns the current OIDC configuration (without secret).
func (p *Provider) GetConfig() Config {
	p.mu.RLock()
	defer p.mu.RUnlock()

	config := *p.config
	// Mask client secret for API responses
	if config.ClientSecret != "" {
		config.ClientSecret = "********"
	}
	return config
}

// GenerateAuthURL creates an authorization URL for OIDC login. It returns the
// URL and the generated state value so the caller can bind that state to the
// initiating browser (e.g. via a cookie) and reject callbacks that don't carry
// it back.
func (p *Provider) GenerateAuthURL(returnURL string) (authURL, state string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.config.Enabled || p.oauth2Config == nil {
		return "", "", errors.New("OIDC is not enabled")
	}

	// Generate state and nonce
	state, err = generateRandomString(32)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	nonce, err := generateRandomString(32)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// PKCE: a per-request verifier defeats authorization-code injection/interception
	// even for a confidential client (defense-in-depth alongside state+nonce).
	codeVerifier := oauth2.GenerateVerifier()

	// Store state for validation
	p.states[state] = OIDCState{
		State:        state,
		Nonce:        nonce,
		CodeVerifier: codeVerifier,
		ExpiresAt:    time.Now().Add(StateExpiry),
		ReturnURL:    returnURL,
	}

	// Generate auth URL with nonce and PKCE S256 challenge
	authURL = p.oauth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.S256ChallengeOption(codeVerifier),
	)

	return authURL, state, nil
}

// ExchangeCode exchanges an authorization code for tokens and validates the ID token.
func (p *Provider) ExchangeCode(ctx context.Context, code, state string) (*UserInfo, string, error) {
	p.mu.Lock()
	storedState, exists := p.states[state]
	if exists {
		delete(p.states, state)
	}
	p.mu.Unlock()

	if !exists {
		return nil, "", errors.New("invalid or expired state")
	}

	if time.Now().After(storedState.ExpiresAt) {
		return nil, "", errors.New("state expired")
	}

	// Exchange code for token, presenting the PKCE verifier bound to this state.
	token, err := p.oauth2Config.Exchange(ctx, code,
		oauth2.VerifierOption(storedState.CodeVerifier),
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to exchange code: %w", err)
	}

	// Extract and verify ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, "", errors.New("no id_token in response")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Verify nonce
	var claims struct {
		Nonce string `json:"nonce"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, "", fmt.Errorf("failed to parse claims: %w", err)
	}
	if claims.Nonce != storedState.Nonce {
		return nil, "", errors.New("nonce mismatch")
	}

	// Extract user info
	var userInfo UserInfo
	if err := idToken.Claims(&userInfo); err != nil {
		return nil, "", fmt.Errorf("failed to extract user info: %w", err)
	}

	// Set issuer from ID token for user provisioning
	userInfo.Issuer = idToken.Issuer

	return &userInfo, storedState.ReturnURL, nil
}

// GetLogoutURL returns the OIDC end session URL if available.
func (p *Provider) GetLogoutURL(postLogoutRedirectURI string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.config.Enabled || p.provider == nil {
		return ""
	}

	// Try to get end_session_endpoint from provider metadata
	var claims struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := p.provider.Claims(&claims); err != nil || claims.EndSessionEndpoint == "" {
		return ""
	}

	logoutURL, err := url.Parse(claims.EndSessionEndpoint)
	if err != nil {
		return ""
	}

	q := logoutURL.Query()
	q.Set("client_id", p.config.ClientID)
	if postLogoutRedirectURI != "" {
		q.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	}
	logoutURL.RawQuery = q.Encode()

	return logoutURL.String()
}

// generateRandomString generates a cryptographically secure random string.
// Returns the full base64-URL-encoded representation of `length` random bytes
// (~1.33× `length` characters). Truncating the encoded string drops entropy and
// was a bug in earlier revisions.
func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// cleanupLoop periodically removes expired states. Exits when Stop() is called.
func (p *Provider) cleanupLoop() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanupExpiredStates()
		case <-p.stop:
			return
		}
	}
}

// Stop terminates the background cleanup goroutine. Safe to call multiple times.
func (p *Provider) Stop() {
	if p == nil {
		return
	}
	p.stopOnce.Do(func() {
		close(p.stop)
	})
}

// cleanupExpiredStates removes expired OIDC states.
func (p *Provider) cleanupExpiredStates() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for state, s := range p.states {
		if now.After(s.ExpiresAt) {
			delete(p.states, state)
		}
	}
}
