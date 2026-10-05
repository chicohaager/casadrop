package storage

import (
	"bytes"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// Since 2.5.2 a failed column migration refuses the start instead of logging a
// warning and running without the columns. Trigger: a users table whose
// oidc_subject is NOT NULL with a legacy ” row, so the normalising UPDATE in
// the same transaction fails and rolls back the lang column added before it —
// exactly the state that used to start "fine" and then lose every mail record.
func TestFailedColumnMigrationRefusesStart(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "shares.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT UNIQUE NOT NULL, name TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'viewer', password_hash TEXT, oidc_subject TEXT NOT NULL, oidc_issuer TEXT NOT NULL,
		is_active INTEGER NOT NULL DEFAULT 1, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, last_login_at DATETIME);
		INSERT INTO users (id, email, name, oidc_subject, oidc_issuer) VALUES ('u1', 'a@example.org', 'A', '', '')`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	logs := captureLog(t)
	st, err := New(dir)
	if err == nil {
		st.Close()
		t.Fatal("start succeeded although the column migration failed")
	}
	// The transaction rolled back, so the log must not claim added columns.
	if strings.Contains(logs.String(), "Added column") {
		t.Errorf("log reports columns that were rolled back:\n%s", logs.String())
	}
	if !strings.Contains(err.Error(), "migration failed") || !strings.Contains(err.Error(), "NOT NULL") {
		t.Errorf("error does not say what failed: %v", err)
	}
}

func writeSharesJSON(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "shares.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A shares.json that cannot be migrated must stop the start and leave no
// database behind — otherwise the next start finds a database, skips the
// migration for good and the shares look lost.
func TestFailedJSONMigrationKeepsJSONAndRetries(t *testing.T) {
	for name, body := range map[string]string{
		"unreadable":    `{not json`,
		"one bad entry": `{"ok0001":{"id":"ok0001","file_name":"f","original_name":"f.pdf","file_size":1,"expires_at":"2030-01-01T00:00:00Z"},"bad001":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeSharesJSON(t, dir, body)

			st, err := New(dir)
			if err == nil {
				st.Close()
				t.Fatal("start succeeded although shares.json could not be migrated")
			}
			for _, f := range []string{"shares.db", "shares.db-wal", "shares.db-shm", "shares.json.backup"} {
				if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
					t.Errorf("%s exists after a failed migration", f)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, "shares.json")); err != nil {
				t.Errorf("shares.json gone after a failed migration: %v", err)
			}
			if !CheckMigrationNeeded(dir) {
				t.Error("next start would not retry the migration")
			}
		})
	}
}

// Positive path: a valid shares.json is migrated completely and backed up.
func TestJSONMigrationSucceeds(t *testing.T) {
	dir := t.TempDir()
	writeSharesJSON(t, dir, `{"ok0001":{"id":"ok0001","file_name":"f","original_name":"f.pdf","file_size":1,"expires_at":"2030-01-01T00:00:00Z"}}`)
	st, err := New(dir)
	if err != nil {
		t.Fatalf("valid shares.json: %v", err)
	}
	defer st.Close()
	if _, ok := st.Get("ok0001"); !ok {
		t.Error("migrated share not found")
	}
	if _, err := os.Stat(filepath.Join(dir, "shares.json.backup")); err != nil {
		t.Errorf("no backup after successful migration: %v", err)
	}
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

// Counterpart: a successful migration still reports what it added.
func TestSuccessfulMigrationLogsAddedColumns(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "shares.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE email_transfers (id TEXT PRIMARY KEY, share_id TEXT NOT NULL,
		recipient_email TEXT NOT NULL, sender_email TEXT NOT NULL, sent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	logs := captureLog(t)
	st, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
	if !strings.Contains(logs.String(), "Added column lang to table email_transfers") {
		t.Errorf("committed column not logged:\n%s", logs.String())
	}
}
