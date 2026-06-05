package models

import (
	"time"
)

type Share struct {
	ID           string    `json:"id"`
	FileName     string    `json:"file_name"`
	OriginalName string    `json:"original_name"`
	FileSize     int64     `json:"file_size"`
	MimeType     string    `json:"mime_type"`
	Password     string    `json:"password,omitempty"`
	HasPassword  bool      `json:"has_password"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	Downloads    int       `json:"downloads"`
	MaxDownloads int       `json:"max_downloads"`         // 0 = unlimited
	SourcePath   string    `json:"source_path,omitempty"` // Original path for share-from-path
	IsSymlink    bool      `json:"is_symlink"`            // True if file is symlinked, not copied

	// Folder share fields (v1.9)
	IsDirectory   bool   `json:"is_directory,omitempty"`
	ParentShareID string `json:"parent_share_id,omitempty"`
	TotalFiles    int    `json:"total_files,omitempty"`
	TotalSize     int64  `json:"total_size,omitempty"`

	// User ownership fields (v2.1 multi-user)
	UserID    string `json:"user_id,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
}

// FolderEntry represents a file or directory within a folder share
type FolderEntry struct {
	ID           string `json:"id"`
	ShareID      string `json:"share_id"`
	RelativePath string `json:"relative_path"`
	FileName     string `json:"file_name"`
	FileSize     int64  `json:"file_size"`
	MimeType     string `json:"mime_type,omitempty"`
	IsDirectory  bool   `json:"is_directory"`
}

// ReceiveLink represents an upload receive link (v2.0)
type ReceiveLink struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Password          string     `json:"password,omitempty"`
	HasPassword       bool       `json:"has_password"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	MaxUploads        int        `json:"max_uploads"`        // 0 = unlimited
	MaxFileSize       int64      `json:"max_file_size"`      // 0 = use default
	AllowedExtensions string     `json:"allowed_extensions"` // comma-separated
	AutoShare         bool       `json:"auto_share"`         // auto-create share for received files
	WebhookURL        string     `json:"webhook_url,omitempty"`
	CurrentUploads    int        `json:"current_uploads"`
	TotalSize         int64      `json:"total_size"`

	// User ownership (v2.1 multi-user)
	UserID string `json:"user_id,omitempty"`
}

// ReceivedFile represents a file uploaded via a receive link
type ReceivedFile struct {
	ID            string    `json:"id"`
	ReceiveLinkID string    `json:"receive_link_id"`
	FileName      string    `json:"file_name"`
	OriginalName  string    `json:"original_name"`
	FileSize      int64     `json:"file_size"`
	MimeType      string    `json:"mime_type,omitempty"`
	UploaderIP    string    `json:"uploader_ip,omitempty"`
	UploaderAgent string    `json:"uploader_agent,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	ShareID       string    `json:"share_id,omitempty"` // If auto-shared
}

// ShareFromPathRequest represents a request to share an existing file
type ShareFromPathRequest struct {
	Path         string `json:"path"`
	Password     string `json:"password,omitempty"`
	ExpiresIn    int    `json:"expires_in"` // hours
	MaxDownloads int    `json:"max_downloads"`
	UseSymlink   bool   `json:"use_symlink"` // If true, create symlink instead of copy
}

type ShareResponse struct {
	ID           string    `json:"id"`
	FileName     string    `json:"file_name"`
	FileSize     int64     `json:"file_size"`
	MimeType     string    `json:"mime_type,omitempty"`
	IsDirectory  bool      `json:"is_directory,omitempty"`
	HasPassword  bool      `json:"has_password"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	Downloads    int       `json:"downloads"`
	MaxDownloads int       `json:"max_downloads"`
	ShareURL     string    `json:"share_url"`
}

// ToResponse converts a Share into its public JSON projection.
// Always use this helper in handlers instead of hand-rolling a
// ShareResponse literal — otherwise, adding a new field to
// ShareResponse requires updating every call site and one will be
// missed (as happened with MimeType in the 2026-04-14 pass, which
// broke share-list thumbnail rendering).
func (s *Share) ToResponse(shareURL string) ShareResponse {
	return ShareResponse{
		ID:           s.ID,
		FileName:     s.OriginalName,
		FileSize:     s.FileSize,
		MimeType:     s.MimeType,
		IsDirectory:  s.IsDirectory,
		HasPassword:  s.HasPassword,
		ExpiresAt:    s.ExpiresAt,
		CreatedAt:    s.CreatedAt,
		Downloads:    s.Downloads,
		MaxDownloads: s.MaxDownloads,
		ShareURL:     shareURL,
	}
}

// UpdateShareRequest represents a request to modify an existing share
type UpdateShareRequest struct {
	ExpiresInHours *int    `json:"expires_in_hours,omitempty"` // New expiry (hours from now), nil = no change
	MaxDownloads   *int    `json:"max_downloads,omitempty"`    // New max downloads (0=unlimited), nil = no change
	Password       *string `json:"password,omitempty"`         // "" = remove, non-empty = set new, nil = keep current
}

// MultiUploadResponse represents the response for multi-file uploads
type MultiUploadResponse struct {
	Shares  []ShareResponse `json:"shares"`
	Success int             `json:"success"`
	Failed  int             `json:"failed"`
	Errors  []string        `json:"errors,omitempty"`
}

// WebhookConfig represents webhook notification configuration
type WebhookConfig struct {
	Enabled        bool   `json:"enabled"`
	URL            string `json:"url"`
	OnDownload     bool   `json:"on_download"`      // Notify when file is downloaded
	OnExpire       bool   `json:"on_expire"`        // Notify when share expires
	OnLimitReached bool   `json:"on_limit_reached"` // Notify when download limit reached
	Secret         string `json:"secret,omitempty"` // HMAC secret for signature
}

// WebhookPayload represents the webhook notification payload
type WebhookPayload struct {
	Event     string    `json:"event"` // "download", "expire", "limit_reached"
	ShareID   string    `json:"share_id"`
	FileName  string    `json:"file_name"`
	Downloads int       `json:"downloads"`
	Timestamp time.Time `json:"timestamp"`
	ClientIP  string    `json:"client_ip,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
}
