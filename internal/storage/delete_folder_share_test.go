package storage

import (
	"os"
	"testing"
	"time"

	"casadrop/internal/models"
)

// TestDeleteFolderShareKeepsUploadsDir: folder shares store FileName="" (they
// reference SourcePath in place). Deleting one must NOT touch the uploads
// directory — filepath.Join(uploadsDir, "") IS the uploads directory, and the
// old code os.Remove()d it whenever it happened to be empty (fresh instance,
// folder share as first action), breaking every subsequent upload with ENOENT.
func TestDeleteFolderShareKeepsUploadsDir(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	share := &models.Share{
		ID:           "dirshare1",
		FileName:     "", // folder shares have no managed file
		OriginalName: "photos",
		IsDirectory:  true,
		SourcePath:   "/tmp/does-not-matter",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
	}
	if err := store.Save(share); err != nil {
		t.Fatalf("save folder share: %v", err)
	}

	uploadsDir := store.UploadsDir()
	if _, err := os.Stat(uploadsDir); err != nil {
		t.Fatalf("uploads dir missing before delete: %v", err)
	}

	if err := store.Delete("dirshare1"); err != nil {
		t.Fatalf("delete folder share: %v", err)
	}

	if _, err := os.Stat(uploadsDir); err != nil {
		t.Fatalf("uploads dir was removed by deleting a folder share: %v", err)
	}
	if _, ok := store.Get("dirshare1"); ok {
		t.Fatal("share row should be gone after delete")
	}
}
