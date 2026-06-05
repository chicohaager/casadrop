package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"casadrop/internal/models"
)

// MigrateUsers performs migration for multi-user support
// This adds user_id columns to existing tables and creates admin user
func (s *SQLiteStorage) MigrateUsers(adminPasswordHash string, adminEmail string) error {
	// Check if users table exists
	var tableName string
	err := s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	if err == sql.ErrNoRows {
		log.Println("Users table does not exist, running migration...")
		return s.runUserMigration(adminPasswordHash, adminEmail)
	}
	if err != nil {
		return err
	}

	// Table exists, check if we need to add columns to shares/receive_links
	return s.migrateExistingTables()
}

// runUserMigration creates users table and migrates existing data
func (s *SQLiteStorage) runUserMigration(adminPasswordHash string, adminEmail string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create users table
	_, err = tx.Exec(`
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
			UNIQUE(oidc_subject, oidc_issuer)
		)
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = tx.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)")
	if err != nil {
		return err
	}
	_, err = tx.Exec("CREATE INDEX IF NOT EXISTS idx_users_oidc ON users(oidc_subject, oidc_issuer)")
	if err != nil {
		return err
	}

	// Create admin user if password hash provided
	var adminUserID string
	if adminPasswordHash != "" {
		adminUserID = uuid.New().String()
		if adminEmail == "" {
			adminEmail = "admin@localhost"
		}
		now := time.Now().UTC()

		_, err = tx.Exec(`
			INSERT INTO users (id, email, name, role, password_hash, is_active, created_at)
			VALUES (?, ?, ?, ?, ?, 1, ?)
		`, adminUserID, adminEmail, "Admin", models.RoleAdmin, adminPasswordHash, now)
		if err != nil {
			return err
		}
		log.Printf("Created admin user: %s (%s)", adminEmail, adminUserID)
	}

	// Add user_id column to shares if not exists
	if err := addColumnIfNotExists(tx, "shares", "user_id", "TEXT"); err != nil {
		return err
	}
	if err := addColumnIfNotExists(tx, "shares", "user_email", "TEXT"); err != nil {
		return err
	}

	// Add user_id column to receive_links if not exists
	if err := addColumnIfNotExists(tx, "receive_links", "user_id", "TEXT"); err != nil {
		return err
	}

	// Assign all existing shares to admin user
	if adminUserID != "" {
		_, err = tx.Exec(`
			UPDATE shares SET user_id = ?, user_email = ?
			WHERE user_id IS NULL OR user_id = ''
		`, adminUserID, adminEmail)
		if err != nil {
			return err
		}

		// Assign all existing receive links to admin user
		_, err = tx.Exec(`
			UPDATE receive_links SET user_id = ?
			WHERE user_id IS NULL OR user_id = ''
		`, adminUserID)
		if err != nil {
			return err
		}

		log.Println("Assigned existing shares and receive links to admin user")
	}

	// Create indexes for user_id columns
	_, _ = tx.Exec("CREATE INDEX IF NOT EXISTS idx_shares_user ON shares(user_id)")
	_, _ = tx.Exec("CREATE INDEX IF NOT EXISTS idx_receive_links_user ON receive_links(user_id)")

	return tx.Commit()
}

// migrateExistingTables adds user_id columns if they don't exist
func (s *SQLiteStorage) migrateExistingTables() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Add columns if not exists
	if err := addColumnIfNotExists(tx, "shares", "user_id", "TEXT"); err != nil {
		return err
	}
	if err := addColumnIfNotExists(tx, "shares", "user_email", "TEXT"); err != nil {
		return err
	}
	if err := addColumnIfNotExists(tx, "receive_links", "user_id", "TEXT"); err != nil {
		return err
	}

	// Create indexes
	_, _ = tx.Exec("CREATE INDEX IF NOT EXISTS idx_shares_user ON shares(user_id)")
	_, _ = tx.Exec("CREATE INDEX IF NOT EXISTS idx_receive_links_user ON receive_links(user_id)")

	return tx.Commit()
}

// allowedMigrationTables restricts which tables can be passed to
// addColumnIfNotExists. PRAGMA table_info does not support parameter binding,
// so we whitelist table identifiers to avoid any chance of SQL injection via
// an attacker-controlled (or accidentally-controlled) table name.
var allowedMigrationTables = map[string]struct{}{
	"shares":        {},
	"receive_links": {},
	"users":         {},
	"oidc_config":   {},
	"api_keys":      {},
}

// addColumnIfNotExists adds a column to a table if it doesn't already exist
func addColumnIfNotExists(tx *sql.Tx, table, column, dataType string) error {
	if _, ok := allowedMigrationTables[table]; !ok {
		return fmt.Errorf("addColumnIfNotExists: table %q is not in the migration whitelist", table)
	}
	// Check if column exists. PRAGMA doesn't support ? placeholders, but the
	// table name is whitelisted above.
	rows, err := tx.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()

	columnExists := false
	for rows.Next() {
		var cid int
		var name, typeName string
		var notNull, pk int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &typeName, &notNull, &defaultValue, &pk); err != nil {
			continue
		}
		if name == column {
			columnExists = true
			break
		}
	}

	if !columnExists {
		_, err = tx.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + dataType)
		if err != nil {
			return err
		}
		log.Printf("Added column %s to table %s", column, table)
	}

	return nil
}

// GetAdminUser returns the first admin user (for migration purposes)
func (s *SQLiteStorage) GetAdminUser() (*models.User, error) {
	query := `
		SELECT id, email, name, role, COALESCE(password_hash, ''),
			   COALESCE(oidc_subject, ''), COALESCE(oidc_issuer, ''),
			   is_active, created_at, last_login_at
		FROM users
		WHERE role = ? AND is_active = 1
		LIMIT 1
	`

	user := &models.User{}
	var lastLoginAt sql.NullTime

	err := s.db.QueryRow(query, models.RoleAdmin).Scan(
		&user.ID, &user.Email, &user.Name, &user.Role, &user.PasswordHash,
		&user.OIDCSubject, &user.OIDCIssuer, &user.IsActive, &user.CreatedAt, &lastLoginAt,
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

// HasAnyUsers returns true if there are any users in the database
func (s *SQLiteStorage) HasAnyUsers() bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}
