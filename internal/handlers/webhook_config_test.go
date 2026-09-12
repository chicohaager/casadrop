package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// postWebhookConfig replays a raw JSON body against the config endpoint, the
// way the browser does, and returns the decoded GET that follows.
func postWebhookConfig(t *testing.T, h *Handler, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.WebhookConfig(rec, req)
	return rec.Code, rec.Body.String()
}

func getWebhookConfig(t *testing.T, h *Handler) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	h.WebhookConfig(rec, httptest.NewRequest("GET", "/api/webhook", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/webhook: expected 200, got %d", rec.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("GET /api/webhook: bad JSON: %v", err)
	}
	return out
}

// A partial write must not reset the fields it does not mention. The settings
// form posts only {url, secret}; decoding that straight into a zero-valued
// WebhookConfig set enabled:false / on_download:false, so saving the form
// disarmed the very webhook it was configuring.
func TestWebhookConfigPartialUpdateKeepsUnsentFields(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	if code, body := postWebhookConfig(t, handler, `{
		"enabled": true,
		"url": "https://hooks.example.com/casadrop",
		"on_download": true,
		"on_limit_reached": true,
		"on_expire": true,
		"secret": "s3cr3t"
	}`); code != http.StatusOK {
		t.Fatalf("initial save: expected 200, got %d: %s", code, body)
	}

	// Exactly what the old form sent.
	if code, body := postWebhookConfig(t, handler,
		`{"url":"https://hooks.example.com/casadrop"}`); code != http.StatusOK {
		t.Fatalf("partial save: expected 200, got %d: %s", code, body)
	}

	got := getWebhookConfig(t, handler)
	for _, field := range []string{"enabled", "on_download", "on_limit_reached", "on_expire"} {
		if got[field] != true {
			t.Errorf("%s: want true after a partial save, got %v — an unsent field was reset", field, got[field])
		}
	}
}

// The GET withholds the secret, so the form field is always blank; an omitted
// secret must therefore mean "keep", never "erase".
func TestWebhookConfigOmittedSecretIsPreserved(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	postWebhookConfig(t, handler, `{"enabled":true,"url":"https://hooks.example.com/x","secret":"s3cr3t"}`)
	if got := getWebhookConfig(t, handler); got["secret_set"] != true {
		t.Fatalf("secret_set: want true right after storing a secret, got %v", got["secret_set"])
	}

	// Save again without the secret key — the reloaded-form case.
	postWebhookConfig(t, handler, `{"enabled":true,"url":"https://hooks.example.com/x"}`)
	if got := getWebhookConfig(t, handler); got["secret_set"] != true {
		t.Error("secret_set: want true — an omitted secret erased the stored one")
	}
	if handler.webhook.GetConfig().Secret != "s3cr3t" {
		t.Errorf("stored secret: want %q, got %q", "s3cr3t", handler.webhook.GetConfig().Secret)
	}
}

// Counter-test: the omit-means-keep rule must not make the secret
// unremovable. An explicit empty string still clears it.
func TestWebhookConfigExplicitEmptySecretClears(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	postWebhookConfig(t, handler, `{"enabled":true,"url":"https://hooks.example.com/x","secret":"s3cr3t"}`)
	postWebhookConfig(t, handler, `{"enabled":true,"url":"https://hooks.example.com/x","secret":""}`)

	if got := getWebhookConfig(t, handler); got["secret_set"] != false {
		t.Error("secret_set: want false — an explicit empty secret must clear the stored one")
	}
	if s := handler.webhook.GetConfig().Secret; s != "" {
		t.Errorf("stored secret: want empty, got %q", s)
	}
}

// The read side must never hand the secret back out.
func TestWebhookConfigGETNeverLeaksSecret(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	postWebhookConfig(t, handler, `{"enabled":true,"url":"https://hooks.example.com/x","secret":"top-secret-value"}`)

	rec := httptest.NewRecorder()
	handler.WebhookConfig(rec, httptest.NewRequest("GET", "/api/webhook", nil))
	if strings.Contains(rec.Body.String(), "top-secret-value") {
		t.Errorf("GET response leaked the secret: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"secret_set":true`) {
		t.Errorf("GET response should report secret_set:true, got %s", rec.Body.String())
	}
}

// Arming a webhook with no target would report "saved" and then drop every
// event silently.
func TestWebhookConfigRejectsEnabledWithoutURL(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	code, body := postWebhookConfig(t, handler, `{"enabled":true,"url":""}`)
	if code != http.StatusBadRequest {
		t.Errorf("expected 400 for enabled-without-URL, got %d: %s", code, body)
	}
}

// The reported symptom, end to end: configure the webhook, then save again the
// way the settings form does (URL only), then press Test. Before the fix the
// second save disarmed the webhook and Test answered 400 "Webhook not
// configured" — so this must exercise the PARTIAL save, not a complete one.
func TestWebhookTestEndpointWorksAfterConfiguring(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	postWebhookConfig(t, handler, `{"enabled":true,"url":"https://hooks.example.com/x","on_download":true}`)
	// What the settings form posts.
	postWebhookConfig(t, handler, `{"url":"https://hooks.example.com/x"}`)

	rec := httptest.NewRecorder()
	handler.TestWebhook(rec, httptest.NewRequest("POST", "/api/webhook/test", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("TestWebhook after a valid save: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// The SSRF guard must still reject a private target on save.
func TestWebhookConfigStillRejectsPrivateTarget(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	os.Unsetenv("STRICT_WEBHOOK_URLS")
	code, body := postWebhookConfig(t, handler, `{"enabled":true,"url":"http://192.0.2.1/hook"}`)
	if code != http.StatusOK {
		t.Fatalf("a public example address must be accepted, got %d: %s", code, body)
	}
	code, body = postWebhookConfig(t, handler, `{"enabled":true,"url":"http://127.0.0.1:9000/hook"}`)
	if code != http.StatusBadRequest {
		t.Errorf("loopback target: expected 400, got %d: %s", code, body)
	}
}
