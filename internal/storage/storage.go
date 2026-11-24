package storage

import (
	"context"
	"fmt"
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
		// Fatal since 2.5.2: continuing created a fresh, empty shares.db, and
		// because a database now existed the JSON data was never migrated on
		// any later start — the shares looked lost.
		if err := MigrateJSONToSQLite(dataDir); err != nil {
			return nil, fmt.Errorf("migration from shares.json failed, shares.json left untouched: %w", err)
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

// expiryNotifierBackend is implemented by backends that run their own expiry
// cleanup and can report what they removed. Kept out of StorageBackend so a
// backend without a cleanup loop is not forced to stub it.
type expiryNotifierBackend interface {
	SetExpiryNotifier(func(context.Context, *models.Share))
}

// expiryRunnerBackend is implemented by backends with an expiry sweep that can
// be run on demand.
type expiryRunnerBackend interface {
	RunExpiryCleanup(context.Context)
}

// RunExpiryCleanup runs one expiry sweep now, if the backend supports it.
// Reports whether a sweep actually ran.
func (s *Storage) RunExpiryCleanup(ctx context.Context) bool {
	b, ok := s.backend.(expiryRunnerBackend)
	if !ok {
		return false
	}
	b.RunExpiryCleanup(ctx)
	return true
}

// SetExpiryNotifier registers a callback invoked for every share the backend's
// expiry cleanup deletes. No-op for backends that do not support it — the
// return value says whether the callback was actually registered, so a caller
// can tell "wired" from "silently dropped".
func (s *Storage) SetExpiryNotifier(fn func(context.Context, *models.Share)) bool {
	b, ok := s.backend.(expiryNotifierBackend)
	if !ok {
		return false
	}
	b.SetExpiryNotifier(fn)
	return true
}

// ============= Share Operations =============

// Save saves or updates a share
func (s *Storage) Save(ctx context.Context, share *models.Share) error {
	return s.backend.Save(ctx, share)
}

// Get retrieves a share by ID
func (s *Storage) Get(ctx context.Context, id string) (*models.Share, bool) {
	return s.backend.Get(ctx, id)
}

// GetAll returns all non-expired shares
func (s *Storage) GetAll(ctx context.Context) []*models.Share {
	return s.backend.GetAll(ctx)
}

// Delete removes a share and its file
func (s *Storage) Delete(ctx context.Context, id string) error {
	return s.backend.Delete(ctx, id)
}

// IncrementDownloads atomically increments the download counter with limit check.
// Returns (true, nil) on success, (false, nil) if the limit was reached.
func (s *Storage) IncrementDownloads(ctx context.Context, id string) (bool, error) {
	return s.backend.IncrementDownloads(ctx, id)
}

// ============= Query Operations =============

// GetExpiringSoon returns shares expiring within the given duration
func (s *Storage) GetExpiringSoon(ctx context.Context, within time.Duration) ([]*models.Share, error) {
	return s.backend.GetExpiringSoon(ctx, within)
}

// GetByDateRange returns shares created within the date range
func (s *Storage) GetByDateRange(ctx context.Context, from, to time.Time) ([]*models.Share, error) {
	return s.backend.GetByDateRange(ctx, from, to)
}

// Search searches for shares by filename
func (s *Storage) Search(ctx context.Context, query string) ([]*models.Share, error) {
	return s.backend.Search(ctx, query)
}

// GetStats returns storage statistics
func (s *Storage) GetStats(ctx context.Context) (*StorageStats, error) {
	return s.backend.GetStats(ctx)
}

// ============= Folder Operations =============

// SaveFolderContents saves folder contents for a share
func (s *Storage) SaveFolderContents(ctx context.Context, shareID string, contents []*models.FolderEntry) error {
	return s.backend.SaveFolderContents(ctx, shareID, contents)
}

// GetFolderContents returns folder contents for a share at a specific path
func (s *Storage) GetFolderContents(ctx context.Context, shareID string, path string) ([]*models.FolderEntry, error) {
	return s.backend.GetFolderContents(ctx, shareID, path)
}

// DeleteFolderContents removes all folder contents for a share
func (s *Storage) DeleteFolderContents(ctx context.Context, shareID string) error {
	return s.backend.DeleteFolderContents(ctx, shareID)
}

// ============= Receive Link Operations =============

// SaveReceiveLink saves a receive link
func (s *Storage) SaveReceiveLink(ctx context.Context, link *models.ReceiveLink) error {
	return s.backend.SaveReceiveLink(ctx, link)
}

// GetReceiveLink retrieves a receive link by ID
func (s *Storage) GetReceiveLink(ctx context.Context, id string) (*models.ReceiveLink, bool) {
	return s.backend.GetReceiveLink(ctx, id)
}

// GetAllReceiveLinks returns all non-expired receive links
func (s *Storage) GetAllReceiveLinks(ctx context.Context) ([]*models.ReceiveLink, error) {
	return s.backend.GetAllReceiveLinks(ctx)
}

// DeleteReceiveLink removes a receive link
func (s *Storage) DeleteReceiveLink(ctx context.Context, id string) error {
	return s.backend.DeleteReceiveLink(ctx, id)
}

// IncrementReceiveLinkUploads atomically increments the upload counter with limit check.
// Returns (true, nil) on success, (false, nil) if the limit was reached.
func (s *Storage) IncrementReceiveLinkUploads(ctx context.Context, id string) (bool, error) {
	return s.backend.IncrementReceiveLinkUploads(ctx, id)
}

// ============= Received Files Operations =============

// SaveReceivedFile saves a received file record
func (s *Storage) SaveReceivedFile(ctx context.Context, file *models.ReceivedFile) error {
	return s.backend.SaveReceivedFile(ctx, file)
}

// GetReceivedFiles returns all files for a receive link
func (s *Storage) GetReceivedFiles(ctx context.Context, linkID string) ([]*models.ReceivedFile, error) {
	return s.backend.GetReceivedFiles(ctx, linkID)
}

// DeleteReceivedFile removes a received file within a transaction
func (s *Storage) DeleteReceivedFile(ctx context.Context, linkID, fileID string) error {
	return s.backend.DeleteReceivedFile(ctx, linkID, fileID)
}

// ============= User Operations =============

// CreateUser creates a new user
func (s *Storage) CreateUser(ctx context.Context, user *models.User) error {
	return s.backend.CreateUser(ctx, user)
}

// GetUser retrieves a user by ID
func (s *Storage) GetUser(ctx context.Context, id string) (*models.User, error) {
	return s.backend.GetUser(ctx, id)
}

// GetUserByEmail retrieves a user by email
func (s *Storage) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.backend.GetUserByEmail(ctx, email)
}

// GetUserByOIDC retrieves a user by OIDC subject and issuer
func (s *Storage) GetUserByOIDC(ctx context.Context, subject, issuer string) (*models.User, error) {
	return s.backend.GetUserByOIDC(ctx, subject, issuer)
}

// GetAllUsers returns all users
func (s *Storage) GetAllUsers(ctx context.Context) ([]*models.User, error) {
	return s.backend.GetAllUsers(ctx)
}

// UpdateUser updates an existing user
func (s *Storage) UpdateUser(ctx context.Context, user *models.User) error {
	return s.backend.UpdateUser(ctx, user)
}

// DeleteUser removes a user
func (s *Storage) DeleteUser(ctx context.Context, id string) error {
	return s.backend.DeleteUser(ctx, id)
}

// GetSharesByUser returns all non-expired shares for a user
func (s *Storage) GetSharesByUser(ctx context.Context, userID string) []*models.Share {
	return s.backend.GetSharesByUser(ctx, userID)
}

// GetReceiveLinksByUser returns all non-expired receive links for a user
func (s *Storage) GetReceiveLinksByUser(ctx context.Context, userID string) ([]*models.ReceiveLink, error) {
	return s.backend.GetReceiveLinksByUser(ctx, userID)
}

// GetUserUsage returns the managed storage (bytes) a user consumes.
func (s *Storage) GetUserUsage(ctx context.Context, userID string) (int64, error) {
	return s.backend.GetUserUsage(ctx, userID)
}

// ============= SMTP/Email Operations =============

// GetSMTPConfig returns the SMTP configuration
func (s *Storage) GetSMTPConfig(ctx context.Context) (*models.SMTPConfig, error) {
	return s.backend.GetSMTPConfig(ctx)
}

// SaveSMTPConfig saves the SMTP configuration
func (s *Storage) SaveSMTPConfig(ctx context.Context, config *models.SMTPConfig) error {
	return s.backend.SaveSMTPConfig(ctx, config)
}

// SaveEmailTransfer saves an email transfer record
func (s *Storage) SaveEmailTransfer(ctx context.Context, transfer *models.EmailTransferRecord) error {
	return s.backend.SaveEmailTransfer(ctx, transfer)
}

// DeleteEmailTransfer removes a transfer record whose e-mail could not be sent.
func (s *Storage) DeleteEmailTransfer(ctx context.Context, id string) error {
	return s.backend.DeleteEmailTransfer(ctx, id)
}

// GetEmailTransfersByShare returns all email transfers for a share
func (s *Storage) GetEmailTransfersByShare(ctx context.Context, shareID string) ([]*models.EmailTransferRecord, error) {
	return s.backend.GetEmailTransfersByShare(ctx, shareID)
}

// MarkEmailTransferDownloaded marks an email transfer as downloaded
func (s *Storage) MarkEmailTransferDownloaded(ctx context.Context, shareID string) error {
	return s.backend.MarkEmailTransferDownloaded(ctx, shareID)
}

// MarkEmailTransferNotified marks an email transfer as notified
func (s *Storage) MarkEmailTransferNotified(ctx context.Context, shareID string) error {
	return s.backend.MarkEmailTransferNotified(ctx, shareID)
}

// GetPendingDownloadNotifications returns email transfers that need download notifications
func (s *Storage) GetPendingDownloadNotifications(ctx context.Context, shareID string) ([]*models.EmailTransferRecord, error) {
	return s.backend.GetPendingDownloadNotifications(ctx, shareID)
}

// ============= API Key Operations =============

// CreateAPIKey creates a new API key record
func (s *Storage) CreateAPIKey(ctx context.Context, id, name, keyHash, prefix, userID, role string) error {
	return s.backend.CreateAPIKey(ctx, id, name, keyHash, prefix, userID, role)
}

// GetAPIKeyByHash looks up an API key by its SHA-256 hash
func (s *Storage) GetAPIKeyByHash(ctx context.Context, keyHash string) (id, name, userID, role string, isActive bool, err error) {
	return s.backend.GetAPIKeyByHash(ctx, keyHash)
}

// ListAPIKeys returns all API keys (without hashes)
func (s *Storage) ListAPIKeys(ctx context.Context) ([]map[string]interface{}, error) {
	return s.backend.ListAPIKeys(ctx)
}

// DeleteAPIKey removes an API key
func (s *Storage) DeleteAPIKey(ctx context.Context, id string) error {
	return s.backend.DeleteAPIKey(ctx, id)
}

// UpdateAPIKeyLastUsed updates the last_used_at timestamp for an API key
func (s *Storage) UpdateAPIKeyLastUsed(ctx context.Context, id string) {
	s.backend.UpdateAPIKeyLastUsed(ctx, id)
}

// DropSharesTableForTest removes the table the readiness probe reads. It exists
// so a test can prove Ping() actually queries application data instead of
// answering `select 1`; there is no production caller.
func (s *Storage) DropSharesTableForTest(ctx context.Context) error {
	return s.backend.DropSharesTableForTest(ctx)
}

// DropEmailTransfersTableForTest — see StorageBackend.DropEmailTransfersTableForTest.
func (s *Storage) DropEmailTransfersTableForTest(ctx context.Context) error {
	return s.backend.DropEmailTransfersTableForTest(ctx)
}

// RecordEvent appends one row to the activity log
func (s *Storage) RecordEvent(ctx context.Context, event *models.Event) error {
	return s.backend.RecordEvent(ctx, event)
}

// ListEvents returns one page of the activity log, newest first, and the total match count
func (s *Storage) ListEvents(ctx context.Context, filter models.EventFilter) ([]*models.Event, int, error) {
	return s.backend.ListEvents(ctx, filter)
}

// PurgeEventsBefore deletes activity-log rows older than cutoff
func (s *Storage) PurgeEventsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.backend.PurgeEventsBefore(ctx, cutoff)
}
