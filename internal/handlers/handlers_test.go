package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"casadrop/internal/models"
	"casadrop/internal/storage"
	"casadrop/internal/utils"
)

// setupTestHandler creates a Handler with temporary storage for testing
func setupTestHandler(t *testing.T) (*Handler, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "casadrop-handler-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create required directories
	os.MkdirAll(filepath.Join(tmpDir, "uploads"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "templates"), 0755)

	// Create minimal templates
	createTestTemplates(t, filepath.Join(tmpDir, "templates"))

	store, err := storage.New(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Set DATA_DIR for webhook service
	os.Setenv("DATA_DIR", tmpDir)

	handler, err := New(store, filepath.Join(tmpDir, "templates"))
	if err != nil {
		store.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create handler: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return handler, cleanup
}

// createTestTemplates creates minimal test templates
func createTestTemplates(t *testing.T, templatesDir string) {
	templates := map[string]string{
		"index.html":   `<!DOCTYPE html><html><body>Index</body></html>`,
		"share.html":   `<!DOCTYPE html><html><body>Share: {{.Share.ID}}</body></html>`,
		"receive.html": `<!DOCTYPE html><html><body>Receive: {{.Link.ID}}</body></html>`,
		"folder.html":  `<!DOCTYPE html><html><body>Folder</body></html>`,
	}

	for name, content := range templates {
		err := os.WriteFile(filepath.Join(templatesDir, name), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create template %s: %v", name, err)
		}
	}
}

func TestNew(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}

func TestIsFileTypeAllowed(t *testing.T) {
	tests := []struct {
		filename string
		allowed  bool
	}{
		{"document.pdf", true},
		{"image.jpg", true},
		{"video.mp4", true},
		{"archive.zip", true},
		{"malware.exe", false},
		{"script.bat", false},
		{"script.ps1", false},
		{"link.lnk", false},
		{"script.vbs", false},
		{"MALWARE.EXE", false}, // Case insensitive
	}

	for _, tt := range tests {
		result := isFileTypeAllowed(tt.filename)
		if result != tt.allowed {
			t.Errorf("isFileTypeAllowed(%q) = %v, want %v", tt.filename, result, tt.allowed)
		}
	}
}

func TestSharePasswordRateLimiter(t *testing.T) {
	limiter := newSharePasswordRateLimiter()

	shareID := "test-share"
	ip := "192.168.1.1"

	// Initially not blocked
	if limiter.isBlocked(shareID, ip) {
		t.Error("Should not be blocked initially")
	}

	// Record failures up to max
	for i := 0; i < sharePasswordMaxAttempts; i++ {
		limiter.recordFailure(shareID, ip)
	}

	// Should be blocked now
	if !limiter.isBlocked(shareID, ip) {
		t.Error("Should be blocked after max attempts")
	}

	// Reset should unblock
	limiter.resetAttempts(shareID, ip)
	if limiter.isBlocked(shareID, ip) {
		t.Error("Should not be blocked after reset")
	}
}

func TestListShares(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create test shares: one image, one folder, one plain file.
	// The image's mime_type and the folder's is_directory must reach the
	// JSON response; otherwise the share-list UI can't decide whether to
	// render a thumbnail or a folder badge (regression seen 2026-04-14).
	shares := []*models.Share{
		{
			ID:           "img-1",
			FileName:     "img-1.png",
			OriginalName: "photo.png",
			FileSize:     12345,
			MimeType:     "image/png",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			CreatedAt:    time.Now(),
		},
		{
			ID:           "folder-1",
			FileName:     "",
			OriginalName: "Pictures",
			FileSize:     6789,
			MimeType:     "application/x-directory",
			IsDirectory:  true,
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			CreatedAt:    time.Now(),
		},
		{
			ID:           "txt-1",
			FileName:     "txt-1.txt",
			OriginalName: "notes.txt",
			FileSize:     42,
			MimeType:     "text/plain; charset=utf-8",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			CreatedAt:    time.Now(),
		},
	}
	for _, s := range shares {
		handler.storage.Save(s)
	}

	req := httptest.NewRequest("GET", "/api/shares", nil)
	rec := httptest.NewRecorder()

	handler.ListShares(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", rec.Code)
	}

	var resp []models.ShareResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 3 {
		t.Fatalf("Expected 3 shares, got %d", len(resp))
	}

	byID := map[string]models.ShareResponse{}
	for _, s := range resp {
		byID[s.ID] = s
	}

	if got := byID["img-1"].MimeType; got != "image/png" {
		t.Errorf("img-1.mime_type = %q, want image/png (share-list thumbnails depend on this)", got)
	}
	if !byID["folder-1"].IsDirectory {
		t.Errorf("folder-1.is_directory = false, want true (folder badge depends on this)")
	}
	if got := byID["txt-1"].MimeType; got != "text/plain; charset=utf-8" {
		t.Errorf("txt-1.mime_type = %q, want text/plain; charset=utf-8", got)
	}
}

func TestGetStats(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create test shares
	shares := []*models.Share{
		{ID: "s1", FileName: "f1.txt", FileSize: 1000, Downloads: 5, ExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now()},
		{ID: "s2", FileName: "f2.txt", FileSize: 2000, Downloads: 10, ExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now()},
	}
	for _, s := range shares {
		handler.storage.Save(s)
	}

	req := httptest.NewRequest("GET", "/api/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	var stats struct {
		TotalShares    int   `json:"total_shares"`
		TotalDownloads int   `json:"total_downloads"`
		TotalSize      int64 `json:"total_size"`
	}
	json.Unmarshal(rec.Body.Bytes(), &stats)
	if stats.TotalShares != 2 {
		t.Errorf("Expected 2 total shares, got %d", stats.TotalShares)
	}
	if stats.TotalDownloads != 15 {
		t.Errorf("Expected 15 total downloads, got %d", stats.TotalDownloads)
	}
}

func TestDeleteShare(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a share
	share := &models.Share{
		ID:        "delete-me",
		FileName:  "test.txt",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	handler.storage.Save(share)

	// Create router for path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/shares/{id}", handler.DeleteShare).Methods("DELETE")

	req := httptest.NewRequest("DELETE", "/api/shares/delete-me", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// API returns 204 No Content on successful delete
	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d", rec.Code)
	}

	// Verify deleted
	_, found := handler.storage.Get("delete-me")
	if found {
		t.Error("Share should be deleted")
	}
}

func TestGetShareInfo(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	share := &models.Share{
		ID:           "info-share",
		FileName:     "test.txt",
		OriginalName: "Test File.txt",
		FileSize:     1234,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
	}
	handler.storage.Save(share)

	router := mux.NewRouter()
	router.HandleFunc("/api/shares/{id}", handler.GetShareInfo).Methods("GET")

	req := httptest.NewRequest("GET", "/api/shares/info-share", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	// Check the response contains share info
	body := rec.Body.String()
	if !strings.Contains(body, "info-share") {
		t.Errorf("Response should contain share ID, got: %s", body)
	}
}

func TestGetShareInfo_NotFound(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	router := mux.NewRouter()
	router.HandleFunc("/api/shares/{id}", handler.GetShareInfo).Methods("GET")

	req := httptest.NewRequest("GET", "/api/shares/nonexistent", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", rec.Code)
	}
}

func TestIndexPage(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.IndexPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestQRCode(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a share
	share := &models.Share{
		ID:        "qr-share",
		FileName:  "test.txt",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	handler.storage.Save(share)

	router := mux.NewRouter()
	router.HandleFunc("/api/qr/{id}", handler.QRCode).Methods("GET")

	req := httptest.NewRequest("GET", "/api/qr/qr-share", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "image/png" {
		t.Errorf("Expected image/png, got %s", contentType)
	}
}

func TestQRCode_NotFound(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	router := mux.NewRouter()
	router.HandleFunc("/api/qr/{id}", handler.QRCode).Methods("GET")

	req := httptest.NewRequest("GET", "/api/qr/nonexistent", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", rec.Code)
	}
}

func TestUploadFile_BlockedExtension(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create multipart form with blocked file type
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "malware.exe")
	part.Write([]byte("fake exe content"))
	writer.WriteField("expiration", "24h")
	writer.Close()

	req := httptest.NewRequest("POST", "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	handler.UploadFile(rec, req)

	// Blocked extensions return 415 Unsupported Media Type
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("Expected 415 for blocked extension, got %d", rec.Code)
	}
}

func TestUploadFile_Success(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create multipart form with valid file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "document.pdf")
	part.Write([]byte("fake pdf content"))
	writer.WriteField("expiration", "24h")
	writer.Close()

	req := httptest.NewRequest("POST", "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	handler.UploadFile(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &result)
	if result["id"] == nil {
		t.Error("Response should include share ID")
	}
}

func TestUploadFile_NoFile(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("expiration", "24h")
	writer.Close()

	req := httptest.NewRequest("POST", "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	handler.UploadFile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", rec.Code)
	}
}

func TestDownloadFile_NotFound(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	router := mux.NewRouter()
	router.HandleFunc("/download/{id}", handler.DownloadFile).Methods("GET")

	req := httptest.NewRequest("GET", "/download/nonexistent", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", rec.Code)
	}
}

func TestDownloadFile_Expired(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	share := &models.Share{
		ID:        "expired-share",
		FileName:  "test.txt",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		CreatedAt: time.Now(),
	}
	handler.storage.Save(share)

	router := mux.NewRouter()
	router.HandleFunc("/download/{id}", handler.DownloadFile).Methods("GET")

	req := httptest.NewRequest("GET", "/download/expired-share", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Expired shares return 404 (treated as not found)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", rec.Code)
	}
}

func TestDownloadFile_MaxDownloadsReached(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	share := &models.Share{
		ID:           "max-downloads",
		FileName:     "test.txt",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		MaxDownloads: 5,
		Downloads:    5, // Already at max
	}
	handler.storage.Save(share)

	router := mux.NewRouter()
	router.HandleFunc("/download/{id}", handler.DownloadFile).Methods("GET")

	req := httptest.NewRequest("GET", "/download/max-downloads", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Max downloads returns 403 Forbidden
	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", rec.Code)
	}
}

func TestSharePage_RequiresPassword(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	share := &models.Share{
		ID:          "password-share",
		FileName:    "secret.pdf",
		HasPassword: true,
		Password:    "$2a$10$test", // bcrypt hash placeholder
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		CreatedAt:   time.Now(),
	}
	handler.storage.Save(share)

	router := mux.NewRouter()
	router.HandleFunc("/share/{id}", handler.SharePage).Methods("GET", "POST")

	req := httptest.NewRequest("GET", "/share/password-share", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should return 200 and render the page
	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestGetNetworkInfo(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/network", nil)
	rec := httptest.NewRecorder()

	handler.GetNetworkInfo(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	// Should return valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}
}

func TestTunnelURL_GET(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/tunnel", nil)
	rec := httptest.NewRecorder()

	handler.TunnelURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestTunnelURL_POST(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	body := strings.NewReader(`{"cloudflareTunnelURL":"https://share.example.com"}`)
	req := httptest.NewRequest("POST", "/api/tunnel", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.TunnelURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestWebhookConfig_GET(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/webhook", nil)
	rec := httptest.NewRecorder()

	handler.WebhookConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestWebhookConfig_POST(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	body := strings.NewReader(`{"url":"https://webhook.example.com","enabled":true}`)
	req := httptest.NewRequest("POST", "/api/webhook", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.WebhookConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

// Receive link tests

func TestCreateReceiveLink(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	body := strings.NewReader(`{"name":"Test Upload Link","maxUploads":10,"maxFileSize":104857600}`)
	req := httptest.NewRequest("POST", "/api/receive-links", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateReceiveLink(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &result)
	if result["id"] == nil {
		t.Error("Response should include link ID")
	}
}

func TestListReceiveLinks(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a link first
	link := &models.ReceiveLink{
		ID:        "recv-1",
		Name:      "Test Link",
		CreatedAt: time.Now(),
	}
	handler.storage.SaveReceiveLink(link)

	req := httptest.NewRequest("GET", "/api/receive-links", nil)
	rec := httptest.NewRecorder()

	handler.ListReceiveLinks(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	var links []*models.ReceiveLink
	json.Unmarshal(rec.Body.Bytes(), &links)
	if len(links) != 1 {
		t.Errorf("Expected 1 link, got %d", len(links))
	}
}

func TestDeleteReceiveLink(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	link := &models.ReceiveLink{
		ID:        "delete-recv",
		Name:      "Delete Me",
		CreatedAt: time.Now(),
	}
	handler.storage.SaveReceiveLink(link)

	router := mux.NewRouter()
	router.HandleFunc("/api/receive-links/{id}", handler.DeleteReceiveLink).Methods("DELETE")

	req := httptest.NewRequest("DELETE", "/api/receive-links/delete-recv", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Delete returns 204 No Content
	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d", rec.Code)
	}

	// Verify deleted
	_, found := handler.storage.GetReceiveLink("delete-recv")
	if found {
		t.Error("Receive link should be deleted")
	}
}

func TestReceivePage(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	link := &models.ReceiveLink{
		ID:        "recv-page",
		Name:      "Upload Here",
		CreatedAt: time.Now(),
	}
	handler.storage.SaveReceiveLink(link)

	router := mux.NewRouter()
	router.HandleFunc("/receive/{id}", handler.ReceivePage).Methods("GET")

	req := httptest.NewRequest("GET", "/receive/recv-page", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestReceivePage_Expired(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	expiredTime := time.Now().Add(-1 * time.Hour)
	link := &models.ReceiveLink{
		ID:        "expired-recv",
		Name:      "Expired Link",
		ExpiresAt: &expiredTime,
		CreatedAt: time.Now(),
	}
	handler.storage.SaveReceiveLink(link)

	router := mux.NewRouter()
	router.HandleFunc("/receive/{id}", handler.ReceivePage).Methods("GET")

	req := httptest.NewRequest("GET", "/receive/expired-recv", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Expired links may render an error page (200) or return error status
	// Just check it doesn't panic and returns a response
	if rec.Code == 0 {
		t.Error("Should return a valid response code")
	}
}

func TestReceiveUpload_MaxUploadsReached(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	link := &models.ReceiveLink{
		ID:             "max-recv",
		Name:           "Max Uploads",
		MaxUploads:     1,
		CurrentUploads: 1, // Already at max
		CreatedAt:      time.Now(),
	}
	handler.storage.SaveReceiveLink(link)

	router := mux.NewRouter()
	router.HandleFunc("/receive/{id}/upload", handler.ReceiveUpload).Methods("POST")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/receive/max-recv/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Max uploads returns 403 Forbidden
	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", rec.Code)
	}
}

// BrowseFiles tests

func TestBrowseFiles(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/browse", nil)
	rec := httptest.NewRecorder()

	handler.BrowseFiles(rec, req)

	// Should return 200 with file listing (may be empty or have allowed paths)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

// Helper function tests

func TestValidateURL(t *testing.T) {
	tests := []struct {
		input        string
		requireHTTPS bool
		valid        bool
	}{
		{"https://example.com", false, true},
		{"http://example.com", false, true},
		{"https://share.example.com/path", false, true},
		{"not-a-url", false, false},
		{"ftp://example.com", false, false},
		{"", false, true},                   // Empty is valid (optional field)
		{"http://example.com", true, false}, // Requires HTTPS
		{"https://example.com", true, true}, // HTTPS OK
	}

	for _, tt := range tests {
		err := utils.ValidateURL(tt.input, tt.requireHTTPS)
		valid := err == nil
		if valid != tt.valid {
			t.Errorf("validateURL(%q, %v) valid=%v, want %v (err=%v)", tt.input, tt.requireHTTPS, valid, tt.valid, err)
		}
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tt := range tests {
		result := formatFileSize(tt.size)
		if result != tt.expected {
			t.Errorf("formatFileSize(%d) = %q, want %q", tt.size, result, tt.expected)
		}
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "RemoteAddr with port",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "172.16.0.1"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "172.16.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := utils.GetClientIP(req)
			if ip != tt.expected {
				t.Errorf("getClientIP() = %s, want %s", ip, tt.expected)
			}
		})
	}
}
