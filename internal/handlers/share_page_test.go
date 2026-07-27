package handlers

import (
	"bytes"
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
)

// setupRealTemplateHandler builds a Handler against the REAL share.html, not
// the stub in createTestTemplates.
//
// This matters more than it looks. The shared harness renders
// `Share: {{.Share.ID}}`, so a test using it can assert nothing about what a
// visitor sees: the entire user-visible payload of the media work — the
// <source type> attribute, which media element is emitted, the playback caveat
// — lives in the real template and would be invisible. Reverting the fix that
// this release exists for left the whole suite green precisely because nothing
// ever rendered the real page.
func setupRealTemplateHandler(t *testing.T) (*Handler, *storage.Storage, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "uploads"), 0o755); err != nil {
		t.Fatalf("uploads dir: %v", err)
	}

	// Repo-relative: internal/handlers → web/templates.
	templatesDir, err := filepath.Abs(filepath.Join("..", "..", "web", "templates"))
	if err != nil {
		t.Fatalf("templates path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(templatesDir, "share.html")); err != nil {
		t.Fatalf("real share.html not found at %s: %v", templatesDir, err)
	}

	store, err := storage.New(tmpDir)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	t.Setenv("DATA_DIR", tmpDir)

	h, err := New(store, templatesDir)
	if err != nil {
		store.Close()
		t.Fatalf("handler: %v", err)
	}

	return h, store, func() { store.Close() }
}

func renderSharePage(t *testing.T, h *Handler, id string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/s/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	rec := httptest.NewRecorder()
	h.SharePage(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("SharePage: status %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.TrimSpace(body) == "" {
		t.Fatal("SharePage: empty body — a 200 with nothing in it is not a rendered page")
	}
	return body
}

func seedShare(t *testing.T, store *storage.Storage, id, name, mime string) {
	t.Helper()
	if err := store.Save(&models.Share{
		ID:           id,
		FileName:     id + filepath.Ext(name),
		OriginalName: name,
		FileSize:     1234,
		MimeType:     mime,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
	}); err != nil {
		t.Fatalf("seed %s: %v", id, err)
	}
}

// TestSharePageRendersMediaCorrectly asserts on what the visitor actually gets
// for the two types this release is about.
func TestSharePageRendersMediaCorrectly(t *testing.T) {
	h, store, cleanup := setupRealTemplateHandler(t)
	defer cleanup()

	seedShare(t, store, "mkv001", "Movie.mkv", "video/x-matroska")
	seedShare(t, store, "flac01", "Track.flac", "audio/flac")
	seedShare(t, store, "wav001", "Sound.wav", "audio/wave")

	t.Run("matroska gets a video element, no type attribute, and the caveat", func(t *testing.T) {
		body := renderSharePage(t, h, "mkv001")
		if !strings.Contains(body, "<video") {
			t.Error("no <video> element — a Matroska share must render a player")
		}
		// The whole point of SourceTypeAttr: browsers reject
		// type="video/x-matroska" without fetching, so it must be omitted.
		if strings.Contains(body, `type="video/x-matroska"`) {
			t.Error(`emitted type="video/x-matroska" — the browser rejects that source unfetched`)
		}
		if !strings.Contains(body, "/stream/mkv001") {
			t.Error("source does not point at /stream/mkv001")
		}
		if !strings.Contains(body, "media-note") {
			t.Error("no playback caveat rendered for a Matroska container")
		}
	})

	t.Run("flac gets an audio element with a usable type attribute", func(t *testing.T) {
		body := renderSharePage(t, h, "flac01")
		if !strings.Contains(body, "<audio") {
			t.Error("no <audio> element — this is the bug 2.4.2 exists to fix")
		}
		if strings.Contains(body, "<video") {
			t.Error("rendered a <video> element for an audio file")
		}
		if !strings.Contains(body, `type="audio/flac"`) {
			t.Error(`missing type="audio/flac" — the browser can evaluate this one`)
		}
	})

	t.Run("wav advertises the spelling browsers accept", func(t *testing.T) {
		body := renderSharePage(t, h, "wav001")
		if !strings.Contains(body, "<audio") {
			t.Error("no <audio> element for a WAV share")
		}
		// Go's sniffer says audio/wave; Chrome's canPlayType rejects that exact
		// string and accepts audio/wav. Emitting the sniffed spelling would make
		// the browser discard the source without fetching it.
		if strings.Contains(body, `type="audio/wave"`) {
			t.Error(`emitted type="audio/wave" — measured: canPlayType rejects it`)
		}
	})

	// No raw template actions must survive into the output on any of them.
	for _, id := range []string{"mkv001", "flac01", "wav001"} {
		body := renderSharePage(t, h, id)
		if strings.Contains(body, "{{") || strings.Contains(body, "<no value>") {
			t.Errorf("%s: unrendered template action or <no value> in the page", id)
		}
	}
}

// TestUploadPersistsRefinedMimeType closes the gap the tests lens found: the
// handlers call utils.DetectMimeType, but nothing asserted that the *stored*
// row carries the refined type. Reverting the call site to a bare
// http.DetectContentType left the suite green.
func TestUploadPersistsRefinedMimeType(t *testing.T) {
	h, store, cleanup := setupRealTemplateHandler(t)
	defer cleanup()

	cases := []struct {
		name    string
		payload []byte
		want    string
		why     string
	}{
		{
			name: "song.flac",
			// fLaC magic: Go's sniffer has no signature for it, so this arrives
			// as application/octet-stream and only the extension can save it.
			payload: append([]byte("fLaC\x00\x00\x00\x22"), make([]byte, 64)...),
			want:    "audio/flac",
			why:     "non-committal sniff must be promoted by extension",
		},
		{
			name:    "untagged.mp3",
			payload: append([]byte{0xFF, 0xFB, 0x90, 0x00}, make([]byte, 64)...),
			want:    "audio/mpeg",
			why:     "MP3 without an ID3 header sniffs as octet-stream",
		},
		{
			name: "evil.flac",
			// Hostile: HTML content wearing an audio extension. The sniffer
			// commits to text/html, which is NOT non-committal, so the extension
			// must not override it.
			payload: []byte("<!DOCTYPE html><html><body><script>alert(1)</script></body></html>"),
			want:    "text/html; charset=utf-8",
			why:     "a recognised type must never be overridden by the file name",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			mw := multipart.NewWriter(&buf)
			fw, err := mw.CreateFormFile("file", c.name)
			if err != nil {
				t.Fatalf("form file: %v", err)
			}
			if _, err := fw.Write(c.payload); err != nil {
				t.Fatalf("write payload: %v", err)
			}
			mw.Close()

			req := httptest.NewRequest(http.MethodPost, "/api/upload", &buf)
			req.Header.Set("Content-Type", mw.FormDataContentType())
			rec := httptest.NewRecorder()
			h.UploadFile(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("upload: status %d, body %q", rec.Code, rec.Body.String())
			}

			shares := store.GetAll()
			var got string
			for _, s := range shares {
				if s.OriginalName == c.name {
					got = s.MimeType
				}
			}
			if got != c.want {
				t.Errorf("stored mime_type = %q, want %q (%s)", got, c.want, c.why)
			}
		})
	}
}
