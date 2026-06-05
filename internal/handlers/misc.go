package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	qrcode "github.com/skip2/go-qrcode"

	"casadrop/internal/auth"
	"casadrop/internal/models"
	"casadrop/internal/preview"
	"casadrop/internal/utils"
)

// WebhookConfig handles GET and POST for webhook configuration
func (h *Handler) WebhookConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var config models.WebhookConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate URL; default fail-closed against loopback/private/link-local
		// targets (SSRF guard). Opt out with STRICT_WEBHOOK_URLS=false.
		if config.URL != "" {
			if os.Getenv("STRICT_WEBHOOK_URLS") == "false" {
				if err := utils.ValidateURL(config.URL, false); err != nil {
					http.Error(w, "Invalid webhook URL: "+err.Error(), http.StatusBadRequest)
					return
				}
			} else if err := utils.ValidateExternalWebhookURL(config.URL); err != nil {
				http.Error(w, "Invalid webhook URL: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		if err := h.webhook.SaveConfig(config); err != nil {
			http.Error(w, "Failed to save webhook config", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// GET - return current config (without secret)
	config := h.webhook.GetConfig()
	config.Secret = "" // Don't expose secret in API response

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// TestWebhook sends a test webhook notification
func (h *Handler) TestWebhook(w http.ResponseWriter, r *http.Request) {
	config := h.webhook.GetConfig()
	if !config.Enabled || config.URL == "" {
		http.Error(w, "Webhook not configured", http.StatusBadRequest)
		return
	}

	// Create test share
	testShare := &models.Share{
		ID:           "test-123",
		OriginalName: "test-file.txt",
		Downloads:    1,
	}

	// Send test notification
	h.webhook.NotifyDownload(testShare, "127.0.0.1", "Zima-Share Test")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "test notification sent"})
}

// QRCode generates a QR code image for a share link
func (h *Handler) QRCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Check if share exists
	share, ok := h.storage.Get(id)
	if !ok || share == nil {
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	// Check if expired
	if time.Now().After(share.ExpiresAt) {
		http.Error(w, "Share expired", http.StatusGone)
		return
	}

	// Build the share URL
	shareURL := fmt.Sprintf("%s/s/%s", h.getPrimaryBaseURL(r), id)

	// Get size from query parameter (default 256)
	size := 256
	if sizeStr := r.URL.Query().Get("size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s >= 64 && s <= 1024 {
			size = s
		}
	}

	// Generate QR code as PNG
	png, err := qrcode.Encode(shareURL, qrcode.Medium, size)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(png)
}

// GetThumbnail returns a thumbnail for an image share
func (h *Handler) GetThumbnail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Check if thumbnail service is available
	if h.thumbnails == nil {
		http.Error(w, "Thumbnail service not available", http.StatusServiceUnavailable)
		return
	}

	// Get share
	share, ok := h.storage.Get(id)
	if !ok {
		http.Error(w, "Share not found or expired", http.StatusNotFound)
		return
	}

	// Check if this is an image
	if !preview.IsImage(share.MimeType) {
		http.Error(w, "Not an image file", http.StatusBadRequest)
		return
	}

	// Check password if required (thumbnails need auth too)
	if share.HasPassword {
		clientIP := utils.GetClientIP(r)

		if h.sharePassLimiter.isBlocked(id, clientIP) {
			http.Error(w, "Too many password attempts", http.StatusTooManyRequests)
			return
		}

		password := r.URL.Query().Get("password")
		if password == "" {
			password = r.Header.Get("X-Password")
		}
		if password == "" {
			if cookie, err := r.Cookie("share_auth_" + id); err == nil {
				password = cookie.Value
			}
		}
		if !auth.CheckPassword(password, share.Password) {
			h.sharePassLimiter.recordFailure(id, clientIP)
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}
		h.sharePassLimiter.resetAttempts(id, clientIP)
	}

	// Get the source file path
	filePath := filepath.Join(h.storage.UploadsDir(), share.FileName)

	// Generate or get cached thumbnail
	thumbPath, err := h.thumbnails.GetThumbnail(id, filePath)
	if err != nil {
		log.Printf("Failed to generate thumbnail for %s: %v", id, err)
		http.Error(w, "Failed to generate thumbnail", http.StatusInternalServerError)
		return
	}

	// Open and serve thumbnail
	thumbFile, err := os.Open(thumbPath)
	if err != nil {
		http.Error(w, "Thumbnail not found", http.StatusNotFound)
		return
	}
	defer thumbFile.Close()

	// Get file info for Content-Length
	thumbInfo, err := thumbFile.Stat()
	if err != nil {
		http.Error(w, "Failed to read thumbnail", http.StatusInternalServerError)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.FormatInt(thumbInfo.Size(), 10))
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24 hours

	if _, err := io.Copy(w, thumbFile); err != nil {
		log.Printf("Error streaming thumbnail: %v", err)
		return
	}
}
