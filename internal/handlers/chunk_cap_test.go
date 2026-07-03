package handlers

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// Regression test for the chunked-upload disk-overrun fix. InitChunkUpload only
// validates the client-DECLARED TotalSize; UploadChunk must enforce a running
// cumulative cap against it, otherwise a client can declare TotalSize:1 and
// stream 10 MB per index to disk until Finalize finally rechecks — an
// authenticated disk-exhaustion / size+quota bypass.
func TestUploadChunkEnforcesCumulativeSize(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	tempDir := t.TempDir()
	upload := &ChunkUpload{
		ID:             "cap-test",
		FileName:       "big.bin",
		TotalSize:      100, // declared tiny
		TotalChunks:    1000,
		ChunksReceived: make(map[int]bool),
		ChunkSizes:     make(map[int]int64),
		TempDir:        tempDir,
		CreatedAt:      time.Now(),
	}
	chunkUploadsMu.Lock()
	chunkUploads["cap-test"] = upload
	chunkUploadsMu.Unlock()
	defer func() {
		chunkUploadsMu.Lock()
		delete(chunkUploads, "cap-test")
		chunkUploadsMu.Unlock()
	}()

	router := mux.NewRouter()
	router.HandleFunc("/api/upload/chunk/{uploadId}", handler.UploadChunk).Methods("POST")

	// A 500-byte chunk against a declared 100-byte total must be rejected 413.
	body := strings.NewReader(strings.Repeat("A", 500))
	req := httptest.NewRequest("POST", "/api/upload/chunk/cap-test?index=0", body)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != 413 {
		t.Fatalf("oversized chunk: got %d, want 413", rec.Code)
	}
	// The rejected chunk must not be left on disk.
	if _, err := os.Stat(filepath.Join(tempDir, "chunk_0")); !os.IsNotExist(err) {
		t.Errorf("rejected chunk file still on disk (err=%v)", err)
	}
	// And nothing should have been counted toward the upload.
	chunkUploadsMu.Lock()
	got := chunkUploads["cap-test"].ReceivedBytes
	chunkUploadsMu.Unlock()
	if got != 0 {
		t.Errorf("ReceivedBytes = %d, want 0 after rejection", got)
	}
}
