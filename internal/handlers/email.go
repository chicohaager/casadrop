package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"casadrop/internal/email"
	"casadrop/internal/models"
	"casadrop/internal/storage"
	"casadrop/internal/utils"
)

// EmailHandler handles email-related HTTP requests
type EmailHandler struct {
	storage      *storage.Storage
	emailService *email.Service
	dataDir      string
	stop         chan struct{}
	stopOnce     sync.Once
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(store *storage.Storage) *EmailHandler {
	// Load SMTP config and create email service
	smtpConfig, err := store.GetSMTPConfig()
	if err != nil {
		log.Printf("Warning: Failed to load SMTP config: %v", err)
		smtpConfig = &models.SMTPConfig{}
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	return &EmailHandler{
		storage:      store,
		emailService: email.NewService(smtpConfig),
		dataDir:      dataDir,
		stop:         make(chan struct{}),
	}
}

// Stop signals the expiry notifier goroutine to exit. Safe to call multiple times.
func (h *EmailHandler) Stop() {
	h.stopOnce.Do(func() {
		close(h.stop)
	})
}

// GetSMTPConfig returns the current SMTP configuration
func (h *EmailHandler) GetSMTPConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.storage.GetSMTPConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Don't expose password in response
	config.Password = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// SaveSMTPConfig saves the SMTP configuration
func (h *EmailHandler) SaveSMTPConfig(w http.ResponseWriter, r *http.Request) {
	var config models.SMTPConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// If password is empty, keep the existing one
	if config.Password == "" {
		existingConfig, err := h.storage.GetSMTPConfig()
		if err == nil && existingConfig != nil {
			config.Password = existingConfig.Password
		}
	}

	if err := h.storage.SaveSMTPConfig(&config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update email service with new config
	h.emailService.UpdateConfig(&config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// TestSMTPConnection tests the SMTP connection
func (h *EmailHandler) TestSMTPConnection(w http.ResponseWriter, r *http.Request) {
	config, err := h.storage.GetSMTPConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update service with current config
	h.emailService.UpdateConfig(config)

	if err := h.emailService.TestConnection(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// EmailTransferRequest represents a request to send a file via email
type EmailTransferRequest struct {
	ShareID        string `json:"share_id"`
	RecipientEmail string `json:"recipient_email"`
	RecipientName  string `json:"recipient_name,omitempty"`
	SenderEmail    string `json:"sender_email"`
	SenderName     string `json:"sender_name,omitempty"`
	Title          string `json:"title,omitempty"`
	Message        string `json:"message,omitempty"`
	NotifyDownload bool   `json:"notify_download"`
}

// SendEmailTransfer sends a file transfer via email
func (h *EmailHandler) SendEmailTransfer(w http.ResponseWriter, r *http.Request) {
	var req EmailTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ShareID == "" || req.RecipientEmail == "" || req.SenderEmail == "" {
		http.Error(w, "share_id, recipient_email, and sender_email are required", http.StatusBadRequest)
		return
	}

	// Get share info
	share, exists := h.storage.Get(req.ShareID)
	if !exists {
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	// Build download URL using primary network setting
	downloadURL := h.getPrimaryBaseURL(r) + "/s/" + share.ID

	// Format file size
	fileSize := formatFileSize(share.FileSize)

	// Create email transfer record
	transfer := &models.EmailTransfer{
		ShareID:        req.ShareID,
		RecipientEmail: req.RecipientEmail,
		RecipientName:  req.RecipientName,
		SenderEmail:    req.SenderEmail,
		SenderName:     req.SenderName,
		Title:          req.Title,
		Message:        req.Message,
		NotifyDownload: req.NotifyDownload,
	}

	// Send email
	if err := h.emailService.SendTransferEmail(transfer, downloadURL, share.FileName, fileSize); err != nil {
		log.Printf("Failed to send email transfer: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to send email: " + err.Error()})
		return
	}

	// Save transfer record
	record := &models.EmailTransferRecord{
		ID:             uuid.New().String(),
		ShareID:        req.ShareID,
		RecipientEmail: req.RecipientEmail,
		RecipientName:  req.RecipientName,
		SenderEmail:    req.SenderEmail,
		SenderName:     req.SenderName,
		Title:          req.Title,
		Message:        req.Message,
		NotifyDownload: req.NotifyDownload,
		SentAt:         time.Now().UTC().Format(time.RFC3339),
	}

	if err := h.storage.SaveEmailTransfer(record); err != nil {
		log.Printf("Warning: Failed to save email transfer record: %v", err)
		// Don't fail the request - email was already sent
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Email sent successfully",
		"share_url":   downloadURL,
		"transfer_id": record.ID,
	})
}

// GetEmailStatus returns the email service status
func (h *EmailHandler) GetEmailStatus(w http.ResponseWriter, r *http.Request) {
	config, err := h.storage.GetSMTPConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled":    config.Enabled && config.Host != "",
		"configured": config.Host != "",
	})
}

// NotifyDownload sends download notification if needed
func (h *EmailHandler) NotifyDownload(shareID string, fileName string) {
	// Get pending notifications for this share
	transfers, err := h.storage.GetPendingDownloadNotifications(shareID)
	if err != nil {
		log.Printf("Failed to get pending notifications: %v", err)
		return
	}

	for _, transfer := range transfers {
		if err := h.emailService.SendDownloadNotification(
			transfer.SenderEmail,
			transfer.SenderName,
			transfer.RecipientEmail,
			fileName,
		); err != nil {
			log.Printf("Failed to send download notification: %v", err)
			continue
		}

		// Mark as notified
		if err := h.storage.MarkEmailTransferNotified(shareID); err != nil {
			log.Printf("Failed to mark notification as sent: %v", err)
		}
	}
}

// MarkDownloaded marks a share as downloaded for email notification
func (h *EmailHandler) MarkDownloaded(shareID string) {
	if err := h.storage.MarkEmailTransferDownloaded(shareID); err != nil {
		log.Printf("Failed to mark email transfer as downloaded: %v", err)
	}
}

// IsEnabled returns true if email service is enabled
func (h *EmailHandler) IsEnabled() bool {
	return h.emailService.IsEnabled()
}

// StartExpiryNotifier starts a background goroutine that checks for expiring shares
// every hour and sends email notifications to recipients. The goroutine exits
// when Stop() is called.
func (h *EmailHandler) StartExpiryNotifier() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				h.checkAndNotifyExpiring()
			case <-h.stop:
				return
			}
		}
	}()
}

// checkAndNotifyExpiring checks for shares expiring within 24 hours and notifies recipients
func (h *EmailHandler) checkAndNotifyExpiring() {
	if !h.IsEnabled() {
		return
	}

	// Get shares expiring in the next 24 hours
	shares, err := h.storage.GetExpiringSoon(24 * time.Hour)
	if err != nil {
		log.Printf("Expiry notifier: error getting expiring shares: %v", err)
		return
	}

	for _, share := range shares {
		// Get email transfers for this share
		transfers, err := h.storage.GetEmailTransfersByShare(share.ID)
		if err != nil || len(transfers) == 0 {
			continue
		}

		for _, transfer := range transfers {
			// Only notify if not already notified about expiry
			if transfer.NotifiedAt != "" {
				continue
			}

			// Send expiry warning email
			err := h.emailService.SendExpiryWarning(
				transfer.RecipientEmail,
				transfer.RecipientName,
				share.OriginalName,
				share.ExpiresAt,
			)
			if err != nil {
				log.Printf("Expiry notifier: failed to send to %s: %v", transfer.RecipientEmail, err)
				continue
			}

			// Mark as notified
			h.storage.MarkEmailTransferNotified(share.ID)
		}
	}
}

// loadTunnelConfig loads the tunnel configuration from file
func (h *EmailHandler) loadTunnelConfig() (*TunnelConfig, error) {
	configPath := filepath.Join(h.dataDir, "tunnel_config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return &TunnelConfig{}, nil
	}
	var config TunnelConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return &TunnelConfig{}, nil
	}
	return &config, nil
}

// getPrimaryBaseURL returns the base URL based on user's primary network setting
func (h *EmailHandler) getPrimaryBaseURL(r *http.Request) string {
	// NOTE: emails go to third parties, so links must NOT be derived from the
	// request host (X-Forwarded-Host is attacker-influenceable → phishing). Use
	// only the operator-configured primary network, falling back to the request
	// host as a last resort.
	tunnelCfg, _ := h.loadTunnelConfig()

	port := os.Getenv("EXTERNAL_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	switch tunnelCfg.PrimaryNetwork {
	case "tailscale":
		tailscaleURL := tunnelCfg.TailscaleURL
		if tailscaleURL == "" {
			tailscaleURL = os.Getenv("TAILSCALE_URL")
		}
		if tailscaleURL != "" {
			return strings.TrimSuffix(tailscaleURL, "/")
		}
	case "easytier":
		easytierIP := tunnelCfg.EasyTierIP
		if easytierIP == "" {
			easytierIP = os.Getenv("EASYTIER_IP")
		}
		if easytierIP != "" {
			return fmt.Sprintf("http://%s:%s", easytierIP, port)
		}
	case "custom":
		customURL := tunnelCfg.CustomURL
		if customURL == "" {
			customURL = os.Getenv("CUSTOM_URL")
		}
		if customURL != "" {
			return strings.TrimSuffix(customURL, "/")
		}
	case "local":
		localIP := os.Getenv("LOCAL_IP")
		if localIP != "" {
			return fmt.Sprintf("http://%s:%s", localIP, port)
		}
	case "cloudflare":
		if tunnelCfg.Enabled && tunnelCfg.URL != "" {
			return strings.TrimSuffix(tunnelCfg.URL, "/")
		}
		tunnelFile := filepath.Join(h.dataDir, "tunnel_url.txt")
		if data, err := os.ReadFile(tunnelFile); err == nil {
			url := strings.TrimSpace(string(data))
			if url != "" && url != "token" {
				return strings.TrimSuffix(url, "/")
			}
		}
	}

	// Fallback to request-based URL
	return utils.GetBaseURL(r)
}
