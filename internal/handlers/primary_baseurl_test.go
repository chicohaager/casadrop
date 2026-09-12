package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// writeTunnelConfig drops a tunnel_config.json into the handler's data dir.
func writeTunnelConfig(t *testing.T, cfg TunnelConfig) {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal tunnel config: %v", err)
	}
	path := filepath.Join(os.Getenv("DATA_DIR"), "tunnel_config.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write tunnel config: %v", err)
	}
}

// With primaryNetwork=local, the address configured in Settings must win over
// the LOCAL_IP environment variable. Reading only the env var made the
// Settings field display-only: /api/network echoed the typed address while
// every generated link used the env one (in bridged Docker that is the
// container's own 172.x address, unreachable from the LAN).
func TestGetPrimaryBaseURLLocalPrefersConfig(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	t.Setenv("LOCAL_IP", "172.18.0.5")
	t.Setenv("EXTERNAL_PORT", "8080")
	writeTunnelConfig(t, TunnelConfig{PrimaryNetwork: "local", LocalIP: "192.168.10.20"})

	req := httptest.NewRequest("GET", "/api/upload", nil)
	req.Host = "127.0.0.1:8080" // loopback: must not override the selection

	got := handler.getPrimaryBaseURL(req)
	want := "http://192.168.10.20:8080"
	if got != want {
		t.Errorf("getPrimaryBaseURL = %q, want %q (the Settings value must beat LOCAL_IP)", got, want)
	}
}

// Counter-test: with nothing configured, the env var is still the fallback —
// the fix must not break the container's auto-detection path.
func TestGetPrimaryBaseURLLocalFallsBackToEnv(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	t.Setenv("LOCAL_IP", "192.168.10.77")
	t.Setenv("EXTERNAL_PORT", "8080")
	writeTunnelConfig(t, TunnelConfig{PrimaryNetwork: "local"})

	req := httptest.NewRequest("GET", "/api/upload", nil)
	req.Host = "127.0.0.1:8080"

	got := handler.getPrimaryBaseURL(req)
	want := "http://192.168.10.77:8080"
	if got != want {
		t.Errorf("getPrimaryBaseURL = %q, want %q (env must remain the fallback)", got, want)
	}
}

// The public-host override must still win over the primary-network selection:
// reaching the app through a real hostname should produce links on that host.
func TestGetPrimaryBaseURLPublicHostStillOverrides(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	t.Setenv("LOCAL_IP", "172.18.0.5")
	writeTunnelConfig(t, TunnelConfig{PrimaryNetwork: "local", LocalIP: "192.168.10.20"})

	req := httptest.NewRequest("GET", "/api/upload", nil)
	req.Host = "files.example.com"

	got := handler.getPrimaryBaseURL(req)
	want := "http://files.example.com"
	if got != want {
		t.Errorf("getPrimaryBaseURL = %q, want %q", got, want)
	}
}
