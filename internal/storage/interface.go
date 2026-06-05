package storage

import (
	"time"

	"casadrop/internal/models"
)

// StorageBackend defines the interface for storage backends
type StorageBackend interface {
	// Share operations
	Save(share *models.Share) error
	Get(id string) (*models.Share, bool)
	GetAll() []*models.Share
	Delete(id string) error
	IncrementDownloads(id string) (bool, error)

	// Query operations
	GetExpiringSoon(within time.Duration) ([]*models.Share, error)
	GetByDateRange(from, to time.Time) ([]*models.Share, error)
	Search(query string) ([]*models.Share, error)
	GetStats() (*StorageStats, error)

	// Folder operations (for v1.9)
	SaveFolderContents(shareID string, contents []*models.FolderEntry) error
	GetFolderContents(shareID string, path string) ([]*models.FolderEntry, error)
	DeleteFolderContents(shareID string) error

	// Receive link operations (for v2.0)
	SaveReceiveLink(link *models.ReceiveLink) error
	GetReceiveLink(id string) (*models.ReceiveLink, bool)
	GetAllReceiveLinks() ([]*models.ReceiveLink, error)
	DeleteReceiveLink(id string) error
	IncrementReceiveLinkUploads(id string) (bool, error)

	// Received files operations (for v2.0)
	SaveReceivedFile(file *models.ReceivedFile) error
	GetReceivedFiles(linkID string) ([]*models.ReceivedFile, error)
	DeleteReceivedFile(linkID, fileID string) error

	// User operations (for v2.1 multi-user)
	CreateUser(user *models.User) error
	GetUser(id string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByOIDC(subject, issuer string) (*models.User, error)
	GetAllUsers() ([]*models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(id string) error
	GetSharesByUser(userID string) []*models.Share
	GetReceiveLinksByUser(userID string) ([]*models.ReceiveLink, error)

	// SMTP/Email operations
	GetSMTPConfig() (*models.SMTPConfig, error)
	SaveSMTPConfig(config *models.SMTPConfig) error
	SaveEmailTransfer(transfer *models.EmailTransferRecord) error
	GetEmailTransfersByShare(shareID string) ([]*models.EmailTransferRecord, error)
	MarkEmailTransferDownloaded(shareID string) error
	MarkEmailTransferNotified(shareID string) error
	GetPendingDownloadNotifications(shareID string) ([]*models.EmailTransferRecord, error)

	// API Key operations
	CreateAPIKey(id, name, keyHash, prefix, userID, role string) error
	GetAPIKeyByHash(keyHash string) (id, name, userID, role string, isActive bool, err error)
	ListAPIKeys() ([]map[string]interface{}, error)
	DeleteAPIKey(id string) error
	UpdateAPIKeyLastUsed(id string)

	// Utility
	UploadsDir() string
	Ping() error
	Close() error
}

// StorageStats contains storage statistics
type StorageStats struct {
	TotalShares    int   `json:"total_shares"`
	TotalDownloads int   `json:"total_downloads"`
	TotalSize      int64 `json:"total_size"`
	ExpiringSoon   int   `json:"expiring_soon"` // Expiring within 24h
}
