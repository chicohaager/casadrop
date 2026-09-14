package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"casadrop/internal/middleware"
	"casadrop/internal/models"
)

// resetChunkRegistry clears the process-global upload map so tests don't leak
// into each other.
func resetChunkRegistry(t *testing.T) {
	t.Helper()
	chunkUploadsMu.Lock()
	chunkUploads = make(map[string]*ChunkUpload)
	chunkUploadsMu.Unlock()
}

func initChunkUpload(t *testing.T, h *Handler, user *middleware.SessionUser, fileName string, totalSize int64, totalChunks int) string {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{"fileName": fileName, "totalSize": totalSize, "totalChunks": totalChunks})
	req := httptest.NewRequest("POST", "/api/upload/chunk/init", strings.NewReader(string(body)))
	if user != nil {
		req = req.WithContext(middleware.ContextWithUser(req.Context(), user))
	}
	rr := httptest.NewRecorder()
	h.InitChunkUpload(rr, req)
	if rr.Code != 200 {
		t.Fatalf("init: %d %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp["uploadId"]
}

func postChunk(t *testing.T, h *Handler, user *middleware.SessionUser, uploadID string, index int, data string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/upload/chunk/"+uploadID+"?index="+strconv.Itoa(index), strings.NewReader(data))
	if user != nil {
		req = req.WithContext(middleware.ContextWithUser(req.Context(), user))
	}
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/upload/chunk/{uploadId}", h.UploadChunk).Methods("POST")
	router.ServeHTTP(rr, req)
	return rr
}

func TestManifestWrittenAtInit(t *testing.T) {
	resetChunkRegistry(t)
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	id := initChunkUpload(t, h, &middleware.SessionUser{ID: "u1", Role: models.RoleUser}, "big.bin", 1000, 4)
	manifest := filepath.Join(h.storage.UploadsDir(), "chunks", id, manifestName)
	if _, err := os.Stat(manifest); err != nil {
		t.Fatalf("manifest not written: %v", err)
	}
}

func TestStatusReportsReceivedChunks(t *testing.T) {
	resetChunkRegistry(t)
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	user := &middleware.SessionUser{ID: "u1", Role: models.RoleUser}

	id := initChunkUpload(t, h, user, "big.bin", 30, 3)
	if rr := postChunk(t, h, user, id, 0, "aaaaaaaaaa"); rr.Code != 200 {
		t.Fatalf("chunk 0: %d %s", rr.Code, rr.Body.String())
	}
	if rr := postChunk(t, h, user, id, 2, "cccccccccc"); rr.Code != 200 {
		t.Fatalf("chunk 2: %d %s", rr.Code, rr.Body.String())
	}

	req := httptest.NewRequest("GET", "/api/upload/chunk/"+id, nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), user))
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/upload/chunk/{uploadId}", h.ChunkUploadStatus).Methods("GET")
	router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status: %d %s", rr.Code, rr.Body.String())
	}

	var status struct {
		Received []int `json:"received"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if len(status.Received) != 2 || status.Received[0] != 0 || status.Received[1] != 2 {
		t.Errorf("received = %v, want [0 2] sorted", status.Received)
	}
}

func TestChunkUploadIsOwnerBound(t *testing.T) {
	resetChunkRegistry(t)
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	owner := &middleware.SessionUser{ID: "owner", Role: models.RoleUser}
	intruder := &middleware.SessionUser{ID: "intruder", Role: models.RoleUser}

	id := initChunkUpload(t, h, owner, "secret.bin", 100, 2)

	// A foreign user must not see status, add chunks, or finalise — all 404 so
	// they cannot even tell the upload exists.
	statusRouter := mux.NewRouter()
	statusRouter.HandleFunc("/api/upload/chunk/{uploadId}", h.ChunkUploadStatus).Methods("GET")
	req := httptest.NewRequest("GET", "/api/upload/chunk/"+id, nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), intruder))
	rr := httptest.NewRecorder()
	statusRouter.ServeHTTP(rr, req)
	if rr.Code != 404 {
		t.Errorf("intruder status: %d, want 404", rr.Code)
	}

	if rr := postChunk(t, h, intruder, id, 0, "xxxxx"); rr.Code != 404 {
		t.Errorf("intruder chunk: %d, want 404", rr.Code)
	}

	// The owner still works — the guard is not just blocking everyone.
	if rr := postChunk(t, h, owner, id, 0, "yyyyy"); rr.Code != 200 {
		t.Errorf("owner chunk: %d, want 200", rr.Code)
	}
}

func TestRestoreChunkUploadsFromDisk(t *testing.T) {
	resetChunkRegistry(t)
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	user := &middleware.SessionUser{ID: "u1", Role: models.RoleUser}

	id := initChunkUpload(t, h, user, "big.bin", 30, 3)
	if rr := postChunk(t, h, user, id, 1, "bbbbbbbbbb"); rr.Code != 200 {
		t.Fatalf("chunk 1: %d %s", rr.Code, rr.Body.String())
	}

	// Simulate a restart: drop the in-memory registry, keep the disk.
	resetChunkRegistry(t)
	if _, ok := chunkUploads[id]; ok {
		t.Fatal("registry not cleared")
	}

	RestoreChunkUploads(h.storage.UploadsDir())

	chunkUploadsMu.Lock()
	restored, ok := chunkUploads[id]
	chunkUploadsMu.Unlock()
	if !ok {
		t.Fatal("upload was not restored after restart")
	}
	if restored.UserID != "u1" || restored.FileName != "big.bin" || restored.TotalChunks != 3 {
		t.Errorf("restored facts wrong: %+v", restored)
	}
	// Progress came back from the chunk file on disk, not from memory.
	if !restored.ChunksReceived[1] || restored.ChunksReceived[0] {
		t.Errorf("restored progress wrong: received=%v", restored.ChunksReceived)
	}
	if restored.ReceivedBytes != 10 {
		t.Errorf("restored ReceivedBytes = %d, want 10", restored.ReceivedBytes)
	}
}

func TestCleanupRemovesManifestAndChunks(t *testing.T) {
	resetChunkRegistry(t)
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	user := &middleware.SessionUser{ID: "u1", Role: models.RoleUser}

	id := initChunkUpload(t, h, user, "old.bin", 30, 3)
	tempDir := filepath.Join(h.storage.UploadsDir(), "chunks", id)

	// Age it past the TTL and sweep.
	chunkUploadsMu.Lock()
	chunkUploads[id].CreatedAt = time.Now().Add(-48 * time.Hour)
	chunkUploadsMu.Unlock()
	cleanupExpiredChunkUploads()

	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("expired upload dir still on disk (err=%v)", err)
	}
	chunkUploadsMu.Lock()
	_, ok := chunkUploads[id]
	chunkUploadsMu.Unlock()
	if ok {
		t.Error("expired upload still in registry")
	}
}
