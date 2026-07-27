package storage

import (
	"os"
	"testing"
)

// TestMigrateMimeTypes seeds a database the way an older version would have
// written it and checks that the migration corrects exactly the rows that were
// wrong — and nothing else.
//
// Note what this does NOT prove: it calls s.migrateMimeTypes() directly, so it
// stays green even if the call site in NewSQLiteStorage were removed. Wiring
// coverage is TestMigrateMimeTypesRunsAtStartup below.
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

// TestRefineMimeColumnReportsErrors pins the fix for a swallowed error: the
// query used to return (0, nil) for *any* failure, on the theory that a table
// might not exist yet. It always exists (initBaseSchema creates all three and
// is fatal on failure), so the only thing that catch-all could hide was a real
// error — and it hid it perfectly, because with updated == 0 the summary log
// line is skipped too. A broken migration then looked exactly like a clean one.
func TestRefineMimeColumnReportsErrors(t *testing.T) {
	dir, err := os.MkdirTemp("", "casadrop-mime-err-*")
	if err != nil {
		t.Fatalf("tmpdir: %v", err)
	}
	defer os.RemoveAll(dir)

	s, err := NewSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	defer s.Close()

	tx, err := s.db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()

	for _, c := range []struct{ table, col, why string }{
		{"no_such_table", "original_name", "missing table"},
		{"shares", "no_such_column", "missing column"},
	} {
		n, err := refineMimeColumn(tx, c.table, c.col)
		if err == nil {
			t.Errorf("%s: refineMimeColumn returned no error — a failed migration is invisible", c.why)
		}
		if n != 0 {
			t.Errorf("%s: count = %d, want 0", c.why, n)
		}
	}
}

// TestMigrateMimeTypesRunsAtStartup is the wiring test: it writes a wrong row
// into a database, closes it, and opens it again through the same constructor
// production uses. Deleting the migration call in NewSQLiteStorage fails here,
// which is precisely what TestMigrateMimeTypes cannot notice.
func TestMigrateMimeTypesRunsAtStartup(t *testing.T) {
	dir, err := os.MkdirTemp("", "casadrop-mime-startup-*")
	if err != nil {
		t.Fatalf("tmpdir: %v", err)
	}
	defer os.RemoveAll(dir)

	first, err := NewSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	// One row per table the migration walks, so dropping any of the three from
	// the loop fails here too.
	if _, err := first.db.Exec(
		`INSERT INTO shares (id, file_name, original_name, file_size, mime_type, expires_at)
		 VALUES ('s1', 's1', 'Movie.mkv', 1, 'video/webm', datetime('now', '+1 day'))`); err != nil {
		t.Fatalf("seed shares: %v", err)
	}
	if _, err := first.db.Exec(
		`INSERT INTO receive_links (id, name) VALUES ('rl1', 'link')`); err != nil {
		t.Fatalf("seed receive_links: %v", err)
	}
	if _, err := first.db.Exec(
		`INSERT INTO received_files (id, receive_link_id, file_name, original_name, file_size, mime_type)
		 VALUES ('rf1', 'rl1', 'rf1', 'take.flac', 1, 'application/octet-stream')`); err != nil {
		t.Fatalf("seed received_files: %v", err)
	}
	if _, err := first.db.Exec(
		`INSERT INTO folder_contents (id, share_id, relative_path, file_name, file_size, mime_type)
		 VALUES ('fc1', 's1', 'a/b', 'track.opus', 1, 'application/ogg')`); err != nil {
		t.Fatalf("seed folder_contents: %v", err)
	}
	first.Close()

	// Reopen: this is the path production takes on every container start.
	second, err := NewSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer second.Close()

	for _, c := range []struct{ query, id, want string }{
		{`SELECT mime_type FROM shares WHERE id = ?`, "s1", "video/x-matroska"},
		{`SELECT mime_type FROM received_files WHERE id = ?`, "rf1", "audio/flac"},
		{`SELECT mime_type FROM folder_contents WHERE id = ?`, "fc1", "audio/ogg"},
	} {
		var got string
		if err := second.db.QueryRow(c.query, c.id).Scan(&got); err != nil {
			t.Fatalf("read %s: %v", c.id, err)
		}
		if got != c.want {
			t.Errorf("%s: mime_type = %q, want %q — startup migration did not reach this table", c.id, got, c.want)
		}
	}
}
