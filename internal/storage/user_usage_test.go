package storage

import (
	"testing"
	"time"

	"casadrop/internal/models"
)

// TestGetUserUsage verifies the quota accounting rules: uploaded/copied share
// files and received-link bytes count; symlink shares and in-place folder
// shares (which don't consume managed storage) do not; other users' data is
// never included.
func TestGetUserUsage(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	const me = "user-me"
	const other = "user-other"
	future := time.Now().Add(24 * time.Hour)

	save := func(s *models.Share) {
		if err := store.Save(s); err != nil {
			t.Fatalf("save share %s: %v", s.ID, err)
		}
	}

	// Counts: a plain uploaded file (1000) + a copied share-from-path (500).
	save(&models.Share{ID: "up1", FileName: "up1.bin", OriginalName: "up1.bin",
		FileSize: 1000, UserID: me, ExpiresAt: future, CreatedAt: time.Now()})
	save(&models.Share{ID: "cp1", FileName: "cp1.bin", OriginalName: "cp1.bin",
		FileSize: 500, UserID: me, ExpiresAt: future, CreatedAt: time.Now()})

	// Excluded: a symlink share (references host file, no copy)...
	save(&models.Share{ID: "sym", FileName: "sym.bin", OriginalName: "sym.bin",
		FileSize: 9_000_000, IsSymlink: true, UserID: me, ExpiresAt: future, CreatedAt: time.Now()})
	// ...and an in-place folder share.
	save(&models.Share{ID: "dir", FileName: "", OriginalName: "photos",
		FileSize: 8_000_000, IsDirectory: true, UserID: me, ExpiresAt: future, CreatedAt: time.Now()})

	// Excluded: another user's file must not leak into my usage.
	save(&models.Share{ID: "other1", FileName: "o.bin", OriginalName: "o.bin",
		FileSize: 7_000_000, UserID: other, ExpiresAt: future, CreatedAt: time.Now()})

	// Counts: received-link bytes owned by me (2500).
	if err := store.SaveReceiveLink(&models.ReceiveLink{
		ID: "rl1", Name: "inbox", UserID: me, TotalSize: 2500, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("save receive link: %v", err)
	}

	got, err := store.GetUserUsage(me)
	if err != nil {
		t.Fatalf("GetUserUsage: %v", err)
	}
	const want = 1000 + 500 + 2500 // symlink, folder, other-user excluded
	if got != want {
		t.Fatalf("GetUserUsage(me) = %d, want %d", got, want)
	}

	// A user with nothing has zero usage.
	if got, err := store.GetUserUsage("nobody"); err != nil || got != 0 {
		t.Fatalf("GetUserUsage(nobody) = %d, %v; want 0, nil", got, err)
	}
}
