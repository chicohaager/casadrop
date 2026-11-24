package storage

import (
	"context"
	"time"

	"casadrop/internal/models"
)

// StorageBackend defines the interface for storage backends
type StorageBackend interface {
	// Share operations
	Save(ctx context.Context, share *models.Share) error
	Get(ctx context.Context, id string) (*models.Share, bool)
	GetAll(ctx context.Context) []*models.Share
	Delete(ctx context.Context, id string) error
	IncrementDownloads(ctx context.Context, id string) (bool, error)

	// Query operations
	GetExpiringSoon(ctx context.Context, within time.Duration) ([]*models.Share, error)
	GetByDateRange(ctx context.Context, from, to time.Time) ([]*models.Share, error)
	Search(ctx context.Context, query string) ([]*models.Share, error)
	GetStats(ctx context.Context) (*StorageStats, error)

	// Folder operations (for v1.9)
	SaveFolderContents(ctx context.Context, shareID string, contents []*models.FolderEntry) error
	GetFolderContents(ctx context.Context, shareID string, path string) ([]*models.FolderEntry, error)
	DeleteFolderContents(ctx context.Context, shareID string) error

	// Receive link operations (for v2.0)
	SaveReceiveLink(ctx context.Context, link *models.ReceiveLink) error
	GetReceiveLink(ctx context.Context, id string) (*models.ReceiveLink, bool)
	GetAllReceiveLinks(ctx context.Context) ([]*models.ReceiveLink, error)
	DeleteReceiveLink(ctx context.Context, id string) error
	IncrementReceiveLinkUploads(ctx context.Context, id string) (bool, error)

	// Received files operations (for v2.0)
	SaveReceivedFile(ctx context.Context, file *models.ReceivedFile) error
	GetReceivedFiles(ctx context.Context, linkID string) ([]*models.ReceivedFile, error)
	DeleteReceivedFile(ctx context.Context, linkID, fileID string) error

	// User operations (for v2.1 multi-user)
	CreateUser(ctx context.Context, user *models.User) error
	GetUser(ctx context.Context, id string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByOIDC(ctx context.Context, subject, issuer string) (*models.User, error)
	GetAllUsers(ctx context.Context) ([]*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id string) error
	GetSharesByUser(ctx context.Context, userID string) []*models.Share
	GetReceiveLinksByUser(ctx context.Context, userID string) ([]*models.ReceiveLink, error)
	GetUserUsage(ctx context.Context, userID string) (int64, error)

	// SMTP/Email operations
	GetSMTPConfig(ctx context.Context) (*models.SMTPConfig, error)
	SaveSMTPConfig(ctx context.Context, config *models.SMTPConfig) error
	SaveEmailTransfer(ctx context.Context, transfer *models.EmailTransferRecord) error
	DeleteEmailTransfer(ctx context.Context, id string) error
	GetEmailTransfersByShare(ctx context.Context, shareID string) ([]*models.EmailTransferRecord, error)
	MarkEmailTransferDownloaded(ctx context.Context, shareID string) error
	MarkEmailTransferNotified(ctx context.Context, shareID string) error
	GetPendingDownloadNotifications(ctx context.Context, shareID string) ([]*models.EmailTransferRecord, error)

	// API Key operations
	CreateAPIKey(ctx context.Context, id, name, keyHash, prefix, userID, role string) error
	GetAPIKeyByHash(ctx context.Context, keyHash string) (id, name, userID, role string, isActive bool, err error)
	ListAPIKeys(ctx context.Context) ([]map[string]interface{}, error)
	DeleteAPIKey(ctx context.Context, id string) error
	UpdateAPIKeyLastUsed(ctx context.Context, id string)

	// Activity log operations (v2.5)
	RecordEvent(ctx context.Context, event *models.Event) error
	ListEvents(ctx context.Context, filter models.EventFilter) ([]*models.Event, int, error)
	PurgeEventsBefore(ctx context.Context, cutoff time.Time) (int64, error)

	// Utility
	UploadsDir() string
	Ping() error
	Close() error

	// DropSharesTableForTest exists solely so a test can prove Ping() reads
	// application data rather than answering a constant. No production caller.
	DropSharesTableForTest(ctx context.Context) error
	// DropEmailTransfersTableForTest lets a test make SaveEmailTransfer fail
	// and prove that no e-mail goes out without its record. No production caller.
	DropEmailTransfersTableForTest(ctx context.Context) error
}

// StorageStats contains storage statistics
type StorageStats struct {
	TotalShares    int   `json:"total_shares"`
	TotalDownloads int   `json:"total_downloads"`
	TotalSize      int64 `json:"total_size"`
	ExpiringSoon   int   `json:"expiring_soon"` // Expiring within 24h
}
