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
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"casadrop/internal/auth"
	"casadrop/internal/middleware"
	"casadrop/internal/models"
	"casadrop/internal/utils"
)

// CreateReceiveLink creates a new receive link for uploads
func (h *Handler) CreateReceiveLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name              string `json:"name"`
		Password          string `json:"password"`
		ExpiresIn         int    `json:"expires_in"` // hours, 0 = never
		MaxUploads        int    `json:"max_uploads"`
		MaxFileSize       int64  `json:"max_file_size"` // bytes
		AllowedExtensions string `json:"allowed_extensions"`
		AutoShare         bool   `json:"auto_share"`
		WebhookURL        string `json:"webhook_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		req.Name = "Receive Link"
	}

	// Validate webhook URL if provided. Default: reject loopback/private/link-local
	// IPs to prevent SSRF into internal services — CasaDrop is commonly exposed
	// via Cloudflare Tunnel / Tailscale Funnel / Pangolin, so fail-closed is the
	// right default. Homelabs that genuinely need LAN webhook targets can opt
	// out with STRICT_WEBHOOK_URLS=false.
	if req.WebhookURL != "" {
		validateFn := func(u string, _ bool) error { return utils.ValidateExternalWebhookURL(u) }
		if os.Getenv("STRICT_WEBHOOK_URLS") == "false" {
			validateFn = utils.ValidateURL
		}
		if err := validateFn(req.WebhookURL, false); err != nil {
			http.Error(w, "Invalid webhook URL: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Generate ID
	id := uuid.New().String()[:8]

	// Hash password if provided
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	// Set expiration
	var expiresAt *time.Time
	if h := utils.ClampExpiryHours(req.ExpiresIn); h > 0 {
		exp := time.Now().Add(time.Duration(h) * time.Hour)
		expiresAt = &exp
	}

	// Get user from context for ownership
	user := middleware.GetUserFromContext(r.Context())

	// Create receive link
	link := &models.ReceiveLink{
		ID:                id,
		Name:              req.Name,
		Password:          hashedPassword,
		HasPassword:       req.Password != "",
		ExpiresAt:         expiresAt,
		CreatedAt:         time.Now(),
		MaxUploads:        req.MaxUploads,
		MaxFileSize:       req.MaxFileSize,
		AllowedExtensions: req.AllowedExtensions,
		AutoShare:         req.AutoShare,
		WebhookURL:        req.WebhookURL,
	}

	// Set user ownership if available
	if user != nil {
		link.UserID = user.ID
	}

	// Create uploads directory for this link
	uploadsDir := filepath.Join(h.storage.UploadsDir(), "received", id)
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Save to database
	if err := h.storage.SaveReceiveLink(link); err != nil {
		os.RemoveAll(uploadsDir)
		http.Error(w, "Failed to save receive link", http.StatusInternalServerError)
		return
	}

	// Return response
	resp := map[string]interface{}{
		"id":           link.ID,
		"name":         link.Name,
		"has_password": link.HasPassword,
		"expires_at":   link.ExpiresAt,
		"created_at":   link.CreatedAt,
		"max_uploads":  link.MaxUploads,
		"auto_share":   link.AutoShare,
		"receive_url":  fmt.Sprintf("%s/r/%s", h.getPrimaryBaseURL(r), link.ID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ListReceiveLinks returns all receive links (scoped by user role)
func (h *Handler) ListReceiveLinks(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	var links []*models.ReceiveLink
	var err error
	if user == nil || user.Role == models.RoleAdmin {
		links, err = h.storage.GetAllReceiveLinks()
	} else {
		links, err = h.storage.GetReceiveLinksByUser(user.ID)
	}
	if err != nil {
		http.Error(w, "Failed to get receive links", http.StatusInternalServerError)
		return
	}

	baseURL := h.getPrimaryBaseURL(r)

	var responses []map[string]interface{}
	for _, link := range links {
		files, _ := h.storage.GetReceivedFiles(link.ID)
		responses = append(responses, map[string]interface{}{
			"id":              link.ID,
			"name":            link.Name,
			"has_password":    link.HasPassword,
			"expires_at":      link.ExpiresAt,
			"created_at":      link.CreatedAt,
			"max_uploads":     link.MaxUploads,
			"current_uploads": link.CurrentUploads,
			"auto_share":      link.AutoShare,
			"total_size":      link.TotalSize,
			"files_count":     len(files),
			"receive_url":     fmt.Sprintf("%s/r/%s", baseURL, link.ID),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// GetReceiveLink returns details of a receive link
func (h *Handler) GetReceiveLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	link, ok := h.storage.GetReceiveLink(id)
	if !ok {
		http.Error(w, "Receive link not found or expired", http.StatusNotFound)
		return
	}

	// Check ownership: non-admin users can only view their own links
	user := middleware.GetUserFromContext(r.Context())
	if user != nil && user.Role != models.RoleAdmin {
		if link.UserID != "" && link.UserID != user.ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	files, _ := h.storage.GetReceivedFiles(id)

	resp := map[string]interface{}{
		"id":              link.ID,
		"name":            link.Name,
		"has_password":    link.HasPassword,
		"expires_at":      link.ExpiresAt,
		"created_at":      link.CreatedAt,
		"max_uploads":     link.MaxUploads,
		"current_uploads": link.CurrentUploads,
		"auto_share":      link.AutoShare,
		"total_size":      link.TotalSize,
		"files":           files,
		"receive_url":     fmt.Sprintf("%s/r/%s", h.getPrimaryBaseURL(r), link.ID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// DeleteReceiveLink removes a receive link
func (h *Handler) DeleteReceiveLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Check ownership
	user := middleware.GetUserFromContext(r.Context())
	link, ok := h.storage.GetReceiveLink(id)
	if ok {
		if user != nil && user.Role != models.RoleAdmin {
			if link.UserID != "" && link.UserID != user.ID {
				http.Error(w, "You can only delete your own receive links", http.StatusForbidden)
				return
			}
		}
	}

	if err := h.storage.DeleteReceiveLink(id); err != nil {
		http.Error(w, "Failed to delete receive link", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetReceivedFiles returns files for a receive link
func (h *Handler) GetReceivedFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Check ownership
	user := middleware.GetUserFromContext(r.Context())
	link, ok := h.storage.GetReceiveLink(id)
	if !ok {
		http.Error(w, "Receive link not found", http.StatusNotFound)
		return
	}
	if user != nil && user.Role != models.RoleAdmin {
		if link.UserID != "" && link.UserID != user.ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	files, err := h.storage.GetReceivedFiles(id)
	if err != nil {
		http.Error(w, "Failed to get files", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// ReceivePage displays the upload page for a receive link
func (h *Handler) ReceivePage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	link, ok := h.storage.GetReceiveLink(id)
	if !ok {
		h.templates.ExecuteTemplate(w, "not_found.html", nil)
		return
	}

	// Check if upload limit reached
	limitReached := link.MaxUploads > 0 && link.CurrentUploads >= link.MaxUploads

	data := map[string]interface{}{
		"ID":                link.ID,
		"Name":              link.Name,
		"HasPassword":       link.HasPassword,
		"MaxUploads":        link.MaxUploads,
		"CurrentUploads":    link.CurrentUploads,
		"MaxFileSize":       link.MaxFileSize,
		"AllowedExtensions": link.AllowedExtensions,
		"LimitReached":      limitReached,
	}

	h.templates.ExecuteTemplate(w, "receive.html", data)
}

// ReceiveUpload handles file uploads to a receive link
func (h *Handler) ReceiveUpload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	link, ok := h.storage.GetReceiveLink(id)
	if !ok {
		http.Error(w, "Receive link not found or expired", http.StatusNotFound)
		return
	}

	// Check upload limit
	if link.MaxUploads > 0 && link.CurrentUploads >= link.MaxUploads {
		http.Error(w, "Upload limit reached", http.StatusForbidden)
		return
	}

	// Check password if required
	if link.HasPassword {
		password := r.FormValue("password")
		if password == "" {
			password = r.Header.Get("X-Password")
		}
		if !auth.CheckPassword(password, link.Password) {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}
	}

	// Set max file size
	maxSize := link.MaxFileSize
	if maxSize <= 0 {
		// Use global config max file size
		config, _ := h.loadTunnelConfig()
		maxSize = config.GetMaxFileSizeBytes()
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	// Keep the in-memory portion of the multipart form small (8 MB). Anything
	// beyond that spills to disk. The previous 32 MB budget let an attacker
	// allocate 32 MB of RAM per request by stuffing many small form fields
	// before MaxBytesReader kicked in.
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check allowed extensions
	if link.AllowedExtensions != "" {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		allowed := false
		for _, allowedExt := range strings.Split(link.AllowedExtensions, ",") {
			allowedExt = strings.TrimSpace(strings.ToLower(allowedExt))
			if !strings.HasPrefix(allowedExt, ".") {
				allowedExt = "." + allowedExt
			}
			if ext == allowedExt {
				allowed = true
				break
			}
		}
		if !allowed {
			http.Error(w, fmt.Sprintf("File type %s not allowed", ext), http.StatusBadRequest)
			return
		}
	}

	// Generate unique filename
	fileID := uuid.New().String()[:8]
	ext := filepath.Ext(header.Filename)
	storedName := fmt.Sprintf("%s%s", fileID, ext)

	// Save file
	uploadsDir := filepath.Join(h.storage.UploadsDir(), "received", id)
	destPath := filepath.Join(uploadsDir, storedName)

	dest, err := os.Create(destPath)
	if err != nil {
		log.Printf("Failed to create file %s: %v", destPath, err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dest.Close()

	size, err := io.Copy(dest, file)
	if err != nil {
		os.Remove(destPath)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Detect MIME type — sync to ensure data is flushed before re-reading
	dest.Sync()
	dest.Seek(0, 0)
	buffer := make([]byte, 512)
	n, _ := dest.Read(buffer)
	mimeType := http.DetectContentType(buffer[:n])

	// Create received file record
	receivedFile := &models.ReceivedFile{
		ID:            fileID,
		ReceiveLinkID: id,
		FileName:      storedName,
		OriginalName:  header.Filename,
		FileSize:      size,
		MimeType:      mimeType,
		UploaderIP:    utils.GetClientIP(r),
		UploaderAgent: r.Header.Get("User-Agent"),
		CreatedAt:     time.Now(),
	}

	// Auto-share if enabled
	if link.AutoShare {
		shareID := uuid.New().String()[:8]
		shareStoredName := fmt.Sprintf("%s%s", shareID, ext)
		sharePath := filepath.Join(h.storage.UploadsDir(), shareStoredName)

		// Copy file to shares with proper error handling
		srcFile, err := os.Open(destPath)
		if err != nil {
			log.Printf("AutoShare: failed to open source file: %v", err)
		} else {
			destFile, err := os.Create(sharePath)
			if err != nil {
				srcFile.Close()
				log.Printf("AutoShare: failed to create share file: %v", err)
			} else {
				_, copyErr := io.Copy(destFile, srcFile)
				srcFile.Close()
				destFile.Close()

				if copyErr != nil {
					os.Remove(sharePath)
					log.Printf("AutoShare: failed to copy file: %v", copyErr)
				} else {
					share := &models.Share{
						ID:           shareID,
						FileName:     shareStoredName,
						OriginalName: header.Filename,
						FileSize:     size,
						MimeType:     mimeType,
						ExpiresAt:    time.Now().Add(24 * time.Hour),
						CreatedAt:    time.Now(),
					}

					if err := h.storage.Save(share); err == nil {
						receivedFile.ShareID = shareID
					} else {
						os.Remove(sharePath)
						log.Printf("AutoShare: failed to save share: %v", err)
					}
				}
			}
		}
	}

	// Save received file record. If this fails we must not proceed to bump the
	// upload counter / send webhooks — otherwise the file (and any auto-share)
	// is left on disk with no tracking record. Roll back and fail.
	if err := h.storage.SaveReceivedFile(receivedFile); err != nil {
		log.Printf("Failed to save received file record: %v", err)
		_ = os.Remove(destPath)
		if receivedFile.ShareID != "" {
			if s, ok := h.storage.Get(receivedFile.ShareID); ok {
				_ = os.Remove(filepath.Join(h.storage.UploadsDir(), s.FileName))
			}
			_ = h.storage.Delete(receivedFile.ShareID)
		}
		http.Error(w, "Failed to save upload", http.StatusInternalServerError)
		return
	}

	// Atomically increment upload counter (with limit check for race safety)
	allowed, incErr := h.storage.IncrementReceiveLinkUploads(id)
	if incErr != nil {
		log.Printf("Error incrementing uploads for receive link %s: %v", id, incErr)
	}
	if !allowed {
		// Race condition: limit was reached between the earlier check and now.
		// Roll back the just-saved file, DB record, and auto-created share so we
		// don't leak orphaned data when returning an error.
		_ = os.Remove(destPath)
		if receivedFile.ShareID != "" {
			if s, ok := h.storage.Get(receivedFile.ShareID); ok {
				_ = os.Remove(filepath.Join(h.storage.UploadsDir(), s.FileName))
			}
			_ = h.storage.Delete(receivedFile.ShareID)
		}
		_ = h.storage.DeleteReceivedFile(id, receivedFile.ID)
		http.Error(w, "Upload limit reached", http.StatusForbidden)
		return
	}

	// Send webhook if configured
	if link.WebhookURL != "" {
		go h.sendReceiveWebhook(link, receivedFile)
	}

	// Return response
	resp := map[string]interface{}{
		"success":   true,
		"file_id":   receivedFile.ID,
		"file_name": receivedFile.OriginalName,
		"file_size": receivedFile.FileSize,
		"share_id":  receivedFile.ShareID,
	}

	if receivedFile.ShareID != "" {
		resp["share_url"] = fmt.Sprintf("%s/s/%s", h.getPrimaryBaseURL(r), receivedFile.ShareID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// sendReceiveWebhook sends a webhook notification for received file
func (h *Handler) sendReceiveWebhook(link *models.ReceiveLink, file *models.ReceivedFile) {
	payload := map[string]interface{}{
		"event":        "file_received",
		"receive_link": link.ID,
		"receive_name": link.Name,
		"file_id":      file.ID,
		"file_name":    file.OriginalName,
		"file_size":    file.FileSize,
		"uploader_ip":  file.UploaderIP,
		"timestamp":    file.CreatedAt,
	}

	if file.ShareID != "" {
		payload["share_id"] = file.ShareID
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal webhook payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", link.WebhookURL, strings.NewReader(string(data)))
	if err != nil {
		log.Printf("Failed to create webhook request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Zima-Share/1.0")

	client := &http.Client{
		Timeout: 10 * time.Second,
		// Refuse redirects: a validated public webhook URL could otherwise 302 to
		// an internal address (127.0.0.1/169.254.169.254/RFC1918), bypassing the
		// SSRF guard applied at link-creation time (mirrors webhook.Service).
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send receive webhook: %v", err)
		return
	}
	defer resp.Body.Close()
	// Drain body to enable HTTP connection reuse
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		log.Printf("Receive webhook returned status %d", resp.StatusCode)
	}
}

// DownloadReceivedFile downloads a received file
func (h *Handler) DownloadReceivedFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	linkID := vars["id"]
	fileID := vars["fileId"]

	// Check ownership
	user := middleware.GetUserFromContext(r.Context())
	link, ok := h.storage.GetReceiveLink(linkID)
	if !ok {
		http.Error(w, "Receive link not found", http.StatusNotFound)
		return
	}
	if user != nil && user.Role != models.RoleAdmin {
		if link.UserID != "" && link.UserID != user.ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	files, err := h.storage.GetReceivedFiles(linkID)
	if err != nil {
		http.Error(w, "Receive link not found", http.StatusNotFound)
		return
	}

	var targetFile *models.ReceivedFile
	for _, f := range files {
		if f.ID == fileID {
			targetFile = f
			break
		}
	}

	if targetFile == nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	filePath := filepath.Join(h.storage.UploadsDir(), "received", linkID, targetFile.FileName)
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, strings.NewReplacer(`"`, `\"`, `\`, `\\`).Replace(targetFile.OriginalName)))
	w.Header().Set("Content-Type", targetFile.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(targetFile.FileSize, 10))

	if _, err := io.Copy(w, file); err != nil {
		log.Printf("Error streaming file: %v", err)
		return
	}
}
