package storage

import (
	"encoding/json"
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
		return err
	}
	defer sqliteStorage.Close()

	// Migrate each share
	migrated := 0
	for _, share := range shares {
		if err := sqliteStorage.Save(share); err != nil {
			log.Printf("Warning: Failed to migrate share %s: %v", share.ID, err)
			continue
		}
		migrated++
	}

	log.Printf("Successfully migrated %d/%d shares", migrated, len(shares))

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
