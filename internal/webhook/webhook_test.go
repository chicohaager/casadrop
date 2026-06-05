package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"casadrop/internal/models"
)

func setupTestService(t *testing.T) (*Service, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "casadrop-webhook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Clear any env vars
	os.Unsetenv("WEBHOOK_URL")
	os.Unsetenv("WEBHOOK_SECRET")
	// Tests deliver to loopback httptest servers; strict SSRF (now on by default)
	// would refuse 127.0.0.1. Opt out, mirroring a homelab LAN webhook receiver.
	t.Setenv("WEBHOOK_STRICT_SSRF", "false")

	service := New(tmpDir)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return service, cleanup
}

func TestNew(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	if service == nil {
		t.Fatal("Expected non-nil service")
	}

	config := service.GetConfig()
	if config.Enabled {
		t.Error("Service should be disabled by default")
	}
}

func TestNewWithEnvVars(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "casadrop-webhook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set env vars
	os.Setenv("WEBHOOK_URL", "https://webhook.example.com/notify")
	os.Setenv("WEBHOOK_SECRET", "test-secret")
	defer os.Unsetenv("WEBHOOK_URL")
	defer os.Unsetenv("WEBHOOK_SECRET")

	service := New(tmpDir)
	config := service.GetConfig()

	if !config.Enabled {
		t.Error("Service should be enabled with WEBHOOK_URL set")
	}
	if config.URL != "https://webhook.example.com/notify" {
		t.Errorf("URL mismatch: got %s", config.URL)
	}
	if config.Secret != "test-secret" {
		t.Errorf("Secret mismatch: got %s", config.Secret)
	}
}

func TestSaveAndGetConfig(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	config := models.WebhookConfig{
		Enabled:        true,
		URL:            "https://example.com/webhook",
		Secret:         "my-secret",
		OnDownload:     true,
		OnLimitReached: true,
		OnExpire:       false,
	}

	err := service.SaveConfig(config)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	retrieved := service.GetConfig()
	if !retrieved.Enabled {
		t.Error("Expected enabled=true")
	}
	if retrieved.URL != config.URL {
		t.Errorf("URL mismatch: got %s, want %s", retrieved.URL, config.URL)
	}
	if retrieved.Secret != config.Secret {
		t.Errorf("Secret mismatch: got %s, want %s", retrieved.Secret, config.Secret)
	}
	if !retrieved.OnDownload {
		t.Error("OnDownload should be true")
	}
	if !retrieved.OnLimitReached {
		t.Error("OnLimitReached should be true")
	}
	if retrieved.OnExpire {
		t.Error("OnExpire should be false")
	}
}

func TestNotifyDownload_Disabled(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	share := &models.Share{
		ID:           "test-share",
		OriginalName: "test.txt",
		Downloads:    1,
	}

	// Should not panic when disabled
	service.NotifyDownload(share, "127.0.0.1", "TestAgent")
}

func TestNotifyDownload_Enabled(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	var received models.WebhookPayload
	var wg sync.WaitGroup
	wg.Add(1)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Enable webhook
	config := models.WebhookConfig{
		Enabled:    true,
		URL:        server.URL,
		OnDownload: true,
	}
	service.SaveConfig(config)

	share := &models.Share{
		ID:           "test-share",
		OriginalName: "test.txt",
		Downloads:    5,
	}

	service.NotifyDownload(share, "192.168.1.1", "Mozilla/5.0")

	// Wait for webhook with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Webhook not received within timeout")
	}

	if received.Event != "download" {
		t.Errorf("Expected event 'download', got %s", received.Event)
	}
	if received.ShareID != "test-share" {
		t.Errorf("Expected shareID 'test-share', got %s", received.ShareID)
	}
	if received.FileName != "test.txt" {
		t.Errorf("Expected fileName 'test.txt', got %s", received.FileName)
	}
	if received.ClientIP != "192.168.1.1" {
		t.Errorf("Expected clientIP '192.168.1.1', got %s", received.ClientIP)
	}
}

func TestNotifyLimitReached_Disabled(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	share := &models.Share{
		ID:           "test-share",
		OriginalName: "test.txt",
		Downloads:    10,
	}

	// Should not panic when disabled
	service.NotifyLimitReached(share)
}

func TestNotifyLimitReached_Enabled(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	var received models.WebhookPayload
	var wg sync.WaitGroup
	wg.Add(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := models.WebhookConfig{
		Enabled:        true,
		URL:            server.URL,
		OnLimitReached: true,
	}
	service.SaveConfig(config)

	share := &models.Share{
		ID:           "limit-share",
		OriginalName: "limited.pdf",
		Downloads:    10,
	}

	service.NotifyLimitReached(share)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Webhook not received within timeout")
	}

	if received.Event != "limit_reached" {
		t.Errorf("Expected event 'limit_reached', got %s", received.Event)
	}
}

func TestNotifyExpire_Disabled(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	share := &models.Share{
		ID:           "test-share",
		OriginalName: "test.txt",
	}

	// Should not panic when disabled
	service.NotifyExpire(share)
}

func TestNotifyExpire_Enabled(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	var received models.WebhookPayload
	var wg sync.WaitGroup
	wg.Add(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := models.WebhookConfig{
		Enabled:  true,
		URL:      server.URL,
		OnExpire: true,
	}
	service.SaveConfig(config)

	share := &models.Share{
		ID:           "expired-share",
		OriginalName: "expired.pdf",
		Downloads:    3,
	}

	service.NotifyExpire(share)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Webhook not received within timeout")
	}

	if received.Event != "expire" {
		t.Errorf("Expected event 'expire', got %s", received.Event)
	}
}

func TestWebhookWithSecret(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	var receivedSignature string
	var wg sync.WaitGroup
	wg.Add(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		receivedSignature = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := models.WebhookConfig{
		Enabled:    true,
		URL:        server.URL,
		Secret:     "my-secret-key",
		OnDownload: true,
	}
	service.SaveConfig(config)

	share := &models.Share{
		ID:           "signed-share",
		OriginalName: "signed.txt",
	}

	service.NotifyDownload(share, "127.0.0.1", "TestAgent")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Webhook not received within timeout")
	}

	// Should have signature header when secret is set
	if receivedSignature == "" {
		t.Error("Expected X-Webhook-Signature header when secret is set")
	}
}

func TestWebhookServerError(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	var wg sync.WaitGroup
	wg.Add(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := models.WebhookConfig{
		Enabled:    true,
		URL:        server.URL,
		OnDownload: true,
	}
	service.SaveConfig(config)

	share := &models.Share{
		ID:           "error-share",
		OriginalName: "error.txt",
	}

	// Should not panic on server error
	service.NotifyDownload(share, "127.0.0.1", "TestAgent")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Webhook request not completed within timeout")
	}
}

func TestConfigPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "casadrop-webhook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create service and save config
	service1 := New(tmpDir)
	config := models.WebhookConfig{
		Enabled:    true,
		URL:        "https://persistent.example.com",
		Secret:     "persistent-secret",
		OnDownload: true,
	}
	service1.SaveConfig(config)

	// Create new service from same dir
	service2 := New(tmpDir)
	loaded := service2.GetConfig()

	if loaded.URL != config.URL {
		t.Errorf("URL not persisted: got %s, want %s", loaded.URL, config.URL)
	}
	if loaded.Secret != config.Secret {
		t.Errorf("Secret not persisted: got %s, want %s", loaded.Secret, config.Secret)
	}
}
