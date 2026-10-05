package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"casadrop/internal/models"
)

// The sender's language is stored with the transfer, because the download and
// expiry mails are sent later from that record (2.5.2).
func TestEmailTransferKeepsLanguage(t *testing.T) {
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Save(&models.Share{ID: "sh0001", FileName: "f", OriginalName: "f.pdf", FileSize: 1,
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveEmailTransfer(&models.EmailTransferRecord{ID: "t1", ShareID: "sh0001", RecipientEmail: "r@example.org",
		SenderEmail: "s@example.org", SentAt: time.Now().UTC().Format(time.RFC3339), Lang: "fr"}); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetEmailTransfersByShare("sh0001")
	if err != nil || len(got) != 1 {
		t.Fatalf("read back: %v, %d records", err, len(got))
	}
	if got[0].Lang != "fr" {
		t.Errorf("Lang = %q, want fr", got[0].Lang)
	}
}

// A database created before 2.5.2 has email_transfers without "lang". Opening
// it must add the column — and must not abort the other migrations (an earlier
// draft of this change did exactly that: the table was missing from the
// migration whitelist and the whole migration rolled back, user_id included).
func TestOldDatabaseGetsLangColumn(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "shares.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE email_transfers (id TEXT PRIMARY KEY, share_id TEXT NOT NULL,
		recipient_email TEXT NOT NULL, recipient_name TEXT, sender_email TEXT NOT NULL, sender_name TEXT,
		title TEXT, message TEXT, notify_download INTEGER DEFAULT 0,
		sent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, downloaded_at DATETIME, notified_at DATETIME)`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	st, err := New(dir)
	if err != nil {
		t.Fatalf("opening a pre-2.5.2 database failed: %v", err)
	}
	defer st.Close()

	db, err = sql.Open("sqlite", filepath.Join(dir, "shares.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, c := range []struct{ table, column string }{{"email_transfers", "lang"}, {"shares", "user_id"}} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, c.table, c.column).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("%s.%s missing after migration", c.table, c.column)
		}
	}
}
