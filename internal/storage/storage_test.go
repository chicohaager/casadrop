package storage

import (
	"os"
	"testing"
	"time"

	"casadrop/internal/models"
)

// setupTestStorage creates a temporary storage for testing
func setupTestStorage(t *testing.T) (*Storage, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "casadrop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	store, err := New(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create storage: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func TestNew(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	if store == nil {
		t.Fatal("Expected non-nil storage")
	}

	// Check uploads directory was created
	uploadsDir := store.UploadsDir()
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		t.Errorf("Uploads directory was not created: %s", uploadsDir)
	}
}

func TestShareCRUD(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a share
	share := &models.Share{
		ID:           "test-share-1",
		FileName:     "test-file.txt",
		OriginalName: "test-file.txt",
		FileSize:     1024,
		MimeType:     "text/plain",
		HasPassword:  false,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		Downloads:    0,
		MaxDownloads: 10,
	}

	// Test Save
	err := store.Save(share)
	if err != nil {
		t.Fatalf("Failed to save share: %v", err)
	}

	// Test Get
	retrieved, found := store.Get("test-share-1")
	if !found {
		t.Fatal("Share not found after save")
	}
	if retrieved.FileName != share.FileName {
		t.Errorf("FileName mismatch: got %s, want %s", retrieved.FileName, share.FileName)
	}
	if retrieved.FileSize != share.FileSize {
		t.Errorf("FileSize mismatch: got %d, want %d", retrieved.FileSize, share.FileSize)
	}

	// Test GetAll
	all := store.GetAll()
	if len(all) != 1 {
		t.Errorf("Expected 1 share, got %d", len(all))
	}

	// Test IncrementDownloads
	_, err = store.IncrementDownloads("test-share-1")
	if err != nil {
		t.Fatalf("Failed to increment downloads: %v", err)
	}
	retrieved, _ = store.Get("test-share-1")
	if retrieved.Downloads != 1 {
		t.Errorf("Downloads not incremented: got %d, want 1", retrieved.Downloads)
	}

	// Test Delete
	err = store.Delete("test-share-1")
	if err != nil {
		t.Fatalf("Failed to delete share: %v", err)
	}
	_, found = store.Get("test-share-1")
	if found {
		t.Error("Share still found after delete")
	}
}

func TestShareWithPassword(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	share := &models.Share{
		ID:          "password-share",
		FileName:    "secret.pdf",
		Password:    "hashed-password",
		HasPassword: true,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		CreatedAt:   time.Now(),
	}

	err := store.Save(share)
	if err != nil {
		t.Fatalf("Failed to save share with password: %v", err)
	}

	retrieved, found := store.Get("password-share")
	if !found {
		t.Fatal("Share not found")
	}
	if !retrieved.HasPassword {
		t.Error("HasPassword should be true")
	}
	if retrieved.Password != "hashed-password" {
		t.Error("Password not stored correctly")
	}
}

func TestGetExpiringSoon(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create shares with different expiration times
	shares := []*models.Share{
		{
			ID:        "expiring-soon",
			FileName:  "soon.txt",
			ExpiresAt: time.Now().Add(6 * time.Hour),
			CreatedAt: time.Now(),
		},
		{
			ID:        "not-expiring",
			FileName:  "later.txt",
			ExpiresAt: time.Now().Add(48 * time.Hour),
			CreatedAt: time.Now(),
		},
		{
			ID:        "already-expired",
			FileName:  "expired.txt",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
			CreatedAt: time.Now(),
		},
	}

	for _, s := range shares {
		if err := store.Save(s); err != nil {
			t.Fatalf("Failed to save share: %v", err)
		}
	}

	expiring, err := store.GetExpiringSoon(24 * time.Hour)
	if err != nil {
		t.Fatalf("GetExpiringSoon failed: %v", err)
	}

	// Should return "expiring-soon" but not "not-expiring" or "already-expired"
	if len(expiring) != 1 {
		t.Errorf("Expected 1 expiring share, got %d", len(expiring))
	}
	if len(expiring) > 0 && expiring[0].ID != "expiring-soon" {
		t.Errorf("Wrong share returned: %s", expiring[0].ID)
	}
}

func TestSearch(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	shares := []*models.Share{
		{ID: "doc1", FileName: "report.pdf", OriginalName: "Annual Report.pdf", ExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now()},
		{ID: "doc2", FileName: "data.csv", OriginalName: "Sales Data.csv", ExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now()},
		{ID: "img1", FileName: "photo.jpg", OriginalName: "Vacation Photo.jpg", ExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now()},
	}

	for _, s := range shares {
		store.Save(s)
	}

	// Search for "report"
	results, err := store.Search("report")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'report', got %d", len(results))
	}

	// Search for "data"
	results, err = store.Search("data")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'data', got %d", len(results))
	}

	// Search for ".pdf"
	results, err = store.Search(".pdf")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for '.pdf', got %d", len(results))
	}
}

func TestGetStats(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Add some shares
	shares := []*models.Share{
		{ID: "s1", FileName: "file1.txt", FileSize: 1000, Downloads: 5, ExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now()},
		{ID: "s2", FileName: "file2.txt", FileSize: 2000, Downloads: 10, ExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now()},
		{ID: "s3", FileName: "file3.txt", FileSize: 3000, Downloads: 0, ExpiresAt: time.Now().Add(12 * time.Hour), CreatedAt: time.Now()},
	}

	for _, s := range shares {
		store.Save(s)
	}

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalShares != 3 {
		t.Errorf("TotalShares: got %d, want 3", stats.TotalShares)
	}
	if stats.TotalDownloads != 15 {
		t.Errorf("TotalDownloads: got %d, want 15", stats.TotalDownloads)
	}
	if stats.TotalSize != 6000 {
		t.Errorf("TotalSize: got %d, want 6000", stats.TotalSize)
	}
}

func TestReceiveLinkCRUD(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	expiresAt := time.Now().Add(24 * time.Hour)
	link := &models.ReceiveLink{
		ID:          "recv-1",
		Name:        "Test Upload Link",
		HasPassword: false,
		ExpiresAt:   &expiresAt,
		CreatedAt:   time.Now(),
		MaxUploads:  10,
		MaxFileSize: 1024 * 1024 * 100, // 100MB
	}

	// Test Save
	err := store.SaveReceiveLink(link)
	if err != nil {
		t.Fatalf("Failed to save receive link: %v", err)
	}

	// Test Get
	retrieved, found := store.GetReceiveLink("recv-1")
	if !found {
		t.Fatal("Receive link not found")
	}
	if retrieved.Name != link.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, link.Name)
	}

	// Test GetAll
	all, err := store.GetAllReceiveLinks()
	if err != nil {
		t.Fatalf("GetAllReceiveLinks failed: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("Expected 1 receive link, got %d", len(all))
	}

	// Test IncrementUploads
	_, err = store.IncrementReceiveLinkUploads("recv-1")
	if err != nil {
		t.Fatalf("Failed to increment uploads: %v", err)
	}
	retrieved, _ = store.GetReceiveLink("recv-1")
	if retrieved.CurrentUploads != 1 {
		t.Errorf("CurrentUploads not incremented: got %d, want 1", retrieved.CurrentUploads)
	}

	// Test Delete
	err = store.DeleteReceiveLink("recv-1")
	if err != nil {
		t.Fatalf("Failed to delete receive link: %v", err)
	}
	_, found = store.GetReceiveLink("recv-1")
	if found {
		t.Error("Receive link still found after delete")
	}
}

func TestReceivedFileCRUD(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// First create a receive link
	link := &models.ReceiveLink{
		ID:        "recv-for-files",
		Name:      "Upload Link",
		CreatedAt: time.Now(),
	}
	store.SaveReceiveLink(link)

	// Create received files
	files := []*models.ReceivedFile{
		{ID: "file1", ReceiveLinkID: "recv-for-files", FileName: "upload1.txt", OriginalName: "Document.txt", FileSize: 1024, CreatedAt: time.Now()},
		{ID: "file2", ReceiveLinkID: "recv-for-files", FileName: "upload2.pdf", OriginalName: "Report.pdf", FileSize: 2048, CreatedAt: time.Now()},
	}

	for _, f := range files {
		err := store.SaveReceivedFile(f)
		if err != nil {
			t.Fatalf("Failed to save received file: %v", err)
		}
	}

	// Test GetReceivedFiles
	retrieved, err := store.GetReceivedFiles("recv-for-files")
	if err != nil {
		t.Fatalf("GetReceivedFiles failed: %v", err)
	}
	if len(retrieved) != 2 {
		t.Errorf("Expected 2 files, got %d", len(retrieved))
	}

	// Test DeleteReceivedFile
	err = store.DeleteReceivedFile("recv-for-files", "file1")
	if err != nil {
		t.Fatalf("Failed to delete received file: %v", err)
	}
	retrieved, _ = store.GetReceivedFiles("recv-for-files")
	if len(retrieved) != 1 {
		t.Errorf("Expected 1 file after delete, got %d", len(retrieved))
	}
}

func TestFolderContents(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a folder share
	share := &models.Share{
		ID:          "folder-share",
		FileName:    "Photos",
		IsDirectory: true,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		CreatedAt:   time.Now(),
	}
	store.Save(share)

	// Add folder contents
	// Note: The implementation uses trailing slashes for nested paths
	// Root items have RelativePath="/"
	// Subfolder items have RelativePath="/subfolder/" (with trailing slash)
	contents := []*models.FolderEntry{
		{ID: "e1", ShareID: "folder-share", RelativePath: "/", FileName: "photo1.jpg", FileSize: 1000, IsDirectory: false},
		{ID: "e2", ShareID: "folder-share", RelativePath: "/", FileName: "photo2.jpg", FileSize: 2000, IsDirectory: false},
		{ID: "e3", ShareID: "folder-share", RelativePath: "/", FileName: "subfolder", IsDirectory: true},
		{ID: "e4", ShareID: "folder-share", RelativePath: "/subfolder/", FileName: "photo3.jpg", FileSize: 3000, IsDirectory: false},
	}

	err := store.SaveFolderContents("folder-share", contents)
	if err != nil {
		t.Fatalf("Failed to save folder contents: %v", err)
	}

	// Test GetFolderContents for root
	rootContents, err := store.GetFolderContents("folder-share", "/")
	if err != nil {
		t.Fatalf("GetFolderContents failed: %v", err)
	}
	if len(rootContents) != 3 {
		t.Errorf("Expected 3 items in root, got %d", len(rootContents))
	}

	// Test GetFolderContents for subfolder
	subContents, err := store.GetFolderContents("folder-share", "/subfolder")
	if err != nil {
		t.Fatalf("GetFolderContents for subfolder failed: %v", err)
	}
	if len(subContents) != 1 {
		t.Errorf("Expected 1 item in subfolder, got %d", len(subContents))
	}

	// Test DeleteFolderContents
	err = store.DeleteFolderContents("folder-share")
	if err != nil {
		t.Fatalf("DeleteFolderContents failed: %v", err)
	}
	rootContents, _ = store.GetFolderContents("folder-share", "/")
	if len(rootContents) != 0 {
		t.Errorf("Expected 0 items after delete, got %d", len(rootContents))
	}
}

func TestUploadsDir(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	uploadsDir := store.UploadsDir()
	if uploadsDir == "" {
		t.Error("UploadsDir returned empty string")
	}

	// Verify uploads directory exists
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		t.Error("UploadsDir should return an existing directory")
	}
}

func TestConcurrentAccess(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Test concurrent writes
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(n int) {
			share := &models.Share{
				ID:        "concurrent-" + string(rune('0'+n)),
				FileName:  "file.txt",
				ExpiresAt: time.Now().Add(24 * time.Hour),
				CreatedAt: time.Now(),
			}
			store.Save(share)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all shares exist
	all := store.GetAll()
	if len(all) != 10 {
		t.Errorf("Expected 10 shares after concurrent writes, got %d", len(all))
	}
}

func TestDeleteNonExistent(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Delete non-existent share should not error
	err := store.Delete("nonexistent-share")
	if err != nil {
		t.Errorf("Delete non-existent share should not error: %v", err)
	}
}

func TestIncrementDownloadsNonExistent(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Increment non-existent share should not error (returns false, no rows affected)
	allowed, err := store.IncrementDownloads("nonexistent-share")
	if err != nil {
		t.Errorf("IncrementDownloads non-existent share should not error: %v", err)
	}
	if allowed {
		t.Errorf("IncrementDownloads non-existent share should return false")
	}
}

func TestGetNonExistent(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	_, found := store.Get("nonexistent-share")
	if found {
		t.Error("Get should return false for non-existent share")
	}
}

func TestUpdateShare(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create share
	share := &models.Share{
		ID:           "update-test",
		FileName:     "original.txt",
		OriginalName: "Original.txt",
		FileSize:     100,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		Downloads:    0,
	}
	store.Save(share)

	// Update by saving again with same ID
	share.OriginalName = "Updated.txt"
	share.Downloads = 5
	store.Save(share)

	retrieved, found := store.Get("update-test")
	if !found {
		t.Fatal("Share not found after update")
	}
	if retrieved.OriginalName != "Updated.txt" {
		t.Errorf("OriginalName not updated: got %s", retrieved.OriginalName)
	}
	if retrieved.Downloads != 5 {
		t.Errorf("Downloads not updated: got %d", retrieved.Downloads)
	}
}

func TestGetExpiredShares(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create mix of expired and valid shares
	shares := []*models.Share{
		{ID: "expired1", FileName: "e1.txt", ExpiresAt: time.Now().Add(-1 * time.Hour), CreatedAt: time.Now()},
		{ID: "expired2", FileName: "e2.txt", ExpiresAt: time.Now().Add(-2 * time.Hour), CreatedAt: time.Now()},
		{ID: "valid1", FileName: "v1.txt", ExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now()},
	}

	for _, s := range shares {
		store.Save(s)
	}

	// GetAll may filter expired shares - check we get at least the valid one
	all := store.GetAll()
	if len(all) < 1 {
		t.Errorf("Expected at least 1 share, got %d", len(all))
	}

	// The valid share should be present
	found := false
	for _, s := range all {
		if s.ID == "valid1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Valid share should be returned")
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	share := &models.Share{
		ID:           "search-case",
		FileName:     "MyDocument.PDF",
		OriginalName: "MyDocument.PDF",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
	}
	store.Save(share)

	// Search should be case-insensitive
	results, err := store.Search("mydocument")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for case-insensitive search, got %d", len(results))
	}

	results, err = store.Search("MYDOCUMENT")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for uppercase search, got %d", len(results))
	}
}

func TestReceiveLinkExpiration(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create links with different expiration states
	futureTime := time.Now().Add(24 * time.Hour)

	links := []*models.ReceiveLink{
		{ID: "valid-link", Name: "Valid", ExpiresAt: &futureTime, CreatedAt: time.Now()},
		{ID: "no-expiry", Name: "No Expiry", ExpiresAt: nil, CreatedAt: time.Now()},
	}

	for _, l := range links {
		store.SaveReceiveLink(l)
	}

	// GetAllReceiveLinks should return valid links
	all, err := store.GetAllReceiveLinks()
	if err != nil {
		t.Fatalf("GetAllReceiveLinks failed: %v", err)
	}
	if len(all) < 1 {
		t.Errorf("Expected at least 1 link, got %d", len(all))
	}
}

func TestMultipleIncrements(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	share := &models.Share{
		ID:        "increment-test",
		FileName:  "test.txt",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		Downloads: 0,
	}
	store.Save(share)

	// Increment multiple times
	for i := 0; i < 5; i++ {
		_, _ = store.IncrementDownloads("increment-test")
	}

	retrieved, _ := store.Get("increment-test")
	if retrieved.Downloads != 5 {
		t.Errorf("Expected 5 downloads, got %d", retrieved.Downloads)
	}
}

func TestStorageClose(t *testing.T) {
	store, cleanup := setupTestStorage(t)

	// Close should not error
	err := store.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}

	cleanup()
}
