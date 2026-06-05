package storage

import (
	"log"
	"time"

	"casadrop/internal/models"
)

// Storage wraps the storage backend for compatibility.
// This is a facade that delegates to any StorageBackend implementation.
type Storage struct {
	backend StorageBackend
}

// New creates a new storage instance with automatic migration
func New(dataDir string) (*Storage, error) {
	// Check if migration is needed
	if CheckMigrationNeeded(dataDir) {
		log.Println("Migration from JSON to SQLite needed...")
		if err := MigrateJSONToSQLite(dataDir); err != nil {
			log.Printf("Warning: Migration failed: %v", err)
			// Continue anyway - will create fresh database
		}
	}

	// Create SQLite backend
	backend, err := NewSQLiteStorage(dataDir)
	if err != nil {
		return nil, err
	}

	return &Storage{backend: backend}, nil
}

// UploadsDir returns the uploads directory path
func (s *Storage) UploadsDir() string {
	return s.backend.UploadsDir()
}

// Ping verifies the storage backend is reachable (used by readiness checks)
func (s *Storage) Ping() error {
	return s.backend.Ping()
}

// Close closes the storage backend
func (s *Storage) Close() error {
	return s.backend.Close()
}

// ============= Share Operations =============

// Save saves or updates a share
func (s *Storage) Save(share *models.Share) error {
	return s.backend.Save(share)
}

// Get retrieves a share by ID
func (s *Storage) Get(id string) (*models.Share, bool) {
	return s.backend.Get(id)
}

// GetAll returns all non-expired shares
func (s *Storage) GetAll() []*models.Share {
	return s.backend.GetAll()
}

// Delete removes a share and its file
func (s *Storage) Delete(id string) error {
	return s.backend.Delete(id)
}

// IncrementDownloads atomically increments the download counter with limit check.
// Returns (true, nil) on success, (false, nil) if the limit was reached.
func (s *Storage) IncrementDownloads(id string) (bool, error) {
	return s.backend.IncrementDownloads(id)
}

// ============= Query Operations =============

// GetExpiringSoon returns shares expiring within the given duration
func (s *Storage) GetExpiringSoon(within time.Duration) ([]*models.Share, error) {
	return s.backend.GetExpiringSoon(within)
}

// GetByDateRange returns shares created within the date range
func (s *Storage) GetByDateRange(from, to time.Time) ([]*models.Share, error) {
	return s.backend.GetByDateRange(from, to)
}

// Search searches for shares by filename
func (s *Storage) Search(query string) ([]*models.Share, error) {
	return s.backend.Search(query)
}

// GetStats returns storage statistics
func (s *Storage) GetStats() (*StorageStats, error) {
	return s.backend.GetStats()
}

// ============= Folder Operations =============

// SaveFolderContents saves folder contents for a share
func (s *Storage) SaveFolderContents(shareID string, contents []*models.FolderEntry) error {
	return s.backend.SaveFolderContents(shareID, contents)
}

// GetFolderContents returns folder contents for a share at a specific path
func (s *Storage) GetFolderContents(shareID string, path string) ([]*models.FolderEntry, error) {
	return s.backend.GetFolderContents(shareID, path)
}

// DeleteFolderContents removes all folder contents for a share
func (s *Storage) DeleteFolderContents(shareID string) error {
	return s.backend.DeleteFolderContents(shareID)
}

// ============= Receive Link Operations =============

// SaveReceiveLink saves a receive link
func (s *Storage) SaveReceiveLink(link *models.ReceiveLink) error {
	return s.backend.SaveReceiveLink(link)
}

// GetReceiveLink retrieves a receive link by ID
func (s *Storage) GetReceiveLink(id string) (*models.ReceiveLink, bool) {
	return s.backend.GetReceiveLink(id)
}

// GetAllReceiveLinks returns all non-expired receive links
func (s *Storage) GetAllReceiveLinks() ([]*models.ReceiveLink, error) {
	return s.backend.GetAllReceiveLinks()
}

// DeleteReceiveLink removes a receive link
func (s *Storage) DeleteReceiveLink(id string) error {
	return s.backend.DeleteReceiveLink(id)
}

// IncrementReceiveLinkUploads atomically increments the upload counter with limit check.
// Returns (true, nil) on success, (false, nil) if the limit was reached.
func (s *Storage) IncrementReceiveLinkUploads(id string) (bool, error) {
	return s.backend.IncrementReceiveLinkUploads(id)
}

// ============= Received Files Operations =============

// SaveReceivedFile saves a received file record
func (s *Storage) SaveReceivedFile(file *models.ReceivedFile) error {
	return s.backend.SaveReceivedFile(file)
}

// GetReceivedFiles returns all files for a receive link
func (s *Storage) GetReceivedFiles(linkID string) ([]*models.ReceivedFile, error) {
	return s.backend.GetReceivedFiles(linkID)
}

// DeleteReceivedFile removes a received file within a transaction
func (s *Storage) DeleteReceivedFile(linkID, fileID string) error {
	return s.backend.DeleteReceivedFile(linkID, fileID)
}

// ============= User Operations =============

// CreateUser creates a new user
func (s *Storage) CreateUser(user *models.User) error {
	return s.backend.CreateUser(user)
}

// GetUser retrieves a user by ID
func (s *Storage) GetUser(id string) (*models.User, error) {
	return s.backend.GetUser(id)
}

// GetUserByEmail retrieves a user by email
func (s *Storage) GetUserByEmail(email string) (*models.User, error) {
	return s.backend.GetUserByEmail(email)
}

// GetUserByOIDC retrieves a user by OIDC subject and issuer
func (s *Storage) GetUserByOIDC(subject, issuer string) (*models.User, error) {
	return s.backend.GetUserByOIDC(subject, issuer)
}

// GetAllUsers returns all users
func (s *Storage) GetAllUsers() ([]*models.User, error) {
	return s.backend.GetAllUsers()
}

// UpdateUser updates an existing user
func (s *Storage) UpdateUser(user *models.User) error {
	return s.backend.UpdateUser(user)
}

// DeleteUser removes a user
func (s *Storage) DeleteUser(id string) error {
	return s.backend.DeleteUser(id)
}

// GetSharesByUser returns all non-expired shares for a user
func (s *Storage) GetSharesByUser(userID string) []*models.Share {
	return s.backend.GetSharesByUser(userID)
}

// GetReceiveLinksByUser returns all non-expired receive links for a user
func (s *Storage) GetReceiveLinksByUser(userID string) ([]*models.ReceiveLink, error) {
	return s.backend.GetReceiveLinksByUser(userID)
}

// ============= SMTP/Email Operations =============

// GetSMTPConfig returns the SMTP configuration
func (s *Storage) GetSMTPConfig() (*models.SMTPConfig, error) {
	return s.backend.GetSMTPConfig()
}

// SaveSMTPConfig saves the SMTP configuration
func (s *Storage) SaveSMTPConfig(config *models.SMTPConfig) error {
	return s.backend.SaveSMTPConfig(config)
}

// SaveEmailTransfer saves an email transfer record
func (s *Storage) SaveEmailTransfer(transfer *models.EmailTransferRecord) error {
	return s.backend.SaveEmailTransfer(transfer)
}

// GetEmailTransfersByShare returns all email transfers for a share
func (s *Storage) GetEmailTransfersByShare(shareID string) ([]*models.EmailTransferRecord, error) {
	return s.backend.GetEmailTransfersByShare(shareID)
}

// MarkEmailTransferDownloaded marks an email transfer as downloaded
func (s *Storage) MarkEmailTransferDownloaded(shareID string) error {
	return s.backend.MarkEmailTransferDownloaded(shareID)
}

// MarkEmailTransferNotified marks an email transfer as notified
func (s *Storage) MarkEmailTransferNotified(shareID string) error {
	return s.backend.MarkEmailTransferNotified(shareID)
}

// GetPendingDownloadNotifications returns email transfers that need download notifications
func (s *Storage) GetPendingDownloadNotifications(shareID string) ([]*models.EmailTransferRecord, error) {
	return s.backend.GetPendingDownloadNotifications(shareID)
}

// ============= API Key Operations =============

// CreateAPIKey creates a new API key record
func (s *Storage) CreateAPIKey(id, name, keyHash, prefix, userID, role string) error {
	return s.backend.CreateAPIKey(id, name, keyHash, prefix, userID, role)
}

// GetAPIKeyByHash looks up an API key by its SHA-256 hash
func (s *Storage) GetAPIKeyByHash(keyHash string) (id, name, userID, role string, isActive bool, err error) {
	return s.backend.GetAPIKeyByHash(keyHash)
}

// ListAPIKeys returns all API keys (without hashes)
func (s *Storage) ListAPIKeys() ([]map[string]interface{}, error) {
	return s.backend.ListAPIKeys()
}

// DeleteAPIKey removes an API key
func (s *Storage) DeleteAPIKey(id string) error {
	return s.backend.DeleteAPIKey(id)
}

// UpdateAPIKeyLastUsed updates the last_used_at timestamp for an API key
func (s *Storage) UpdateAPIKeyLastUsed(id string) {
	s.backend.UpdateAPIKeyLastUsed(id)
}
