package storage

import (
	"sync"
	"testing"
	"time"

	"casadrop/internal/models"
)

// newTestSQLite builds a throwaway SQLite backend in a temp dir.
func newTestSQLite(t *testing.T) *SQLiteStorage {
	t.Helper()
	s, err := NewSQLiteStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewSQLiteStorage: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// The expiry cleanup is the only possible producer of the "expire" webhook
// event. Before this callback existed, webhook.NotifyExpire had no caller
// anywhere in the tree, so on_expire was a flag that could be set and never
// did anything.
func TestCleanupExpiredSharesNotifies(t *testing.T) {
	s := newTestSQLite(t)

	var mu sync.Mutex
	var seen []*models.Share
	s.SetExpiryNotifier(func(sh *models.Share) {
		mu.Lock()
		defer mu.Unlock()
		seen = append(seen, sh)
	})

	expired := &models.Share{
		ID:           "gone-1",
		FileName:     "gone-1.txt",
		OriginalName: "invoice.pdf",
		FileSize:     42,
		Downloads:    3,
		ExpiresAt:    time.Now().Add(-2 * time.Hour),
		CreatedAt:    time.Now().Add(-3 * time.Hour),
	}
	alive := &models.Share{
		ID:           "keep-1",
		FileName:     "keep-1.txt",
		OriginalName: "keep.txt",
		ExpiresAt:    time.Now().Add(2 * time.Hour),
		CreatedAt:    time.Now(),
	}
	for _, sh := range []*models.Share{expired, alive} {
		if err := s.Save(sh); err != nil {
			t.Fatalf("Save(%s): %v", sh.ID, err)
		}
	}

	s.cleanupExpiredShares()

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 1 {
		t.Fatalf("expected exactly 1 expiry notification, got %d: %+v", len(seen), seen)
	}
	if seen[0].ID != "gone-1" {
		t.Errorf("notified share ID = %q, want %q", seen[0].ID, "gone-1")
	}
	// The webhook payload is built from these two fields, so an empty
	// OriginalName would ship a nameless notification.
	if seen[0].OriginalName != "invoice.pdf" {
		t.Errorf("notified OriginalName = %q, want %q", seen[0].OriginalName, "invoice.pdf")
	}
	if seen[0].Downloads != 3 {
		t.Errorf("notified Downloads = %d, want 3", seen[0].Downloads)
	}
	// And the unexpired share must survive untouched.
	if _, ok := s.Get("keep-1"); !ok {
		t.Error("an unexpired share was deleted by the cleanup")
	}
}

// No notifier registered must not panic — the callback is optional.
func TestCleanupExpiredSharesWithoutNotifier(t *testing.T) {
	s := newTestSQLite(t)

	if err := s.Save(&models.Share{
		ID: "gone-2", FileName: "gone-2.txt", OriginalName: "x.txt",
		ExpiresAt: time.Now().Add(-time.Hour), CreatedAt: time.Now().Add(-2 * time.Hour),
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	s.cleanupExpiredShares() // must not panic

	if _, ok := s.Get("gone-2"); ok {
		t.Error("expired share should have been deleted")
	}
}

// The facade must report whether the callback was actually wired, so a caller
// can tell "registered" from "silently dropped".
func TestStorageSetExpiryNotifierReportsWiring(t *testing.T) {
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if !st.SetExpiryNotifier(func(*models.Share) {}) {
		t.Error("SetExpiryNotifier returned false for the SQLite backend, which supports it")
	}
}
