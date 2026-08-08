package storage

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	// Pure-Go SQLite driver. Registers driver name "sqlite".
	// CGO is not required — this lets us ship a fully static binary in a
	// scratch container.
	_ "modernc.org/sqlite"

	"casadrop/internal/models"
)

// SQLiteStorage implements StorageBackend using SQLite
type SQLiteStorage struct {
	db         *sql.DB
	dataDir    string
	uploadsDir string

	stopCleanup chan struct{}
	stopOnce    sync.Once
}

// NewSQLiteStorage creates a new SQLite storage backend
func NewSQLiteStorage(dataDir string) (*SQLiteStorage, error) {
	uploadsDir := filepath.Join(dataDir, "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "shares.db")
	// WAL mode for best concurrent read/write performance.
	//
	// modernc.org/sqlite uses `_pragma=<name>(<value>)` query params instead
	// of the `_journal_mode` style that mattn/go-sqlite3 supports.
	dsn := "file:" + dbPath +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Configure connection pool for SQLite
	// SQLite handles concurrency via WAL mode, but we still set reasonable limits
	// WAL mode allows concurrent readers alongside a single writer.
	// Setting MaxOpenConns > 1 lets read queries run in parallel while
	// writes are serialised by SQLite's internal locking.
	db.SetMaxOpenConns(4)                   // Allow concurrent WAL readers
	db.SetMaxIdleConns(4)                   // Keep connections ready for reuse
	db.SetConnMaxLifetime(time.Hour)        // Recycle connections hourly
	db.SetConnMaxIdleTime(15 * time.Minute) // Close idle connections after 15 min

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	// Ensure WAL mode is set (in case DB was created with different mode)
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		log.Printf("Warning: Failed to set WAL mode: %v", err)
	}

	// Checkpoint WAL on startup to consolidate any pending writes
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		log.Printf("Warning: Failed to checkpoint WAL on startup: %v", err)
	}

	s := &SQLiteStorage{
		db:          db,
		dataDir:     dataDir,
		uploadsDir:  uploadsDir,
		stopCleanup: make(chan struct{}),
	}

	// Initialize base schema (without user_id indexes that might fail on old DBs)
	if err := s.initBaseSchema(); err != nil {
		return nil, err
	}

	// Fix timezone format in existing dates (migration)
	s.fixDateTimeFormats()

	// Run user migration for existing databases (adds user_id columns if needed)
	if err := s.migrateExistingTables(); err != nil {
		log.Printf("Warning: User migration failed: %v", err)
		// Continue anyway - columns might already exist
	}

	// Create indexes that depend on migrated columns
	if err := s.createUserIndexes(); err != nil {
		log.Printf("Warning: Failed to create user indexes: %v", err)
	}

	// Correct mime_type rows written before the sniffer got extension-based
	// disambiguation (FLAC/M4A/untagged MP3 stored as octet-stream and thus
	// previewable nowhere; Matroska stored as video/webm).
	if err := s.migrateMimeTypes(); err != nil {
		log.Printf("Warning: MIME migration failed: %v", err)
	}

	// Start cleanup goroutine
	go s.cleanupExpired()

	return s, nil
}

// initBaseSchema creates the database tables if they don't exist
// Note: user_id columns and indexes are handled by migration for compatibility
func (s *SQLiteStorage) initBaseSchema() error {
	schema := `
	-- Users table (v2.1 multi-user)
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'viewer',
		password_hash TEXT,
		oidc_subject TEXT,
		oidc_issuer TEXT,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_login_at DATETIME,
		quota_bytes INTEGER NOT NULL DEFAULT 0,
		UNIQUE(oidc_subject, oidc_issuer)
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_oidc ON users(oidc_subject, oidc_issuer);

	-- Shares table (without user_id - added by migration)
	CREATE TABLE IF NOT EXISTS shares (
		id TEXT PRIMARY KEY,
		file_name TEXT NOT NULL,
		original_name TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		mime_type TEXT,
		password_hash TEXT,
		has_password INTEGER NOT NULL DEFAULT 0,
		expires_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		downloads INTEGER NOT NULL DEFAULT 0,
		max_downloads INTEGER NOT NULL DEFAULT 0,
		source_path TEXT,
		is_symlink INTEGER NOT NULL DEFAULT 0,
		is_directory INTEGER NOT NULL DEFAULT 0,
		parent_share_id TEXT,
		total_files INTEGER DEFAULT 0,
		total_size INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_shares_expires_at ON shares(expires_at);
	CREATE INDEX IF NOT EXISTS idx_shares_created_at ON shares(created_at);
	CREATE INDEX IF NOT EXISTS idx_shares_parent ON shares(parent_share_id);

	-- Folder contents table (for folder shares)
	CREATE TABLE IF NOT EXISTS folder_contents (
		id TEXT PRIMARY KEY,
		share_id TEXT NOT NULL,
		relative_path TEXT NOT NULL,
		file_name TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		mime_type TEXT,
		is_directory INTEGER NOT NULL DEFAULT 0,
		FOREIGN KEY (share_id) REFERENCES shares(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_folder_contents_share ON folder_contents(share_id);
	CREATE INDEX IF NOT EXISTS idx_folder_contents_share_path ON folder_contents(share_id, relative_path);

	-- Receive links table (without user_id - added by migration)
	CREATE TABLE IF NOT EXISTS receive_links (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		password_hash TEXT,
		has_password INTEGER NOT NULL DEFAULT 0,
		expires_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		max_uploads INTEGER NOT NULL DEFAULT 0,
		max_file_size INTEGER NOT NULL DEFAULT 0,
		allowed_extensions TEXT,
		auto_share INTEGER NOT NULL DEFAULT 0,
		webhook_url TEXT,
		current_uploads INTEGER NOT NULL DEFAULT 0,
		total_size INTEGER NOT NULL DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_receive_links_expires ON receive_links(expires_at);

	-- Received files table
	CREATE TABLE IF NOT EXISTS received_files (
		id TEXT PRIMARY KEY,
		receive_link_id TEXT NOT NULL,
		file_name TEXT NOT NULL,
		original_name TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		mime_type TEXT,
		uploader_ip TEXT,
		uploader_agent TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		share_id TEXT,
		FOREIGN KEY (receive_link_id) REFERENCES receive_links(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_received_files_link ON received_files(receive_link_id);
	CREATE INDEX IF NOT EXISTS idx_received_files_link_created ON received_files(receive_link_id, created_at);

	-- SMTP Config table (single row, id=1)
	CREATE TABLE IF NOT EXISTS smtp_config (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		enabled INTEGER NOT NULL DEFAULT 0,
		host TEXT,
		port INTEGER DEFAULT 587,
		username TEXT,
		password TEXT,
		from_email TEXT,
		from_name TEXT,
		use_tls INTEGER DEFAULT 0,
		use_starttls INTEGER DEFAULT 1,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Email transfers table (for tracking)
	CREATE TABLE IF NOT EXISTS email_transfers (
		id TEXT PRIMARY KEY,
		share_id TEXT NOT NULL,
		recipient_email TEXT NOT NULL,
		recipient_name TEXT,
		sender_email TEXT NOT NULL,
		sender_name TEXT,
		title TEXT,
		message TEXT,
		notify_download INTEGER DEFAULT 0,
		sent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		downloaded_at DATETIME,
		notified_at DATETIME,
		FOREIGN KEY (share_id) REFERENCES shares(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_email_transfers_share ON email_transfers(share_id);
	CREATE INDEX IF NOT EXISTS idx_email_transfers_recipient ON email_transfers(recipient_email);

	-- API Keys table (v2.2)
	CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		key_hash TEXT NOT NULL,
		prefix TEXT NOT NULL,
		user_id TEXT,
		role TEXT NOT NULL DEFAULT 'admin',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME,
		is_active INTEGER NOT NULL DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
	`

	_, err := s.db.Exec(schema)
	return err
}

// createUserIndexes creates indexes on user_id columns after migration
func (s *SQLiteStorage) createUserIndexes() error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_shares_user ON shares(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_receive_links_user ON receive_links(user_id)",
	}

	for _, idx := range indexes {
		if _, err := s.db.Exec(idx); err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
			// Continue with other indexes
		}
	}

	return nil
}

// UploadsDir returns the uploads directory path
func (s *SQLiteStorage) UploadsDir() string {
	return s.uploadsDir
}

// Ping verifies the database connection is alive (used by readiness checks).
// Ping verifies the database is actually readable, not merely that a
// connection object exists.
//
// It used to call db.Ping(), which on the modernc driver issues `select 1` — a
// statement the SQL engine answers without ever looking at application data, so
// it stayed green for a database whose schema was gone. Reading a row from a
// real table exercises the pool, the schema and the b-tree; an empty table is
// fine and is not an error. The short context stops a locked database from
// hanging the probe until the caller gives up.
//
// What this still cannot see, measured rather than assumed: a database file
// clobbered *underneath* an open connection. SQLite answers from its warm page
// cache, so overwriting shares.db with garbage leaves this check green until
// something forces an uncached read. Detecting that would mean reopening the
// file on every probe, which is not a reasonable thing to do every 30 seconds.
// The limitation is pinned in TestReadyzQueriesTheRealTable so nobody has to
// rediscover it.
func (s *SQLiteStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM shares LIMIT 1`).Scan(&id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}

// Close stops the background cleanup goroutine and closes the database.
func (s *SQLiteStorage) Close() error {
	s.stopOnce.Do(func() { close(s.stopCleanup) })
	return s.db.Close()
}

// ============= Share Operations =============

// sqliteTimeFormat is the single textual datetime format written to every
// timestamp column. Storing all timestamps the same way keeps the
// substr(...,1,19) + datetime() expiry comparisons valid regardless of how the
// SQL driver might otherwise serialize a time.Time (e.g. RFC3339 with 'T').
const sqliteTimeFormat = "2006-01-02 15:04:05"

func fmtTime(t time.Time) string { return t.UTC().Format(sqliteTimeFormat) }

func fmtTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format(sqliteTimeFormat)
}

// Save saves or updates a share
func (s *SQLiteStorage) Save(share *models.Share) error {
	query := `
		INSERT INTO shares (
			id, file_name, original_name, file_size, mime_type,
			password_hash, has_password, expires_at, created_at,
			downloads, max_downloads, source_path, is_symlink,
			is_directory, parent_share_id, total_files, total_size,
			user_id, user_email
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			file_name=excluded.file_name,
			original_name=excluded.original_name,
			file_size=excluded.file_size,
			mime_type=excluded.mime_type,
			password_hash=excluded.password_hash,
			has_password=excluded.has_password,
			expires_at=excluded.expires_at,
			created_at=excluded.created_at,
			downloads=excluded.downloads,
			max_downloads=excluded.max_downloads,
			source_path=excluded.source_path,
			is_symlink=excluded.is_symlink,
			is_directory=excluded.is_directory,
			parent_share_id=excluded.parent_share_id,
			total_files=excluded.total_files,
			total_size=excluded.total_size,
			user_id=excluded.user_id,
			user_email=excluded.user_email
	`

	// Format times as SQLite-compatible strings (without timezone)
	expiresAt := share.ExpiresAt.UTC().Format("2006-01-02 15:04:05")
	createdAt := share.CreatedAt.UTC().Format("2006-01-02 15:04:05")

	_, err := s.db.Exec(query,
		share.ID, share.FileName, share.OriginalName, share.FileSize, share.MimeType,
		share.Password, share.HasPassword, expiresAt, createdAt,
		share.Downloads, share.MaxDownloads, share.SourcePath, share.IsSymlink,
		share.IsDirectory, share.ParentShareID, share.TotalFiles, share.TotalSize,
		share.UserID, share.UserEmail,
	)
	return err
}

// Get retrieves a share by ID
func (s *SQLiteStorage) Get(id string) (*models.Share, bool) {
	// Use strftime to normalize datetime comparison - handles both formats
	query := `
		SELECT id, file_name, original_name, file_size, mime_type,
			   password_hash, has_password, expires_at, created_at,
			   downloads, max_downloads, source_path, is_symlink,
			   is_directory, COALESCE(parent_share_id, ''), total_files, total_size,
			   COALESCE(user_id, ''), COALESCE(user_email, '')
		FROM shares
		WHERE id = ? AND datetime(substr(expires_at, 1, 19)) > datetime('now')
	`

	share := &models.Share{}
	err := s.db.QueryRow(query, id).Scan(
		&share.ID, &share.FileName, &share.OriginalName, &share.FileSize, &share.MimeType,
		&share.Password, &share.HasPassword, &share.ExpiresAt, &share.CreatedAt,
		&share.Downloads, &share.MaxDownloads, &share.SourcePath, &share.IsSymlink,
		&share.IsDirectory, &share.ParentShareID, &share.TotalFiles, &share.TotalSize,
		&share.UserID, &share.UserEmail,
	)

	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("Error getting share %s: %v", id, err)
		return nil, false
	}

	return share, true
}

// GetAll returns all non-expired shares
func (s *SQLiteStorage) GetAll() []*models.Share {
	query := `
		SELECT id, file_name, original_name, file_size, mime_type,
			   password_hash, has_password, expires_at, created_at,
			   downloads, max_downloads, source_path, is_symlink,
			   is_directory, COALESCE(parent_share_id, ''), total_files, total_size,
			   COALESCE(user_id, ''), COALESCE(user_email, '')
		FROM shares
		WHERE datetime(substr(expires_at, 1, 19)) > datetime('now')
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("Error getting all shares: %v", err)
		return nil
	}
	defer rows.Close()

	var shares []*models.Share
	for rows.Next() {
		share := &models.Share{}
		if err := rows.Scan(
			&share.ID, &share.FileName, &share.OriginalName, &share.FileSize, &share.MimeType,
			&share.Password, &share.HasPassword, &share.ExpiresAt, &share.CreatedAt,
			&share.Downloads, &share.MaxDownloads, &share.SourcePath, &share.IsSymlink,
			&share.IsDirectory, &share.ParentShareID, &share.TotalFiles, &share.TotalSize,
			&share.UserID, &share.UserEmail,
		); err != nil {
			log.Printf("Error scanning share: %v", err)
			continue
		}
		shares = append(shares, share)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating shares: %v", err)
	}

	return shares
}

// Delete removes a share and its file
func (s *SQLiteStorage) Delete(id string) error {
	// Get the share first to delete the file
	share, ok := s.Get(id)
	if !ok {
		// Still try to delete from DB in case it's expired
		_, err := s.db.Exec("DELETE FROM shares WHERE id = ?", id)
		return err
	}

	// Delete the file. Folder shares store FileName="" (they reference
	// SourcePath in place) — filepath.Join(uploadsDir, "") is the uploads
	// directory itself, and os.Remove would delete it when it happens to be
	// empty, breaking every subsequent upload with ENOENT.
	if share.FileName != "" {
		filePath := filepath.Join(s.uploadsDir, share.FileName)
		os.Remove(filePath)
	}

	// Delete from database
	_, err := s.db.Exec("DELETE FROM shares WHERE id = ?", id)
	return err
}

// IncrementDownloads atomically increments the download counter.
// Returns (true, nil) if the increment succeeded, (false, nil) if the
// download limit was already reached, or (false, err) on database error.
func (s *SQLiteStorage) IncrementDownloads(id string) (bool, error) {
	result, err := s.db.Exec(
		`UPDATE shares SET downloads = downloads + 1
		 WHERE id = ? AND (max_downloads = 0 OR downloads < max_downloads)`,
		id,
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil // false means limit was reached
}

// ============= Query Operations =============

// GetExpiringSoon returns shares expiring within the given duration
func (s *SQLiteStorage) GetExpiringSoon(within time.Duration) ([]*models.Share, error) {
	// Format deadline the same way Save() formats expires_at so the string
	// comparison inside datetime() lines up across drivers. modernc.org/sqlite
	// binds time.Time as RFC3339Nano ("…T…Z"), which SQLite *can* parse but
	// then compares against the space-separated column value with different
	// sub-second precision, producing silent misses.
	deadline := time.Now().Add(within).UTC().Format("2006-01-02 15:04:05")

	query := `
		SELECT id, file_name, original_name, file_size, mime_type,
			   password_hash, has_password, expires_at, created_at,
			   downloads, max_downloads, source_path, is_symlink,
			   is_directory, COALESCE(parent_share_id, ''), total_files, total_size,
			   COALESCE(user_id, ''), COALESCE(user_email, '')
		FROM shares
		WHERE datetime(substr(expires_at, 1, 19)) > datetime('now')
		  AND datetime(substr(expires_at, 1, 19)) <= datetime(?)
		ORDER BY expires_at ASC
	`

	rows, err := s.db.Query(query, deadline)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*models.Share
	for rows.Next() {
		share := &models.Share{}
		if err := rows.Scan(
			&share.ID, &share.FileName, &share.OriginalName, &share.FileSize, &share.MimeType,
			&share.Password, &share.HasPassword, &share.ExpiresAt, &share.CreatedAt,
			&share.Downloads, &share.MaxDownloads, &share.SourcePath, &share.IsSymlink,
			&share.IsDirectory, &share.ParentShareID, &share.TotalFiles, &share.TotalSize,
			&share.UserID, &share.UserEmail,
		); err != nil {
			continue
		}
		shares = append(shares, share)
	}
	if err := rows.Err(); err != nil {
		return shares, err
	}

	return shares, nil
}

// GetByDateRange returns shares created within the date range
func (s *SQLiteStorage) GetByDateRange(from, to time.Time) ([]*models.Share, error) {
	query := `
		SELECT id, file_name, original_name, file_size, mime_type,
			   password_hash, has_password, expires_at, created_at,
			   downloads, max_downloads, source_path, is_symlink,
			   is_directory, COALESCE(parent_share_id, ''), total_files, total_size,
			   COALESCE(user_id, ''), COALESCE(user_email, '')
		FROM shares
		WHERE created_at >= ? AND created_at <= ?
		ORDER BY created_at DESC
	`

	// Bind the same textual format the column is stored in, so the string
	// comparison is valid (a time.Time bound here could serialize as RFC3339
	// and mis-compare against the space-separated stored values).
	rows, err := s.db.Query(query, fmtTime(from), fmtTime(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*models.Share
	for rows.Next() {
		share := &models.Share{}
		if err := rows.Scan(
			&share.ID, &share.FileName, &share.OriginalName, &share.FileSize, &share.MimeType,
			&share.Password, &share.HasPassword, &share.ExpiresAt, &share.CreatedAt,
			&share.Downloads, &share.MaxDownloads, &share.SourcePath, &share.IsSymlink,
			&share.IsDirectory, &share.ParentShareID, &share.TotalFiles, &share.TotalSize,
			&share.UserID, &share.UserEmail,
		); err != nil {
			continue
		}
		shares = append(shares, share)
	}
	if err := rows.Err(); err != nil {
		return shares, err
	}

	return shares, nil
}

// Search searches for shares by filename
func (s *SQLiteStorage) Search(query string) ([]*models.Share, error) {
	searchQuery := `
		SELECT id, file_name, original_name, file_size, mime_type,
			   password_hash, has_password, expires_at, created_at,
			   downloads, max_downloads, source_path, is_symlink,
			   is_directory, COALESCE(parent_share_id, ''), total_files, total_size,
			   COALESCE(user_id, ''), COALESCE(user_email, '')
		FROM shares
		WHERE datetime(substr(expires_at, 1, 19)) > datetime('now')
		  AND (original_name LIKE ? ESCAPE '\' OR file_name LIKE ? ESCAPE '\')
		ORDER BY created_at DESC
	`

	// Escape LIKE metacharacters in the user's query so a literal % or _ doesn't
	// turn into a wildcard (e.g. "_" matching everything). Backslash is the
	// escape char declared via ESCAPE above.
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(query)
	searchPattern := "%" + escaped + "%"
	rows, err := s.db.Query(searchQuery, searchPattern, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*models.Share
	for rows.Next() {
		share := &models.Share{}
		if err := rows.Scan(
			&share.ID, &share.FileName, &share.OriginalName, &share.FileSize, &share.MimeType,
			&share.Password, &share.HasPassword, &share.ExpiresAt, &share.CreatedAt,
			&share.Downloads, &share.MaxDownloads, &share.SourcePath, &share.IsSymlink,
			&share.IsDirectory, &share.ParentShareID, &share.TotalFiles, &share.TotalSize,
			&share.UserID, &share.UserEmail,
		); err != nil {
			continue
		}
		shares = append(shares, share)
	}
	if err := rows.Err(); err != nil {
		return shares, err
	}

	return shares, nil
}

// GetStats returns storage statistics
func (s *SQLiteStorage) GetStats() (*StorageStats, error) {
	stats := &StorageStats{}

	// Count active shares
	err := s.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(downloads), 0), COALESCE(SUM(file_size), 0)
		FROM shares WHERE datetime(substr(expires_at, 1, 19)) > datetime('now')
	`).Scan(&stats.TotalShares, &stats.TotalDownloads, &stats.TotalSize)
	if err != nil {
		return nil, err
	}

	// Count expiring soon (within 24h)
	deadline := time.Now().Add(24 * time.Hour).UTC().Format("2006-01-02 15:04:05")
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM shares
		WHERE datetime(substr(expires_at, 1, 19)) > datetime('now')
		  AND datetime(substr(expires_at, 1, 19)) <= datetime(?)
	`, deadline).Scan(&stats.ExpiringSoon)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// ============= Folder Operations =============

// SaveFolderContents saves folder contents for a share
func (s *SQLiteStorage) SaveFolderContents(shareID string, contents []*models.FolderEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO folder_contents (id, share_id, relative_path, file_name, file_size, mime_type, is_directory)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, entry := range contents {
		_, err := stmt.Exec(entry.ID, shareID, entry.RelativePath, entry.FileName, entry.FileSize, entry.MimeType, entry.IsDirectory)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetFolderContents returns folder contents for a share at a specific path
func (s *SQLiteStorage) GetFolderContents(shareID string, path string) ([]*models.FolderEntry, error) {
	// Normalize path
	if path == "" {
		path = "/"
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	query := `
		SELECT id, share_id, relative_path, file_name, file_size, COALESCE(mime_type, ''), is_directory
		FROM folder_contents
		WHERE share_id = ? AND relative_path LIKE ? AND relative_path NOT LIKE ?
		ORDER BY is_directory DESC, file_name ASC
	`

	// Match direct children only
	directChildren := path + "%"
	nestedChildren := path + "%/%"

	rows, err := s.db.Query(query, shareID, directChildren, nestedChildren)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.FolderEntry
	for rows.Next() {
		entry := &models.FolderEntry{}
		if err := rows.Scan(&entry.ID, &entry.ShareID, &entry.RelativePath, &entry.FileName, &entry.FileSize, &entry.MimeType, &entry.IsDirectory); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return entries, err
	}

	return entries, nil
}

// DeleteFolderContents removes all folder contents for a share
func (s *SQLiteStorage) DeleteFolderContents(shareID string) error {
	_, err := s.db.Exec("DELETE FROM folder_contents WHERE share_id = ?", shareID)
	return err
}

// ============= Receive Link Operations =============

// SaveReceiveLink saves a receive link
func (s *SQLiteStorage) SaveReceiveLink(link *models.ReceiveLink) error {
	query := `
		INSERT OR REPLACE INTO receive_links (
			id, name, password_hash, has_password, expires_at, created_at,
			max_uploads, max_file_size, allowed_extensions, auto_share,
			webhook_url, current_uploads, total_size, user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	expiresAt := fmtTimePtr(link.ExpiresAt)

	_, err := s.db.Exec(query,
		link.ID, link.Name, link.Password, link.HasPassword, expiresAt, fmtTime(link.CreatedAt),
		link.MaxUploads, link.MaxFileSize, link.AllowedExtensions, link.AutoShare,
		link.WebhookURL, link.CurrentUploads, link.TotalSize, link.UserID,
	)
	return err
}

// GetReceiveLink retrieves a receive link by ID
func (s *SQLiteStorage) GetReceiveLink(id string) (*models.ReceiveLink, bool) {
	query := `
		SELECT id, name, password_hash, has_password, expires_at, created_at,
			   max_uploads, max_file_size, allowed_extensions, auto_share,
			   webhook_url, current_uploads, total_size, COALESCE(user_id, '')
		FROM receive_links
		WHERE id = ? AND (expires_at IS NULL OR datetime(substr(expires_at, 1, 19)) > datetime('now'))
	`

	link := &models.ReceiveLink{}
	var expiresAt sql.NullTime

	err := s.db.QueryRow(query, id).Scan(
		&link.ID, &link.Name, &link.Password, &link.HasPassword, &expiresAt, &link.CreatedAt,
		&link.MaxUploads, &link.MaxFileSize, &link.AllowedExtensions, &link.AutoShare,
		&link.WebhookURL, &link.CurrentUploads, &link.TotalSize, &link.UserID,
	)

	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("Error getting receive link %s: %v", id, err)
		return nil, false
	}

	if expiresAt.Valid {
		link.ExpiresAt = &expiresAt.Time
	}

	return link, true
}

// GetAllReceiveLinks returns all non-expired receive links
func (s *SQLiteStorage) GetAllReceiveLinks() ([]*models.ReceiveLink, error) {
	query := `
		SELECT id, name, password_hash, has_password, expires_at, created_at,
			   max_uploads, max_file_size, allowed_extensions, auto_share,
			   webhook_url, current_uploads, total_size, COALESCE(user_id, '')
		FROM receive_links
		WHERE expires_at IS NULL OR datetime(substr(expires_at, 1, 19)) > datetime('now')
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*models.ReceiveLink
	for rows.Next() {
		link := &models.ReceiveLink{}
		var expiresAt sql.NullTime

		if err := rows.Scan(
			&link.ID, &link.Name, &link.Password, &link.HasPassword, &expiresAt, &link.CreatedAt,
			&link.MaxUploads, &link.MaxFileSize, &link.AllowedExtensions, &link.AutoShare,
			&link.WebhookURL, &link.CurrentUploads, &link.TotalSize, &link.UserID,
		); err != nil {
			continue
		}

		if expiresAt.Valid {
			link.ExpiresAt = &expiresAt.Time
		}

		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return links, err
	}

	return links, nil
}

// DeleteReceiveLink removes a receive link
func (s *SQLiteStorage) DeleteReceiveLink(id string) error {
	// Delete received files first (cascade will handle it, but also delete actual files)
	files, _ := s.GetReceivedFiles(id)
	for _, file := range files {
		filePath := filepath.Join(s.uploadsDir, "received", id, file.FileName)
		os.Remove(filePath)
	}

	// Remove the received directory
	os.RemoveAll(filepath.Join(s.uploadsDir, "received", id))

	_, err := s.db.Exec("DELETE FROM receive_links WHERE id = ?", id)
	return err
}

// IncrementReceiveLinkUploads atomically increments the upload counter.
// Returns (true, nil) if the increment succeeded, (false, nil) if the
// upload limit was already reached, or (false, err) on database error.
func (s *SQLiteStorage) IncrementReceiveLinkUploads(id string) (bool, error) {
	result, err := s.db.Exec(
		`UPDATE receive_links SET current_uploads = current_uploads + 1
		 WHERE id = ? AND (max_uploads = 0 OR current_uploads < max_uploads)`,
		id,
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil // false means limit was reached
}

// ============= Received Files Operations =============

// SaveReceivedFile saves a received file record
func (s *SQLiteStorage) SaveReceivedFile(file *models.ReceivedFile) error {
	// Insert the file row and bump the link's running total_size in ONE
	// transaction. total_size feeds GetUserUsage (quota accounting), so the two
	// must not drift: a crash/error between two separate Execs would leave the
	// file counted in received_files but not in total_size. DeleteReceivedFile
	// already uses a transaction for the reverse operation.
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		INSERT INTO received_files (
			id, receive_link_id, file_name, original_name, file_size,
			mime_type, uploader_ip, uploader_agent, created_at, share_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		file.ID, file.ReceiveLinkID, file.FileName, file.OriginalName, file.FileSize,
		file.MimeType, file.UploaderIP, file.UploaderAgent, fmtTime(file.CreatedAt), file.ShareID,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(
		"UPDATE receive_links SET total_size = total_size + ? WHERE id = ?",
		file.FileSize, file.ReceiveLinkID,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// GetReceivedFiles returns all files for a receive link
func (s *SQLiteStorage) GetReceivedFiles(linkID string) ([]*models.ReceivedFile, error) {
	query := `
		SELECT id, receive_link_id, file_name, original_name, file_size,
			   COALESCE(mime_type, ''), COALESCE(uploader_ip, ''), COALESCE(uploader_agent, ''),
			   created_at, COALESCE(share_id, '')
		FROM received_files
		WHERE receive_link_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, linkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*models.ReceivedFile
	for rows.Next() {
		file := &models.ReceivedFile{}
		if err := rows.Scan(
			&file.ID, &file.ReceiveLinkID, &file.FileName, &file.OriginalName, &file.FileSize,
			&file.MimeType, &file.UploaderIP, &file.UploaderAgent, &file.CreatedAt, &file.ShareID,
		); err != nil {
			continue
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return files, err
	}

	return files, nil
}

// DeleteReceivedFile removes a received file using a transaction to keep
// the received_files table and the receive_links.total_size in sync.
func (s *SQLiteStorage) DeleteReceivedFile(linkID, fileID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get file info first
	var fileName string
	var fileSize int64
	err = tx.QueryRow(
		"SELECT file_name, file_size FROM received_files WHERE id = ? AND receive_link_id = ?",
		fileID, linkID,
	).Scan(&fileName, &fileSize)
	if err != nil {
		return err
	}

	// Delete the DB record
	_, err = tx.Exec("DELETE FROM received_files WHERE id = ? AND receive_link_id = ?", fileID, linkID)
	if err != nil {
		return err
	}

	// Update total size
	_, err = tx.Exec("UPDATE receive_links SET total_size = MAX(0, total_size - ?) WHERE id = ?", fileSize, linkID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Delete the actual file after successful commit
	filePath := filepath.Join(s.uploadsDir, "received", linkID, fileName)
	os.Remove(filePath)

	return nil
}

// ============= User Operations =============

// nullIfEmpty maps an empty string to a SQL NULL.
//
// The users table carries UNIQUE(oidc_subject, oidc_issuer). SQLite treats two
// NULLs as distinct but two empty strings as equal, so storing "" for the
// non-OIDC (local) users would let only the *first* local account exist — every
// later one collides on (”, ”). Local accounts must therefore write NULL.
// All read paths already COALESCE these columns back to "".
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// CreateUser creates a new user
func (s *SQLiteStorage) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (
			id, email, name, role, password_hash,
			oidc_subject, oidc_issuer, is_active, created_at, last_login_at, quota_bytes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	lastLoginAt := fmtTimePtr(user.LastLoginAt)

	_, err := s.db.Exec(query,
		user.ID, user.Email, user.Name, user.Role, user.PasswordHash,
		nullIfEmpty(user.OIDCSubject), nullIfEmpty(user.OIDCIssuer),
		user.IsActive, fmtTime(user.CreatedAt), lastLoginAt, user.QuotaBytes,
	)
	return err
}

// GetUser retrieves a user by ID
func (s *SQLiteStorage) GetUser(id string) (*models.User, error) {
	query := `
		SELECT id, email, name, role, COALESCE(password_hash, ''),
			   COALESCE(oidc_subject, ''), COALESCE(oidc_issuer, ''),
			   is_active, created_at, last_login_at, COALESCE(quota_bytes, 0)
		FROM users
		WHERE id = ?
	`

	user := &models.User{}
	var lastLoginAt sql.NullTime

	err := s.db.QueryRow(query, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.Role, &user.PasswordHash,
		&user.OIDCSubject, &user.OIDCIssuer, &user.IsActive, &user.CreatedAt, &lastLoginAt, &user.QuotaBytes,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *SQLiteStorage) GetUserByEmail(email string) (*models.User, error) {
	// quota_bytes MUST be selected here: callers (the OIDC login path) load a
	// user through this query and hand the very same struct back to UpdateUser,
	// which persists every column. Omitting it would write back a zeroed quota
	// — i.e. silently promote the account to "unlimited" on each login.
	query := `
		SELECT id, email, name, role, COALESCE(password_hash, ''),
			   COALESCE(oidc_subject, ''), COALESCE(oidc_issuer, ''),
			   is_active, created_at, last_login_at, COALESCE(quota_bytes, 0)
		FROM users
		WHERE email = ?
	`

	user := &models.User{}
	var lastLoginAt sql.NullTime

	err := s.db.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.Name, &user.Role, &user.PasswordHash,
		&user.OIDCSubject, &user.OIDCIssuer, &user.IsActive, &user.CreatedAt, &lastLoginAt,
		&user.QuotaBytes,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// GetUserByOIDC retrieves a user by OIDC subject and issuer
func (s *SQLiteStorage) GetUserByOIDC(subject, issuer string) (*models.User, error) {
	// quota_bytes MUST be selected — see the note in GetUserByEmail. This is the
	// query the SSO login path uses before writing the user back.
	query := `
		SELECT id, email, name, role, COALESCE(password_hash, ''),
			   COALESCE(oidc_subject, ''), COALESCE(oidc_issuer, ''),
			   is_active, created_at, last_login_at, COALESCE(quota_bytes, 0)
		FROM users
		WHERE oidc_subject = ? AND oidc_issuer = ?
	`

	user := &models.User{}
	var lastLoginAt sql.NullTime

	err := s.db.QueryRow(query, subject, issuer).Scan(
		&user.ID, &user.Email, &user.Name, &user.Role, &user.PasswordHash,
		&user.OIDCSubject, &user.OIDCIssuer, &user.IsActive, &user.CreatedAt, &lastLoginAt,
		&user.QuotaBytes,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// GetAllUsers returns all users
func (s *SQLiteStorage) GetAllUsers() ([]*models.User, error) {
	query := `
		SELECT id, email, name, role, COALESCE(password_hash, ''),
			   COALESCE(oidc_subject, ''), COALESCE(oidc_issuer, ''),
			   is_active, created_at, last_login_at, COALESCE(quota_bytes, 0)
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var lastLoginAt sql.NullTime

		if err := rows.Scan(
			&user.ID, &user.Email, &user.Name, &user.Role, &user.PasswordHash,
			&user.OIDCSubject, &user.OIDCIssuer, &user.IsActive, &user.CreatedAt, &lastLoginAt, &user.QuotaBytes,
		); err != nil {
			continue
		}

		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}

		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return users, err
	}

	return users, nil
}

// UpdateUser updates an existing user
func (s *SQLiteStorage) UpdateUser(user *models.User) error {
	query := `
		UPDATE users SET
			email = ?, name = ?, role = ?, password_hash = ?,
			oidc_subject = ?, oidc_issuer = ?, is_active = ?, last_login_at = ?, quota_bytes = ?
		WHERE id = ?
	`

	lastLoginAt := fmtTimePtr(user.LastLoginAt)

	_, err := s.db.Exec(query,
		user.Email, user.Name, user.Role, user.PasswordHash,
		nullIfEmpty(user.OIDCSubject), nullIfEmpty(user.OIDCIssuer),
		user.IsActive, lastLoginAt, user.QuotaBytes,
		user.ID,
	)
	return err
}

// DeleteUser removes a user
func (s *SQLiteStorage) DeleteUser(id string) error {
	// Note: This doesn't delete user's shares/receive links
	// They will remain but be "orphaned" (user_id will point to non-existent user)
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

// GetUserUsage returns the managed storage a user consumes: the sum of their
// uploaded/copied share files plus everything received via their receive links.
// Symlink shares (is_symlink=1) and in-place folder shares (is_directory=1)
// reference host files on disk without copying, so they consume no managed
// storage and are excluded — matching what actually lives under data/uploads.
func (s *SQLiteStorage) GetUserUsage(userID string) (int64, error) {
	var shareBytes, receivedBytes int64
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(file_size), 0) FROM shares
		 WHERE user_id = ? AND is_symlink = 0 AND is_directory = 0`, userID,
	).Scan(&shareBytes)
	if err != nil {
		return 0, err
	}
	err = s.db.QueryRow(
		`SELECT COALESCE(SUM(total_size), 0) FROM receive_links WHERE user_id = ?`, userID,
	).Scan(&receivedBytes)
	if err != nil {
		return 0, err
	}
	return shareBytes + receivedBytes, nil
}

// GetSharesByUser returns all non-expired shares for a user
func (s *SQLiteStorage) GetSharesByUser(userID string) []*models.Share {
	query := `
		SELECT id, file_name, original_name, file_size, mime_type,
			   password_hash, has_password, expires_at, created_at,
			   downloads, max_downloads, source_path, is_symlink,
			   is_directory, COALESCE(parent_share_id, ''), total_files, total_size,
			   COALESCE(user_id, ''), COALESCE(user_email, '')
		FROM shares
		WHERE user_id = ? AND datetime(substr(expires_at, 1, 19)) > datetime('now')
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		log.Printf("Error getting shares for user %s: %v", userID, err)
		return nil
	}
	defer rows.Close()

	var shares []*models.Share
	for rows.Next() {
		share := &models.Share{}
		if err := rows.Scan(
			&share.ID, &share.FileName, &share.OriginalName, &share.FileSize, &share.MimeType,
			&share.Password, &share.HasPassword, &share.ExpiresAt, &share.CreatedAt,
			&share.Downloads, &share.MaxDownloads, &share.SourcePath, &share.IsSymlink,
			&share.IsDirectory, &share.ParentShareID, &share.TotalFiles, &share.TotalSize,
			&share.UserID, &share.UserEmail,
		); err != nil {
			continue
		}
		shares = append(shares, share)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating shares for user %s: %v", userID, err)
	}

	return shares
}

// GetReceiveLinksByUser returns all non-expired receive links for a user
func (s *SQLiteStorage) GetReceiveLinksByUser(userID string) ([]*models.ReceiveLink, error) {
	query := `
		SELECT id, name, password_hash, has_password, expires_at, created_at,
			   max_uploads, max_file_size, allowed_extensions, auto_share,
			   webhook_url, current_uploads, total_size, COALESCE(user_id, '')
		FROM receive_links
		WHERE user_id = ? AND (expires_at IS NULL OR datetime(substr(expires_at, 1, 19)) > datetime('now'))
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*models.ReceiveLink
	for rows.Next() {
		link := &models.ReceiveLink{}
		var expiresAt sql.NullTime

		if err := rows.Scan(
			&link.ID, &link.Name, &link.Password, &link.HasPassword, &expiresAt, &link.CreatedAt,
			&link.MaxUploads, &link.MaxFileSize, &link.AllowedExtensions, &link.AutoShare,
			&link.WebhookURL, &link.CurrentUploads, &link.TotalSize, &link.UserID,
		); err != nil {
			continue
		}

		if expiresAt.Valid {
			link.ExpiresAt = &expiresAt.Time
		}

		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return links, err
	}

	return links, nil
}

// ============= Cleanup =============

func (s *SQLiteStorage) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCleanup:
			return
		case <-ticker.C:
			s.cleanupExpiredShares()
			s.cleanupExpiredReceiveLinks()
		}
	}
}

func (s *SQLiteStorage) cleanupExpiredShares() {
	// Get expired shares to delete files
	rows, err := s.db.Query(`
		SELECT id, file_name, is_directory FROM shares WHERE datetime(substr(expires_at, 1, 19)) <= datetime('now')
	`)
	if err != nil {
		log.Printf("Error querying expired shares: %v", err)
		return
	}
	defer rows.Close()

	var expiredIDs []string
	for rows.Next() {
		var id, fileName string
		var isDirectory bool
		if err := rows.Scan(&id, &fileName, &isDirectory); err != nil {
			continue
		}

		// Delete the file or directory
		filePath := filepath.Join(s.uploadsDir, fileName)
		if isDirectory {
			os.RemoveAll(filePath)
		} else {
			os.Remove(filePath)
		}
		expiredIDs = append(expiredIDs, id)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating expired shares: %v", err)
	}

	// Delete exactly the rows whose files we just removed. Re-running the
	// datetime('now') predicate here would delete any share that crossed the
	// expiry boundary between the SELECT and now — leaving its file orphaned on
	// disk because it was never in expiredIDs. Newly-expired rows are picked up
	// (file and all) on the next cleanup pass instead.
	if len(expiredIDs) > 0 {
		if _, err := s.db.Exec(
			"DELETE FROM shares WHERE id IN ("+sqlPlaceholders(len(expiredIDs))+")",
			toAnySlice(expiredIDs)...,
		); err != nil {
			log.Printf("Error deleting expired shares: %v", err)
		} else {
			log.Printf("Cleaned up %d expired shares", len(expiredIDs))
		}
	}
}

// sqlPlaceholders returns "?, ?, ..." with n placeholders for an IN clause.
func sqlPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// toAnySlice converts a []string to []any for variadic db.Exec args.
func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func (s *SQLiteStorage) cleanupExpiredReceiveLinks() {
	// Get expired receive links
	rows, err := s.db.Query(`
		SELECT id FROM receive_links WHERE expires_at IS NOT NULL AND datetime(substr(expires_at, 1, 19)) <= datetime('now')
	`)
	if err != nil {
		log.Printf("Error querying expired receive links: %v", err)
		return
	}
	defer rows.Close()

	var expiredIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		expiredIDs = append(expiredIDs, id)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating expired receive links: %v", err)
	}

	// Delete expired receive links (this will cascade delete received files)
	for _, id := range expiredIDs {
		os.RemoveAll(filepath.Join(s.uploadsDir, "received", id))
	}

	if len(expiredIDs) > 0 {
		// Delete exactly the collected IDs, not a re-evaluated now() predicate,
		// so a link that expires mid-cleanup isn't removed from the DB while its
		// received-files directory is left orphaned (see cleanupExpiredShares).
		if _, err := s.db.Exec(
			"DELETE FROM receive_links WHERE id IN ("+sqlPlaceholders(len(expiredIDs))+")",
			toAnySlice(expiredIDs)...,
		); err != nil {
			log.Printf("Error deleting expired receive links: %v", err)
		} else {
			log.Printf("Cleaned up %d expired receive links", len(expiredIDs))
		}
	}
}

// ============= SMTP Config Operations =============

// GetSMTPConfig retrieves the SMTP configuration
func (s *SQLiteStorage) GetSMTPConfig() (*models.SMTPConfig, error) {
	query := `
		SELECT enabled, COALESCE(host, ''), COALESCE(port, 587),
			   COALESCE(username, ''), COALESCE(password, ''),
			   COALESCE(from_email, ''), COALESCE(from_name, ''),
			   COALESCE(use_tls, 0), COALESCE(use_starttls, 1)
		FROM smtp_config WHERE id = 1
	`

	config := &models.SMTPConfig{}
	err := s.db.QueryRow(query).Scan(
		&config.Enabled, &config.Host, &config.Port,
		&config.Username, &config.Password,
		&config.FromEmail, &config.FromName,
		&config.UseTLS, &config.UseStartTLS,
	)

	if err == sql.ErrNoRows {
		// Return default config
		return &models.SMTPConfig{
			Enabled:     false,
			Port:        587,
			UseStartTLS: true,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return config, nil
}

// SaveSMTPConfig saves the SMTP configuration
func (s *SQLiteStorage) SaveSMTPConfig(config *models.SMTPConfig) error {
	query := `
		INSERT INTO smtp_config (id, enabled, host, port, username, password, from_email, from_name, use_tls, use_starttls, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			host = excluded.host,
			port = excluded.port,
			username = excluded.username,
			password = excluded.password,
			from_email = excluded.from_email,
			from_name = excluded.from_name,
			use_tls = excluded.use_tls,
			use_starttls = excluded.use_starttls,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := s.db.Exec(query,
		config.Enabled, config.Host, config.Port,
		config.Username, config.Password,
		config.FromEmail, config.FromName,
		config.UseTLS, config.UseStartTLS,
	)
	return err
}

// ============= Email Transfer Operations =============

// SaveEmailTransfer saves an email transfer record
func (s *SQLiteStorage) SaveEmailTransfer(transfer *models.EmailTransferRecord) error {
	query := `
		INSERT INTO email_transfers (
			id, share_id, recipient_email, recipient_name,
			sender_email, sender_name, title, message,
			notify_download, sent_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		transfer.ID, transfer.ShareID, transfer.RecipientEmail, transfer.RecipientName,
		transfer.SenderEmail, transfer.SenderName, transfer.Title, transfer.Message,
		transfer.NotifyDownload, transfer.SentAt,
	)
	return err
}

// GetEmailTransfersByShare retrieves email transfers for a share
func (s *SQLiteStorage) GetEmailTransfersByShare(shareID string) ([]*models.EmailTransferRecord, error) {
	query := `
		SELECT id, share_id, recipient_email, COALESCE(recipient_name, ''),
			   sender_email, COALESCE(sender_name, ''), COALESCE(title, ''),
			   COALESCE(message, ''), notify_download, sent_at,
			   COALESCE(downloaded_at, ''), COALESCE(notified_at, '')
		FROM email_transfers
		WHERE share_id = ?
		ORDER BY sent_at DESC
	`

	rows, err := s.db.Query(query, shareID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []*models.EmailTransferRecord
	for rows.Next() {
		t := &models.EmailTransferRecord{}
		if err := rows.Scan(
			&t.ID, &t.ShareID, &t.RecipientEmail, &t.RecipientName,
			&t.SenderEmail, &t.SenderName, &t.Title, &t.Message,
			&t.NotifyDownload, &t.SentAt, &t.DownloadedAt, &t.NotifiedAt,
		); err != nil {
			continue
		}
		transfers = append(transfers, t)
	}
	if err := rows.Err(); err != nil {
		return transfers, err
	}

	return transfers, nil
}

// MarkEmailTransferDownloaded marks an email transfer as downloaded
func (s *SQLiteStorage) MarkEmailTransferDownloaded(shareID string) error {
	query := `
		UPDATE email_transfers
		SET downloaded_at = CURRENT_TIMESTAMP
		WHERE share_id = ? AND downloaded_at IS NULL
	`
	_, err := s.db.Exec(query, shareID)
	return err
}

// MarkEmailTransferNotified marks an email transfer notification as sent
func (s *SQLiteStorage) MarkEmailTransferNotified(shareID string) error {
	query := `
		UPDATE email_transfers
		SET notified_at = CURRENT_TIMESTAMP
		WHERE share_id = ? AND notified_at IS NULL
	`
	_, err := s.db.Exec(query, shareID)
	return err
}

// GetPendingDownloadNotifications gets transfers that need download notification
func (s *SQLiteStorage) GetPendingDownloadNotifications(shareID string) ([]*models.EmailTransferRecord, error) {
	query := `
		SELECT id, share_id, recipient_email, COALESCE(recipient_name, ''),
			   sender_email, COALESCE(sender_name, ''), COALESCE(title, ''),
			   COALESCE(message, ''), notify_download, sent_at,
			   COALESCE(downloaded_at, ''), COALESCE(notified_at, '')
		FROM email_transfers
		WHERE share_id = ?
		  AND notify_download = 1
		  AND downloaded_at IS NOT NULL
		  AND notified_at IS NULL
	`

	rows, err := s.db.Query(query, shareID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []*models.EmailTransferRecord
	for rows.Next() {
		t := &models.EmailTransferRecord{}
		if err := rows.Scan(
			&t.ID, &t.ShareID, &t.RecipientEmail, &t.RecipientName,
			&t.SenderEmail, &t.SenderName, &t.Title, &t.Message,
			&t.NotifyDownload, &t.SentAt, &t.DownloadedAt, &t.NotifiedAt,
		); err != nil {
			continue
		}
		transfers = append(transfers, t)
	}
	if err := rows.Err(); err != nil {
		return transfers, err
	}

	return transfers, nil
}

// ============= API Key Operations =============

// CreateAPIKey creates a new API key record
func (s *SQLiteStorage) CreateAPIKey(id, name, keyHash, prefix, userID, role string) error {
	_, err := s.db.Exec(
		"INSERT INTO api_keys (id, name, key_hash, prefix, user_id, role, created_at, is_active) VALUES (?, ?, ?, ?, ?, ?, ?, 1)",
		id, name, keyHash, prefix, userID, role, time.Now().UTC(),
	)
	return err
}

// GetAPIKeyByHash looks up an API key by its SHA-256 hash
func (s *SQLiteStorage) GetAPIKeyByHash(keyHash string) (id, name, userID, role string, isActive bool, err error) {
	err = s.db.QueryRow(
		"SELECT id, name, COALESCE(user_id, ''), role, is_active FROM api_keys WHERE key_hash = ?",
		keyHash,
	).Scan(&id, &name, &userID, &role, &isActive)
	if err == sql.ErrNoRows {
		return "", "", "", "", false, nil
	}
	return
}

// ListAPIKeys returns all API keys (without hashes)
func (s *SQLiteStorage) ListAPIKeys() ([]map[string]interface{}, error) {
	rows, err := s.db.Query("SELECT id, name, prefix, COALESCE(user_id, ''), role, created_at, last_used_at, is_active FROM api_keys ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []map[string]interface{}
	for rows.Next() {
		var id, name, prefix, userID, role string
		var createdAt time.Time
		var lastUsedAt sql.NullTime
		var isActive bool
		if err := rows.Scan(&id, &name, &prefix, &userID, &role, &createdAt, &lastUsedAt, &isActive); err != nil {
			continue
		}
		key := map[string]interface{}{
			"id": id, "name": name, "prefix": prefix, "user_id": userID,
			"role": role, "created_at": createdAt, "is_active": isActive,
		}
		if lastUsedAt.Valid {
			key["last_used_at"] = lastUsedAt.Time
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return keys, err
	}
	return keys, nil
}

// DeleteAPIKey removes an API key
func (s *SQLiteStorage) DeleteAPIKey(id string) error {
	_, err := s.db.Exec("DELETE FROM api_keys WHERE id = ?", id)
	return err
}

// UpdateAPIKeyLastUsed updates the last_used_at timestamp for an API key
func (s *SQLiteStorage) UpdateAPIKeyLastUsed(id string) {
	s.db.Exec("UPDATE api_keys SET last_used_at = ? WHERE id = ?", time.Now().UTC(), id)
}

// fixDateTimeFormats fixes timezone format in existing dates
func (s *SQLiteStorage) fixDateTimeFormats() {
	// Update shares table - remove timezone suffix from expires_at and created_at
	_, err := s.db.Exec(`
		UPDATE shares 
		SET expires_at = substr(expires_at, 1, 19),
		    created_at = substr(created_at, 1, 19)
		WHERE expires_at LIKE "%+00:00" OR created_at LIKE "%+00:00"
	`)
	if err != nil {
		log.Printf("Warning: Failed to fix datetime formats in shares: %v", err)
	}

	// Update receive_links table
	_, err = s.db.Exec(`
		UPDATE receive_links 
		SET expires_at = substr(expires_at, 1, 19),
		    created_at = substr(created_at, 1, 19)
		WHERE expires_at LIKE "%+00:00" OR created_at LIKE "%+00:00"
	`)
	if err != nil {
		log.Printf("Warning: Failed to fix datetime formats in receive_links: %v", err)
	}
}

// DropSharesTableForTest — see Storage.DropSharesTableForTest.
func (s *SQLiteStorage) DropSharesTableForTest() error {
	_, err := s.db.Exec(`DROP TABLE shares`)
	return err
}
