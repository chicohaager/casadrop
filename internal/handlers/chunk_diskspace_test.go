package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"casadrop/internal/middleware"
	"casadrop/internal/models"
)

// stubFreeDisk makes the free-space preflight see exactly free bytes.
func stubFreeDisk(t *testing.T, free int64) {
	t.Helper()
	orig := freeDiskBytes
	freeDiskBytes = func(string) (int64, bool) { return free, true }
	t.Cleanup(func() { freeDiskBytes = orig })
}

func initRaw(h *Handler, user *middleware.SessionUser, name string, size int64, chunks int) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]interface{}{"fileName": name, "totalSize": size, "totalChunks": chunks})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/chunk/init", strings.NewReader(string(body)))
	if user != nil {
		req = req.WithContext(middleware.ContextWithUser(req.Context(), user))
	}
	rr := httptest.NewRecorder()
	h.InitChunkUpload(rr, req)
	return rr
}

// The 2026-11-24 incident: a 47 GB upload streamed for 8 minutes into a disk
// with ~44 GB free and died at chunk 5670 of 6011. Init must refuse up front.
func TestInitRefusesWhenDiskTooSmall(t *testing.T) {
	resetChunkRegistry(t)
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	const size = int64(8) << 30   // under the default 10 GB cap
	stubFreeDisk(t, size-(3<<30)) // 3 GB short, like the incident

	rr := initRaw(h, nil, "movie.mkv", size, 1024)
	if rr.Code != http.StatusInsufficientStorage {
		t.Fatalf("init on too-small disk: %d %q, want 507", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Not enough free disk space") {
		t.Errorf("message does not say why: %q", rr.Body.String())
	}
	entries, _ := os.ReadDir(filepath.Join(h.storage.UploadsDir(), "chunks"))
	if len(entries) != 0 {
		t.Errorf("refused init left %d temp dir(s) behind", len(entries))
	}
	chunkUploadsMu.Lock()
	n := len(chunkUploads)
	chunkUploadsMu.Unlock()
	if n != 0 {
		t.Errorf("refused init registered %d upload(s)", n)
	}

	// Positive control: with size + reserve free, the same init goes through.
	stubFreeDisk(t, size+chunkDiskReserve)
	if rr := initRaw(h, nil, "movie.mkv", size, 1024); rr.Code != 200 {
		t.Fatalf("init with enough space: %d %q, want 200", rr.Code, rr.Body.String())
	}
}

// Space already promised to a running upload is not free for a second one.
func TestInitCountsPendingUploads(t *testing.T) {
	resetChunkRegistry(t)
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	const size = int64(10) << 30
	stubFreeDisk(t, size+chunkDiskReserve+(size/2)) // room for 1.5 uploads

	if rr := initRaw(h, nil, "a.mkv", size, 1280); rr.Code != 200 {
		t.Fatalf("first init: %d %q", rr.Code, rr.Body.String())
	}
	if rr := initRaw(h, nil, "b.mkv", size, 1280); rr.Code != http.StatusInsufficientStorage {
		t.Fatalf("second init while first still pending: %d, want 507", rr.Code)
	}
}

// Assembly must not hold the file twice: each chunk is gone once appended.
func TestFinalizeDeletesChunksWhileAssembling(t *testing.T) {
	resetChunkRegistry(t)
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	user := &middleware.SessionUser{ID: "u1", Role: models.RoleUser}

	id := initChunkUpload(t, h, user, "film.mkv", 40, 4)
	for i := 0; i < 4; i++ {
		if rr := postChunk(t, h, user, id, i, strings.Repeat(fmt.Sprint(i), 10)); rr.Code != 200 {
			t.Fatalf("chunk %d: %d %s", i, rr.Code, rr.Body.String())
		}
	}

	var stillThere []string
	orig := chunkAssembled
	chunkAssembled = func(tempDir string, i int) {
		for j := 0; j <= i; j++ {
			if _, err := os.Stat(filepath.Join(tempDir, fmt.Sprintf("chunk_%d", j))); err == nil {
				stillThere = append(stillThere, fmt.Sprintf("after %d: chunk_%d", i, j))
			}
		}
	}
	defer func() { chunkAssembled = orig }()

	req := httptest.NewRequest(http.MethodPost, "/api/upload/chunk/"+id+"/finalize", strings.NewReader(`{"expires_in":1}`))
	req = req.WithContext(middleware.ContextWithUser(req.Context(), user))
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/upload/chunk/{uploadId}/finalize", h.FinalizeChunkUpload).Methods("POST")
	router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("finalize: %d %s", rr.Code, rr.Body.String())
	}
	if len(stillThere) > 0 {
		t.Errorf("appended chunks still on disk during assembly (file held twice): %v", stillThere)
	}

	// The assembled file is complete and in order.
	var resp struct {
		ID string `json:"id"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	data, err := os.ReadFile(filepath.Join(h.storage.UploadsDir(), resp.ID+".mkv"))
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Repeat("0", 10) + strings.Repeat("1", 10) + strings.Repeat("2", 10) + strings.Repeat("3", 10); string(data) != want {
		t.Errorf("assembled = %q, want %q", data, want)
	}
}

// Directories without a manifest (pre-persistence versions, crashed init)
// were skipped by restore and never reached by the in-memory cleanup — two
// from June stayed on a full production disk for six months.
func TestOrphanChunkDirsAreSwept(t *testing.T) {
	resetChunkRegistry(t)
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	root := filepath.Join(h.storage.UploadsDir(), "chunks")
	old := time.Now().Add(-chunkUploadTTL - time.Hour)

	mkOrphan := func(name string, mtime time.Time) string {
		dir := filepath.Join(root, name)
		os.MkdirAll(dir, 0755)
		f := filepath.Join(dir, "chunk_0")
		os.WriteFile(f, []byte("x"), 0644)
		os.Chtimes(f, mtime, mtime)
		os.Chtimes(dir, mtime, mtime)
		return dir
	}
	oldOrphan := mkOrphan("61ecffa6-old", old)
	freshOrphan := mkOrphan("fresh-orphan", time.Now()) // may be an init in progress

	// A restored upload whose last chunk arrived recently must survive even
	// though it was created long ago.
	user := &middleware.SessionUser{ID: "u1", Role: models.RoleUser}
	liveID := initChunkUpload(t, h, user, "slow.mkv", 30, 3)
	liveDir := filepath.Join(root, liveID)
	resetChunkRegistry(t) // simulate restart

	RestoreChunkUploads(h.storage.UploadsDir())

	if _, err := os.Stat(oldOrphan); !os.IsNotExist(err) {
		t.Errorf("orphan older than TTL still on disk (err=%v)", err)
	}
	if _, err := os.Stat(freshOrphan); err != nil {
		t.Errorf("fresh orphan removed: %v", err)
	}
	if _, err := os.Stat(liveDir); err != nil {
		t.Errorf("registered upload dir removed: %v", err)
	}
}

// The TTL is idle time, not age: a long upload still receiving chunks stays.
func TestCleanupUsesIdleTimeNotAge(t *testing.T) {
	resetChunkRegistry(t)
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	user := &middleware.SessionUser{ID: "u1", Role: models.RoleUser}

	id := initChunkUpload(t, h, user, "slow.mkv", 30, 3)
	chunkUploadsMu.Lock()
	chunkUploads[id].CreatedAt = time.Now().Add(-30 * time.Hour)
	chunkUploads[id].LastActivity = time.Now().Add(-30 * time.Hour) // idle so far
	chunkUploadsMu.Unlock()
	if rr := postChunk(t, h, user, id, 0, "aaaaaaaaaa"); rr.Code != 200 {
		t.Fatalf("chunk: %d", rr.Code)
	}
	cleanupExpiredChunkUploads()

	chunkUploadsMu.Lock()
	_, ok := chunkUploads[id]
	chunkUploadsMu.Unlock()
	if !ok {
		t.Fatal("upload created 30 h ago but active now was cleaned up")
	}

	// And idle time survives a restart (taken from the newest chunk file).
	tempDir := filepath.Join(h.storage.UploadsDir(), "chunks", id)
	idle := time.Now().Add(-chunkUploadTTL - time.Hour)
	for _, n := range []string{"chunk_0", manifestName} {
		os.Chtimes(filepath.Join(tempDir, n), idle, idle)
	}
	os.Chtimes(tempDir, idle, idle)
	resetChunkRegistry(t)
	RestoreChunkUploads(h.storage.UploadsDir())
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("upload idle past TTL survived restart (err=%v)", err)
	}
}

// A full disk is 507 with a message that says so, not an anonymous 500 the
// client retries into the same full disk.
func TestDiskFullAnswers507(t *testing.T) {
	rr := httptest.NewRecorder()
	chunkWriteFailed(rr, "write chunk 5670", "id", &os.PathError{Op: "write", Path: "chunk_5670", Err: syscall.ENOSPC})
	if rr.Code != http.StatusInsufficientStorage || !strings.Contains(rr.Body.String(), "disk is full") {
		t.Errorf("ENOSPC: %d %q, want 507 'disk is full'", rr.Code, rr.Body.String())
	}
	rr = httptest.NewRecorder()
	chunkWriteFailed(rr, "write", "id", &os.PathError{Op: "write", Path: "x", Err: syscall.EIO})
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("EIO: %d, want 500", rr.Code)
	}
}
