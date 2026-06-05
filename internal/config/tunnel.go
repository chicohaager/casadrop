// Package config provides configuration management for CasaDrop
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DefaultBlockedExtensions contains file extensions blocked by default for security
var DefaultBlockedExtensions = map[string]bool{
	".exe": true, ".bat": true, ".cmd": true, ".com": true, ".msi": true,
	".scr": true, ".pif": true, ".vbs": true, ".vbe": true, ".js": true,
	".jse": true, ".ws": true, ".wsf": true, ".wsc": true, ".wsh": true,
	".ps1": true, ".ps1xml": true, ".ps2": true, ".ps2xml": true,
	".psc1": true, ".psc2": true, ".reg": true, ".inf": true, ".lnk": true,
	".hta": true, ".cpl": true, ".msc": true, ".jar": true,
}

// TunnelConfig represents the user's network and admin configuration
type TunnelConfig struct {
	Enabled        bool   `json:"enabled"`
	URL            string `json:"url"`            // Legacy field, kept for compatibility
	CloudflareURL  string `json:"cloudflareUrl"`  // Cloudflare Tunnel URL (manual override)
	TailscaleURL   string `json:"tailscaleUrl"`   // Tailscale Funnel URL
	EasyTierIP     string `json:"easytierIp"`     // EasyTier IP address
	CustomURL      string `json:"customUrl"`      // Custom URL (WireGuard, Reverse Proxy, etc.)
	LocalIP        string `json:"localIp"`        // Local network IP (manual override)
	PrimaryNetwork string `json:"primaryNetwork"` // Which network to use for share links

	// Enabled flags - when false, network is hidden from display
	// Default to true (nil/missing = enabled) for backwards compatibility
	CloudflareEnabled *bool `json:"cloudflareEnabled,omitempty"`
	TailscaleEnabled  *bool `json:"tailscaleEnabled,omitempty"`
	EasyTierEnabled   *bool `json:"easytierEnabled,omitempty"`
	CustomEnabled     *bool `json:"customEnabled,omitempty"`
	LocalEnabled      *bool `json:"localEnabled,omitempty"`

	// Legacy disabled flags - kept for backwards compatibility
	CloudflareDisabled bool `json:"cloudflareDisabled,omitempty"`
	TailscaleDisabled  bool `json:"tailscaleDisabled,omitempty"`
	EasyTierDisabled   bool `json:"easytierDisabled,omitempty"`
	CustomDisabled     bool `json:"customDisabled,omitempty"`
	LocalDisabled      bool `json:"localDisabled,omitempty"`

	// Admin settings
	MaxFileSizeGB     int    `json:"maxFileSizeGB"`     // Maximum file size in GB (0 = default 10 GB)
	AllowedExtensions string `json:"allowedExtensions"` // Comma-separated allowed extensions
	BlockedExtensions string `json:"blockedExtensions"` // Comma-separated blocked extensions
}

// IsCloudflareEnabled returns true if Cloudflare network is enabled
func (c *TunnelConfig) IsCloudflareEnabled() bool {
	if c.CloudflareEnabled != nil {
		return *c.CloudflareEnabled
	}
	return !c.CloudflareDisabled
}

// IsTailscaleEnabled returns true if Tailscale network is enabled
func (c *TunnelConfig) IsTailscaleEnabled() bool {
	if c.TailscaleEnabled != nil {
		return *c.TailscaleEnabled
	}
	return !c.TailscaleDisabled
}

// IsEasyTierEnabled returns true if EasyTier network is enabled
func (c *TunnelConfig) IsEasyTierEnabled() bool {
	if c.EasyTierEnabled != nil {
		return *c.EasyTierEnabled
	}
	return !c.EasyTierDisabled
}

// IsCustomEnabled returns true if Custom network is enabled
func (c *TunnelConfig) IsCustomEnabled() bool {
	if c.CustomEnabled != nil {
		return *c.CustomEnabled
	}
	return !c.CustomDisabled
}

// IsLocalEnabled returns true if Local network is enabled
func (c *TunnelConfig) IsLocalEnabled() bool {
	if c.LocalEnabled != nil {
		return *c.LocalEnabled
	}
	return !c.LocalDisabled
}

// GetMaxFileSizeBytes returns the maximum file size in bytes
func (c *TunnelConfig) GetMaxFileSizeBytes() int64 {
	if c.MaxFileSizeGB <= 0 {
		return 10 << 30 // Default: 10 GB
	}
	return int64(c.MaxFileSizeGB) << 30
}

// GetBlockedExtensions returns the map of blocked extensions
func (c *TunnelConfig) GetBlockedExtensions() map[string]bool {
	if c.BlockedExtensions != "" {
		blocked := make(map[string]bool)
		for _, ext := range strings.Split(c.BlockedExtensions, ",") {
			ext = strings.TrimSpace(strings.ToLower(ext))
			if ext != "" {
				if !strings.HasPrefix(ext, ".") {
					ext = "." + ext
				}
				blocked[ext] = true
			}
		}
		return blocked
	}
	return DefaultBlockedExtensions
}

// GetAllowedExtensions returns the map of allowed extensions (nil = all allowed)
func (c *TunnelConfig) GetAllowedExtensions() map[string]bool {
	if c.AllowedExtensions == "" {
		return nil
	}
	allowed := make(map[string]bool)
	for _, ext := range strings.Split(c.AllowedExtensions, ",") {
		ext = strings.TrimSpace(strings.ToLower(ext))
		if ext != "" {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			allowed[ext] = true
		}
	}
	return allowed
}

// IsExtensionAllowed checks if a file extension is allowed based on config
func (c *TunnelConfig) IsExtensionAllowed(filename string) (bool, string) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return true, "" // Files without extension are allowed
	}

	// Check blocked extensions first
	blocked := c.GetBlockedExtensions()
	if blocked[ext] {
		return false, fmt.Sprintf("File type %s is blocked for security reasons", ext)
	}

	// If allowed extensions are specified, check against whitelist
	allowed := c.GetAllowedExtensions()
	if allowed != nil && !allowed[ext] {
		return false, fmt.Sprintf("File type %s is not in the allowed extensions list", ext)
	}

	return true, ""
}

// IsFileTypeAllowed is a simple check using default blocked extensions
func IsFileTypeAllowed(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return !DefaultBlockedExtensions[ext]
}

// ConfigManager handles loading and saving TunnelConfig with caching
type ConfigManager struct {
	dataDir   string
	cache     *TunnelConfig
	cacheTime time.Time
	cacheTTL  time.Duration
	mu        sync.RWMutex
}

// NewConfigManager creates a new ConfigManager
func NewConfigManager(dataDir string) *ConfigManager {
	if dataDir == "" {
		dataDir = os.Getenv("DATA_DIR")
		if dataDir == "" {
			dataDir = "./data"
		}
	}
	return &ConfigManager{
		dataDir:  dataDir,
		cacheTTL: 30 * time.Second,
	}
}

// GetDataDir returns the data directory path
func (m *ConfigManager) GetDataDir() string {
	return m.dataDir
}

// GetConfigPath returns the tunnel config file path
func (m *ConfigManager) GetConfigPath() string {
	return filepath.Join(m.dataDir, "tunnel_config.json")
}

// Load loads the tunnel configuration (with caching)
func (m *ConfigManager) Load() (*TunnelConfig, error) {
	m.mu.RLock()
	if m.cache != nil && time.Since(m.cacheTime) < m.cacheTTL {
		config := m.cache
		m.mu.RUnlock()
		return config, nil
	}
	m.mu.RUnlock()

	data, err := os.ReadFile(m.GetConfigPath())
	if err != nil {
		return &TunnelConfig{}, nil // Return empty config if file doesn't exist
	}

	var config TunnelConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return &TunnelConfig{}, nil
	}

	m.mu.Lock()
	m.cache = &config
	m.cacheTime = time.Now()
	m.mu.Unlock()

	return &config, nil
}

// Save saves the tunnel configuration
func (m *ConfigManager) Save(config *TunnelConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(m.GetConfigPath(), data, 0644); err != nil {
		return err
	}

	m.mu.Lock()
	m.cache = config
	m.cacheTime = time.Now()
	m.mu.Unlock()

	return nil
}

// InvalidateCache forces the next Load() to read from disk
func (m *ConfigManager) InvalidateCache() {
	m.mu.Lock()
	m.cache = nil
	m.mu.Unlock()
}
