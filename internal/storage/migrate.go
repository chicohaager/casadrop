package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"casadrop/internal/models"
)

// MigrateJSONToSQLite migrates data from JSON file to SQLite database
func MigrateJSONToSQLite(dataDir string) error {
	jsonPath := filepath.Join(dataDir, "shares.json")
	dbPath := filepath.Join(dataDir, "shares.db")
	backupPath := filepath.Join(dataDir, "shares.json.backup")

	// Check if JSON file exists
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		log.Println("No shares.json found, skipping migration")
		return nil
	}

	// Check if database already exists
	if _, err := os.Stat(dbPath); err == nil {
		log.Println("shares.db already exists, skipping migration")
		return nil
	}

	log.Println("Starting migration from JSON to SQLite...")

	// Read JSON file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}

	var shares map[string]*models.Share
	if err := json.Unmarshal(data, &shares); err != nil {
		return err
	}

	log.Printf("Found %d shares to migrate", len(shares))

	// Create SQLite storage (this creates the database and schema)
	sqliteStorage, err := NewSQLiteStorage(dataDir)
	if err != nil {
		removeDatabaseFiles(dbPath)
		return err
	}

	// All or nothing (2.5.2): a share that fails used to be skipped while the
	// JSON was still renamed to .backup, so it vanished from CasaDrop. Now any
	// failure removes the database this call created and keeps shares.json,
	// so the next start retries the whole migration.
	for id, share := range shares {
		if share == nil {
			err = fmt.Errorf("share %s: empty entry in shares.json", id)
		} else {
			err = sqliteStorage.Save(share)
		}
		if err != nil {
			sqliteStorage.Close()
			removeDatabaseFiles(dbPath)
			return fmt.Errorf("migrate share %s: %w", id, err)
		}
	}
	sqliteStorage.Close()

	log.Printf("Successfully migrated %d shares", len(shares))

	// Rename JSON file to backup
	if err := os.Rename(jsonPath, backupPath); err != nil {
		log.Printf("Warning: Failed to rename JSON file to backup: %v", err)
		// Continue anyway - migration was successful
	} else {
		log.Printf("JSON file backed up to %s", backupPath)
	}

	log.Println("Migration completed successfully!")
	return nil
}

// CheckMigrationNeeded checks if migration from JSON to SQLite is needed
func CheckMigrationNeeded(dataDir string) bool {
	jsonPath := filepath.Join(dataDir, "shares.json")
	dbPath := filepath.Join(dataDir, "shares.db")

	// Migration needed if JSON exists and DB doesn't
	jsonExists := false
	dbExists := false

	if _, err := os.Stat(jsonPath); err == nil {
		jsonExists = true
	}

	if _, err := os.Stat(dbPath); err == nil {
		dbExists = true
	}

	return jsonExists && !dbExists
}

// removeDatabaseFiles deletes a database created by a failed migration,
// including the WAL sidecars, so CheckMigrationNeeded sees no database and the
// migration runs again on the next start. Only called for a database that did
// not exist before this migration started.
func removeDatabaseFiles(dbPath string) {
	for _, p := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: could not remove %s after failed migration: %v", p, err)
		}
	}
}
