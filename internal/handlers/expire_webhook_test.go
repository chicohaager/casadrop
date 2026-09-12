package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"casadrop/internal/models"
)

// End-to-end wiring for the "expire" webhook event: an expired share swept by
// the storage cleanup must reach the configured webhook receiver.
//
// This is deliberately the whole chain, not the callback alone. The defect it
// guards was purely a missing connection — webhook.NotifyExpire existed, was
// unit-tested, and had no caller anywhere in the tree, so on_expire could be
// set to true and nothing ever happened. A test of either half on its own
// stays green through exactly that bug.
func TestExpireEventReachesWebhookReceiver(t *testing.T) {
	t.Setenv("STRICT_WEBHOOK_URLS", "false")
	t.Setenv("WEBHOOK_STRICT_SSRF", "false")

	var mu sync.Mutex
	var payloads []models.WebhookPayload
	var events []string
	done := make(chan struct{}, 4)

	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p models.WebhookPayload
		_ = json.Unmarshal(body, &p)
		mu.Lock()
		payloads = append(payloads, p)
		events = append(events, r.Header.Get("X-Webhook-Event"))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		done <- struct{}{}
	}))
	defer receiver.Close()

	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	// Arm the webhook for expiry events only, so a stray download event can't
	// be mistaken for the thing under test.
	req := httptest.NewRequest("POST", "/api/webhook", strings.NewReader(
		`{"enabled":true,"url":"`+receiver.URL+`","on_expire":true,"on_download":false,"on_limit_reached":false}`))
	rec := httptest.NewRecorder()
	handler.WebhookConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("configuring the webhook: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := handler.storage.Save(&models.Share{
		ID:           "expired-1",
		FileName:     "expired-1.txt",
		OriginalName: "quarterly.pdf",
		Downloads:    7,
		ExpiresAt:    time.Now().Add(-time.Hour),
		CreatedAt:    time.Now().Add(-2 * time.Hour),
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if !handler.storage.RunExpiryCleanup() {
		t.Fatal("storage backend does not support an on-demand expiry sweep")
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("no webhook delivery within 5s — the expire event is not wired to the expiry cleanup")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(payloads) != 1 {
		t.Fatalf("expected exactly 1 delivery, got %d", len(payloads))
	}
	if payloads[0].Event != "expire" {
		t.Errorf("payload event = %q, want %q", payloads[0].Event, "expire")
	}
	if events[0] != "expire" {
		t.Errorf("X-Webhook-Event = %q, want %q", events[0], "expire")
	}
	if payloads[0].ShareID != "expired-1" {
		t.Errorf("share_id = %q, want %q", payloads[0].ShareID, "expired-1")
	}
	if payloads[0].FileName != "quarterly.pdf" {
		t.Errorf("file_name = %q, want %q", payloads[0].FileName, "quarterly.pdf")
	}
	if payloads[0].Downloads != 7 {
		t.Errorf("downloads = %d, want 7", payloads[0].Downloads)
	}
}

// Counter-test: with on_expire off, the sweep must stay silent. Without this,
// a handler that fired unconditionally would pass the test above.
func TestExpireEventNotSentWhenDisabled(t *testing.T) {
	t.Setenv("STRICT_WEBHOOK_URLS", "false")
	t.Setenv("WEBHOOK_STRICT_SSRF", "false")

	var mu sync.Mutex
	hits := 0
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/api/webhook", strings.NewReader(
		`{"enabled":true,"url":"`+receiver.URL+`","on_expire":false,"on_download":true}`))
	rec := httptest.NewRecorder()
	handler.WebhookConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("configuring the webhook: got %d: %s", rec.Code, rec.Body.String())
	}

	if err := handler.storage.Save(&models.Share{
		ID: "expired-2", FileName: "expired-2.txt", OriginalName: "x.pdf",
		ExpiresAt: time.Now().Add(-time.Hour), CreatedAt: time.Now().Add(-2 * time.Hour),
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	handler.storage.RunExpiryCleanup()

	time.Sleep(500 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if hits != 0 {
		t.Errorf("expected no delivery with on_expire=false, got %d", hits)
	}
}
