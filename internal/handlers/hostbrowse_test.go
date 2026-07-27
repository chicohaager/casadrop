package handlers

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// writeTestPNG writes a small but decodable PNG and returns its path.
func writeTestPNG(t *testing.T, path string) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 6))
	for x := 0; x < 8; x++ {
		for y := 0; y < 6; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 30), G: uint8(y * 40), B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write test png: %v", err)
	}
	return path
}

// hostBrowseFixture prepares an allowed root holding one image, one non-image
// and one subdirectory, and points SHARE_ALLOWED_PATHS at it.
func hostBrowseFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	// The handler resolves symlinks; macOS /var → /private/var would otherwise
	// make the allowed-root prefix check fail for reasons unrelated to the test.
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve temp root: %v", err)
	}
	writeTestPNG(t, filepath.Join(resolved, "photo.png"))
	if err := os.WriteFile(filepath.Join(resolved, "notes.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write txt: %v", err)
	}
	if err := os.Mkdir(filepath.Join(resolved, "sub"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("SHARE_ALLOWED_PATHS", resolved)
	return resolved
}

// The browse listing must tell the UI which rows can show a preview — without
// it the dialog can only render a plain file list (reported against v2.4.3).
func TestBrowseFilesMarksImages(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	root := hostBrowseFixture(t)

	req := httptest.NewRequest("GET", "/api/browse?path="+root, nil)
	rec := httptest.NewRecorder()
	handler.BrowseFiles(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("BrowseFiles status = %d, body %q", rec.Code, rec.Body.String())
	}

	var resp struct {
		Entries []struct {
			Name    string `json:"name"`
			IsDir   bool   `json:"is_dir"`
			IsImage bool   `json:"is_image"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode browse response: %v", err)
	}

	want := map[string]bool{"photo.png": true, "notes.txt": false, "sub": false}
	seen := map[string]bool{}
	for _, e := range resp.Entries {
		seen[e.Name] = true
		if got, ok := want[e.Name]; ok && e.IsImage != got {
			t.Errorf("%s: is_image = %v, want %v", e.Name, e.IsImage, got)
		}
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("entry %q missing from listing", name)
		}
	}
}

func TestHostThumbnailServesImage(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	root := hostBrowseFixture(t)

	req := httptest.NewRequest("GET", "/api/browse/thumbnail?path="+filepath.Join(root, "photo.png"), nil)
	rec := httptest.NewRecorder()
	handler.HostThumbnail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("Content-Type = %q, want image/jpeg", ct)
	}
	// Assert on the payload, not just the status: a 200 with an empty or
	// undecodable body would still leave the dialog without previews.
	if _, format, err := image.Decode(bytes.NewReader(rec.Body.Bytes())); err != nil || format != "jpeg" {
		t.Errorf("body is not a decodable JPEG: format=%q err=%v (%d bytes)", format, err, rec.Body.Len())
	}
	if cc := rec.Header().Get("Cache-Control"); cc == "" || cc[:7] != "private" {
		t.Errorf("Cache-Control = %q, want a private policy (admin-only content)", cc)
	}
}

func TestHostThumbnailRejects(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	root := hostBrowseFixture(t)
	outside := filepath.Join(t.TempDir(), "elsewhere.png")
	writeTestPNG(t, outside)

	cases := []struct {
		name string
		path string
		want int
	}{
		{"outside allowed roots", outside, http.StatusForbidden},
		{"traversal", root + "/../etc/passwd", http.StatusBadRequest},
		{"relative path", "photo.png", http.StatusBadRequest},
		{"empty path", "", http.StatusBadRequest},
		{"missing file", filepath.Join(root, "nope.png"), http.StatusBadRequest},
		{"directory", filepath.Join(root, "sub"), http.StatusBadRequest},
		{"not an image", filepath.Join(root, "notes.txt"), http.StatusUnsupportedMediaType},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/browse/thumbnail?path="+tc.path, nil)
			rec := httptest.NewRecorder()
			handler.HostThumbnail(rec, req)
			if rec.Code != tc.want {
				t.Errorf("status = %d, want %d (body %q)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}
