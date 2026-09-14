package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"casadrop/internal/models"
	"casadrop/internal/storage"
)

// The share dialog posts share_id and recipient_email but never sender_email.
// The handler used to require sender_email, so EVERY email-share failed with
// "share_id, recipient_email, and sender_email are required" even when SMTP was
// correctly configured. sender_email must be optional; the request must reach
// the send stage instead of being rejected at validation.
func TestSendEmailTransferDoesNotRequireSender(t *testing.T) {
	store := newEmailTestStore(t)
	h := NewEmailHandler(store)

	store.Save(&models.Share{
		ID:           "share-1",
		FileName:     "doc.pdf",
		OriginalName: "doc.pdf",
		ExpiresAt:    time.Now().Add(time.Hour),
		CreatedAt:    time.Now(),
	})

	// Exactly what the frontend sends: no sender_email.
	body := `{"share_id":"share-1","recipient_email":"someone@example.com","message":"hi"}`
	rr := httptest.NewRecorder()
	h.SendEmailTransfer(rr, httptest.NewRequest("POST", "/api/email/send", strings.NewReader(body)))

	// SMTP is not configured in this test, so the send itself fails (500). The
	// point is that it is NOT the 400 validation rejection the user hit.
	if rr.Code == 400 {
		t.Fatalf("payload without sender_email was rejected at validation: %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "required") {
		t.Errorf("still emitting a required-fields error: %s", rr.Body.String())
	}
}

func TestSendEmailTransferStillRequiresRecipientAndShare(t *testing.T) {
	store := newEmailTestStore(t)
	h := NewEmailHandler(store)

	for _, body := range []string{
		`{"recipient_email":"a@example.com"}`,  // no share_id
		`{"share_id":"share-1"}`,               // no recipient_email
		`{"share_id":"","recipient_email":""}`, // both empty
	} {
		rr := httptest.NewRecorder()
		h.SendEmailTransfer(rr, httptest.NewRequest("POST", "/api/email/send", strings.NewReader(body)))
		if rr.Code != 400 {
			t.Errorf("payload %q: got %d, want 400", body, rr.Code)
		}
	}
}

func newEmailTestStore(t *testing.T) *storage.Storage {
	t.Helper()
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}
