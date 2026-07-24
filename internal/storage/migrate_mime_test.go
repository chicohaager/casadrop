package storage

import (
	"os"
	"testing"
)

// TestMigrateMimeTypes seeds a database the way an older version would have
// written it, reopens it, and checks that startup corrected exactly the rows
// that were wrong — and nothing else.
func TestMigrateMimeTypes(t *testing.T) {
	dir, err := os.MkdirTemp("", "casadrop-mime-migration-*")
	if err != nil {
		t.Fatalf("tmpdir: %v", err)
	}
	defer os.RemoveAll(dir)

	// The migration lives on the SQLite backend, so drive that directly rather
	// than through the Storage facade.
	s, err := NewSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}

	// Rows exactly as http.DetectContentType would have classified them.
	seed := []struct {
		id, name, mime string
	}{
		{"mkv1", "Movie.mkv", "video/webm"},                         // → video/x-matroska
		{"flac1", "06 Mind's Eye.flac", "application/octet-stream"}, // → audio/flac
		{"mp3untagged", "song.mp3", "application/octet-stream"},     // → audio/mpeg
		{"opus1", "podcast.opus", "application/ogg"},                // → audio/ogg
		{"webm1", "clip.webm", "video/webm"},                        // unchanged: really WebM
		{"png1", "photo.png", "image/png"},                          // unchanged: recognised
		{"html1", "page.html", "text/html; charset=utf-8"},          // unchanged: recognised
		{"zip1", "archive.zip", "application/octet-stream"},         // unchanged: not media
		{"evil", "evil.svg", "application/octet-stream"},            // unchanged: active type never reached
	}
	for _, r := range seed {
		if _, err := s.db.Exec(
			`INSERT INTO shares (id, file_name, original_name, file_size, mime_type, expires_at)
			 VALUES (?, ?, ?, 1, ?, datetime('now', '+1 day'))`,
			r.id, r.id, r.name, r.mime,
		); err != nil {
			t.Fatalf("seed %s: %v", r.id, err)
		}
	}

	if err := s.migrateMimeTypes(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	want := map[string]string{
		"mkv1":        "video/x-matroska",
		"flac1":       "audio/flac",
		"mp3untagged": "audio/mpeg",
		"opus1":       "audio/ogg",
		"webm1":       "video/webm",
		"png1":        "image/png",
		"html1":       "text/html; charset=utf-8",
		"zip1":        "application/octet-stream",
		"evil":        "application/octet-stream",
	}
	for id, expect := range want {
		var got string
		if err := s.db.QueryRow(`SELECT mime_type FROM shares WHERE id = ?`, id).Scan(&got); err != nil {
			t.Fatalf("read %s: %v", id, err)
		}
		if got != expect {
			t.Errorf("share %s: mime_type = %q, want %q", id, got, expect)
		}
	}

	// Idempotent: a second run must change nothing.
	if err := s.migrateMimeTypes(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	var still string
	if err := s.db.QueryRow(`SELECT mime_type FROM shares WHERE id = 'mkv1'`).Scan(&still); err != nil {
		t.Fatal(err)
	}
	if still != "video/x-matroska" {
		t.Errorf("after second run: %q, want video/x-matroska", still)
	}

	s.Close()
}
