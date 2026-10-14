package handlers

import (
	"bytes"
	"context"
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

	"casadrop/internal/auth"
	"casadrop/internal/models"
)

// multipartBody builds a form with one file part plus extra fields after it
// (receive.js appends the password AFTER the file).
func multipartBody(t *testing.T, field, name string, content []byte, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	part, err := mw.CreateFormFile(field, name)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(content)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	mw.Close()
	return body, mw.FormDataContentType()
}

// saveLink stores a receive link and creates its directory, as the create
// handler (CreateReceiveLink) does.
func saveLink(t *testing.T, h *Handler, link *models.ReceiveLink) {
	t.Helper()
	if err := h.storage.SaveReceiveLink(context.Background(), link); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(h.storage.UploadsDir(), "received", link.ID), 0755); err != nil {
		t.Fatal(err)
	}
}

func receiveRouter(h *Handler) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/r/{id}/upload", h.ReceiveUpload).Methods("POST")
	return r
}

func postReceive(h *Handler, id string, body *bytes.Buffer, ctype string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/r/"+id+"/upload", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	receiveRouter(h).ServeHTTP(rec, req)
	return rec
}

// r.FormValue("password") ran before MaxBytesReader was installed and parsed
// the whole body itself, so a password-protected link accepted files of any
// size. The size cap must hold with and without a password.
func TestReceiveSizeLimitHoldsForPasswordLinks(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	hash, _ := auth.HashPassword("geheim123")
	for _, withPW := range []bool{false, true} {
		link := &models.ReceiveLink{ID: "small", Name: "small", MaxFileSize: 1000, CreatedAt: time.Now()}
		fields := map[string]string{}
		if withPW {
			link.ID = "small-pw"
			link.HasPassword = true
			link.Password = hash
			fields["password"] = "geheim123"
		}
		saveLink(t, h, link)
		body, ct := multipartBody(t, "file", "big.txt", bytes.Repeat([]byte("x"), 50_000), fields)
		rec := postReceive(h, link.ID, body, ct)
		if rec.Code == http.StatusOK {
			t.Errorf("password=%v: 50 KB file on a 1 KB link accepted (200)", withPW)
		}
		files, _ := h.storage.GetReceivedFiles(context.Background(), link.ID)
		if len(files) != 0 {
			t.Errorf("password=%v: %d file(s) stored despite size cap", withPW, len(files))
		}

		// Positive control: a file under the cap goes through.
		small, ct := multipartBody(t, "file", "ok.txt", []byte("small"), fields)
		if rec := postReceive(h, link.ID, small, ct); rec.Code != http.StatusOK {
			t.Errorf("password=%v: small file: %d %s", withPW, rec.Code, rec.Body.String())
		}
	}
}

func TestReceiveRefusesWhenDiskTooSmall(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	saveLink(t, h, &models.ReceiveLink{ID: "rl", Name: "rl", CreatedAt: time.Now()})
	stubFreeDisk(t, chunkDiskReserve) // nothing left above the reserve

	body, ct := multipartBody(t, "file", "a.txt", []byte("hello"), nil)
	rec := postReceive(h, "rl", body, ct)
	if rec.Code != http.StatusInsufficientStorage {
		t.Fatalf("receive on full disk: %d %s, want 507", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || !strings.Contains(resp["error"], "disk space") {
		t.Errorf("error not a translated diskFull JSON: %q", rec.Body.String())
	}

	stubFreeDisk(t, chunkDiskReserve+1<<20) // positive control
	body, ct = multipartBody(t, "file", "a.txt", []byte("hello"), nil)
	if rec := postReceive(h, "rl", body, ct); rec.Code != http.StatusOK {
		t.Errorf("receive with room: %d %s", rec.Code, rec.Body.String())
	}
}

// Auto-share used to copy the received file, so it sat on disk twice.
func TestReceiveAutoShareIsHardLink(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	saveLink(t, h, &models.ReceiveLink{ID: "as", Name: "as", AutoShare: true, CreatedAt: time.Now()})

	body, ct := multipartBody(t, "file", "doc.txt", []byte("auto shared content"), nil)
	if rec := postReceive(h, "as", body, ct); rec.Code != http.StatusOK {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	files, _ := h.storage.GetReceivedFiles(context.Background(), "as")
	if len(files) != 1 || files[0].ShareID == "" {
		t.Fatalf("want 1 received file with auto-share, got %+v", files)
	}
	share, ok := h.storage.Get(context.Background(), files[0].ShareID)
	if !ok {
		t.Fatal("auto-share not stored")
	}
	recvInfo, err1 := os.Stat(filepath.Join(h.storage.UploadsDir(), "received", "as", files[0].FileName))
	shareInfo, err2 := os.Stat(filepath.Join(h.storage.UploadsDir(), share.FileName))
	if err1 != nil || err2 != nil {
		t.Fatalf("stat: %v / %v", err1, err2)
	}
	if !os.SameFile(recvInfo, shareInfo) {
		t.Error("auto-share is a separate copy, not a hard link — the upload is on disk twice")
	}
}

// A spilled multipart part must become the share file without a copy.
func TestPlaceUploadedFileLinksSpilledPart(t *testing.T) {
	dir := t.TempDir()
	spill, err := os.CreateTemp(dir, "multipart-")
	if err != nil {
		t.Fatal(err)
	}
	spill.WriteString("spilled bytes")
	spill.Seek(0, 0)
	before, _ := spill.Stat()

	dest, size, err := placeUploadedFile(spill, filepath.Join(dir, "share.bin"))
	if err != nil {
		t.Fatal(err)
	}
	defer dest.Close()
	after, _ := dest.Stat()
	if !os.SameFile(before, after) {
		t.Error("spilled part was copied instead of hard-linked")
	}
	if size != int64(len("spilled bytes")) {
		t.Errorf("size = %d", size)
	}
	if _, err := os.Stat(spill.Name()); !os.IsNotExist(err) {
		t.Errorf("spill file still present (err=%v)", err)
	}
	if after.Mode().Perm() != 0644 {
		t.Errorf("mode = %v, want 0644 like createShareFile", after.Mode().Perm())
	}
}

// End to end: a file above the 8 MB memory budget spills into the data-disk
// temp dir and leaves nothing behind there.
func TestUploadFileLargeSpillLeavesNoTemp(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	t.Setenv("TMPDIR", os.TempDir()) // restored after the test
	tmp, err := SetupMultipartTempDir(h.storage.UploadsDir())
	if err != nil {
		t.Fatal(err)
	}
	content := bytes.Repeat([]byte("0123456789abcdef"), (9<<20)/16) // 9 MB > budget
	body, ct := multipartBody(t, "file", "big.txt", content, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	h.UploadFile(rec, req)
	if req.MultipartForm != nil {
		req.MultipartForm.RemoveAll() // what net/http does after the handler
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		ID string `json:"id"`
	}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	got, err := os.ReadFile(filepath.Join(h.storage.UploadsDir(), resp.ID+".txt"))
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("stored file differs (err=%v, %d bytes)", err, len(got))
	}
	left, _ := os.ReadDir(tmp)
	if len(left) != 0 {
		t.Errorf("%d file(s) left in multipart temp dir", len(left))
	}
}

func TestSetupMultipartTempDirRemovesLeftovers(t *testing.T) {
	t.Setenv("TMPDIR", os.TempDir())
	uploads := t.TempDir()
	dir := filepath.Join(uploads, multipartTempName)
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "multipart-123"), []byte("crash leftover"), 0600)
	os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("not ours"), 0600)

	got, err := SetupMultipartTempDir(uploads)
	if err != nil {
		t.Fatal(err)
	}
	if got != dir || os.TempDir() != dir {
		t.Errorf("temp dir = %q, os.TempDir() = %q, want %q", got, os.TempDir(), dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "multipart-123")); !os.IsNotExist(err) {
		t.Error("leftover multipart file not removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err != nil {
		t.Error("unrelated file removed")
	}
}

func TestAdminUploadPathsRefuseWhenDiskTooSmall(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	root := hostBrowseFixture(t)
	cases := []struct {
		name string
		do   func() *httptest.ResponseRecorder
	}{
		{"upload", func() *httptest.ResponseRecorder {
			body, ct := multipartBody(t, "file", "a.txt", []byte("hello"), nil)
			req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
			req.Header.Set("Content-Type", ct)
			rec := httptest.NewRecorder()
			h.UploadFile(rec, req)
			return rec
		}},
		{"multi", func() *httptest.ResponseRecorder {
			body, ct := multipartBody(t, "files", "a.txt", []byte("hello"), nil)
			req := httptest.NewRequest(http.MethodPost, "/api/upload/multi", body)
			req.Header.Set("Content-Type", ct)
			rec := httptest.NewRecorder()
			h.UploadMultipleFiles(rec, req)
			return rec
		}},
		{"share-from-path copy", func() *httptest.ResponseRecorder {
			js, _ := json.Marshal(models.ShareFromPathRequest{Path: filepath.Join(root, "notes.txt"), ExpiresIn: 1})
			req := httptest.NewRequest(http.MethodPost, "/api/share-from-path", bytes.NewReader(js))
			rec := httptest.NewRecorder()
			h.ShareFromPath(rec, req)
			return rec
		}},
	}
	for _, c := range cases {
		stubFreeDisk(t, chunkDiskReserve)
		if rec := c.do(); rec.Code != http.StatusInsufficientStorage {
			t.Errorf("%s on full disk: %d %q, want 507", c.name, rec.Code, rec.Body.String())
		}
		stubFreeDisk(t, chunkDiskReserve+1<<20) // positive control
		if rec := c.do(); rec.Code != http.StatusOK {
			t.Errorf("%s with room: %d %q, want 200", c.name, rec.Code, rec.Body.String())
		}
	}
}
