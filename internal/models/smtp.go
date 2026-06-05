package models

// SMTPConfig stores SMTP server configuration for email transfers
type SMTPConfig struct {
	Enabled     bool   `json:"enabled"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"` // Omit in JSON responses
	FromEmail   string `json:"from_email"`
	FromName    string `json:"from_name"`
	UseTLS      bool   `json:"use_tls"`
	UseStartTLS bool   `json:"use_starttls"`
}

// EmailTransfer represents an email transfer request
type EmailTransfer struct {
	ShareID        string `json:"share_id"`
	RecipientEmail string `json:"recipient_email"`
	RecipientName  string `json:"recipient_name,omitempty"`
	SenderEmail    string `json:"sender_email"`
	SenderName     string `json:"sender_name,omitempty"`
	Title          string `json:"title,omitempty"`
	Message        string `json:"message,omitempty"`
	NotifyDownload bool   `json:"notify_download"`
}

// EmailTransferRecord stores email transfer history in database
type EmailTransferRecord struct {
	ID             string `json:"id"`
	ShareID        string `json:"share_id"`
	RecipientEmail string `json:"recipient_email"`
	RecipientName  string `json:"recipient_name,omitempty"`
	SenderEmail    string `json:"sender_email"`
	SenderName     string `json:"sender_name,omitempty"`
	Title          string `json:"title,omitempty"`
	Message        string `json:"message,omitempty"`
	NotifyDownload bool   `json:"notify_download"`
	SentAt         string `json:"sent_at"`
	DownloadedAt   string `json:"downloaded_at,omitempty"`
	NotifiedAt     string `json:"notified_at,omitempty"`
}
