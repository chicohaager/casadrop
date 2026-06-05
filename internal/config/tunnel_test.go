package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTunnelConfig_IsEnabled(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		config   TunnelConfig
		check    func(*TunnelConfig) bool
		expected bool
	}{
		{
			name:     "Cloudflare enabled by pointer",
			config:   TunnelConfig{CloudflareEnabled: &trueVal},
			check:    func(c *TunnelConfig) bool { return c.IsCloudflareEnabled() },
			expected: true,
		},
		{
			name:     "Cloudflare disabled by pointer",
			config:   TunnelConfig{CloudflareEnabled: &falseVal},
			check:    func(c *TunnelConfig) bool { return c.IsCloudflareEnabled() },
			expected: false,
		},
		{
			name:     "Cloudflare disabled by legacy flag",
			config:   TunnelConfig{CloudflareDisabled: true},
			check:    func(c *TunnelConfig) bool { return c.IsCloudflareEnabled() },
			expected: false,
		},
		{
			name:     "Cloudflare default enabled",
			config:   TunnelConfig{},
			check:    func(c *TunnelConfig) bool { return c.IsCloudflareEnabled() },
			expected: true,
		},
		{
			name:     "Tailscale enabled",
			config:   TunnelConfig{TailscaleEnabled: &trueVal},
			check:    func(c *TunnelConfig) bool { return c.IsTailscaleEnabled() },
			expected: true,
		},
		{
			name:     "EasyTier enabled",
			config:   TunnelConfig{EasyTierEnabled: &trueVal},
			check:    func(c *TunnelConfig) bool { return c.IsEasyTierEnabled() },
			expected: true,
		},
		{
			name:     "Custom enabled",
			config:   TunnelConfig{CustomEnabled: &trueVal},
			check:    func(c *TunnelConfig) bool { return c.IsCustomEnabled() },
			expected: true,
		},
		{
			name:     "Local enabled",
			config:   TunnelConfig{LocalEnabled: &trueVal},
			check:    func(c *TunnelConfig) bool { return c.IsLocalEnabled() },
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check(&tt.config)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTunnelConfig_GetMaxFileSizeBytes(t *testing.T) {
	tests := []struct {
		name     string
		sizeGB   int
		expected int64
	}{
		{"Default (0 GB)", 0, 10 << 30},
		{"1 GB", 1, 1 << 30},
		{"5 GB", 5, 5 << 30},
		{"100 GB", 100, 100 << 30},
		{"Negative", -1, 10 << 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := TunnelConfig{MaxFileSizeGB: tt.sizeGB}
			result := config.GetMaxFileSizeBytes()
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestTunnelConfig_GetBlockedExtensions(t *testing.T) {
	tests := []struct {
		name            string
		blockedStr      string
		checkExtension  string
		shouldBeBlocked bool
	}{
		{"Default blocks .exe", "", ".exe", true},
		{"Default blocks .bat", "", ".bat", true},
		{"Default allows .pdf", "", ".pdf", false},
		{"Custom blocks .pdf", "pdf,doc", ".pdf", true},
		{"Custom blocks with dot", ".pdf,.doc", ".pdf", true},
		{"Custom allows .exe", "pdf,doc", ".exe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := TunnelConfig{BlockedExtensions: tt.blockedStr}
			blocked := config.GetBlockedExtensions()
			isBlocked := blocked[tt.checkExtension]
			if isBlocked != tt.shouldBeBlocked {
				t.Errorf("Extension %s: expected blocked=%v, got %v", tt.checkExtension, tt.shouldBeBlocked, isBlocked)
			}
		})
	}
}

func TestTunnelConfig_GetAllowedExtensions(t *testing.T) {
	tests := []struct {
		name           string
		allowedStr     string
		checkExtension string
		shouldBeNil    bool
		shouldAllow    bool
	}{
		{"Empty returns nil", "", ".pdf", true, false},
		{"Custom allows .pdf", "pdf,jpg", ".pdf", false, true},
		{"Custom allows with dot", ".pdf,.jpg", ".pdf", false, true},
		{"Custom denies .exe", "pdf,jpg", ".exe", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := TunnelConfig{AllowedExtensions: tt.allowedStr}
			allowed := config.GetAllowedExtensions()

			if tt.shouldBeNil {
				if allowed != nil {
					t.Errorf("Expected nil, got %v", allowed)
				}
				return
			}

			if allowed == nil {
				t.Errorf("Expected non-nil map")
				return
			}

			isAllowed := allowed[tt.checkExtension]
			if isAllowed != tt.shouldAllow {
				t.Errorf("Extension %s: expected allowed=%v, got %v", tt.checkExtension, tt.shouldAllow, isAllowed)
			}
		})
	}
}

func TestTunnelConfig_IsExtensionAllowed(t *testing.T) {
	tests := []struct {
		name     string
		config   TunnelConfig
		filename string
		allowed  bool
	}{
		{"Normal PDF allowed", TunnelConfig{}, "document.pdf", true},
		{"EXE blocked by default", TunnelConfig{}, "program.exe", false},
		{"No extension allowed", TunnelConfig{}, "README", true},
		{"Whitelist allows only PDF", TunnelConfig{AllowedExtensions: "pdf"}, "doc.pdf", true},
		{"Whitelist blocks JPG", TunnelConfig{AllowedExtensions: "pdf"}, "image.jpg", false},
		{"Custom blocked", TunnelConfig{BlockedExtensions: "pdf"}, "doc.pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, _ := tt.config.IsExtensionAllowed(tt.filename)
			if allowed != tt.allowed {
				t.Errorf("IsExtensionAllowed(%q) = %v, want %v", tt.filename, allowed, tt.allowed)
			}
		})
	}
}

func TestIsFileTypeAllowed(t *testing.T) {
	tests := []struct {
		filename string
		allowed  bool
	}{
		{"document.pdf", true},
		{"image.jpg", true},
		{"program.exe", false},
		{"script.bat", false},
		{"script.ps1", false},
		{"archive.jar", false},
		{"README", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsFileTypeAllowed(tt.filename)
			if result != tt.allowed {
				t.Errorf("IsFileTypeAllowed(%q) = %v, want %v", tt.filename, result, tt.allowed)
			}
		})
	}
}

func TestConfigManager_LoadSave(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "casadrop-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewConfigManager(tmpDir)

	// Test loading non-existent config returns empty
	config, err := manager.Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}
	if config == nil {
		t.Error("Load() returned nil config")
	}

	// Test saving and loading
	testConfig := &TunnelConfig{
		Enabled:       true,
		CloudflareURL: "https://test.example.com",
		MaxFileSizeGB: 5,
	}

	if err := manager.Save(testConfig); err != nil {
		t.Errorf("Save() error = %v", err)
	}

	// Invalidate cache to force reload
	manager.InvalidateCache()

	loaded, err := manager.Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}

	if loaded.Enabled != testConfig.Enabled {
		t.Errorf("Enabled = %v, want %v", loaded.Enabled, testConfig.Enabled)
	}
	if loaded.CloudflareURL != testConfig.CloudflareURL {
		t.Errorf("CloudflareURL = %v, want %v", loaded.CloudflareURL, testConfig.CloudflareURL)
	}
	if loaded.MaxFileSizeGB != testConfig.MaxFileSizeGB {
		t.Errorf("MaxFileSizeGB = %v, want %v", loaded.MaxFileSizeGB, testConfig.MaxFileSizeGB)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, "tunnel_config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestConfigManager_Caching(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "casadrop-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewConfigManager(tmpDir)

	// Save initial config
	config1 := &TunnelConfig{MaxFileSizeGB: 5}
	manager.Save(config1)

	// Load (should cache)
	loaded1, _ := manager.Load()
	if loaded1.MaxFileSizeGB != 5 {
		t.Errorf("First load: MaxFileSizeGB = %v, want 5", loaded1.MaxFileSizeGB)
	}

	// Modify file directly
	config2 := &TunnelConfig{MaxFileSizeGB: 10}
	manager.Save(config2)

	// Load again (should return cached value due to TTL)
	// Actually, Save updates the cache, so this should return 10
	loaded2, _ := manager.Load()
	if loaded2.MaxFileSizeGB != 10 {
		t.Errorf("Second load: MaxFileSizeGB = %v, want 10", loaded2.MaxFileSizeGB)
	}
}
