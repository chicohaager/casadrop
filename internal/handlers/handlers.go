package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"casadrop/internal/auth"
	"casadrop/internal/config"
	"casadrop/internal/middleware"
	"casadrop/internal/models"
	"casadrop/internal/preview"
	"casadrop/internal/storage"
	"casadrop/internal/utils"
	"casadrop/internal/webhook"
)

// isFileTypeAllowed checks if the file extension is allowed using config defaults
func isFileTypeAllowed(filename string) bool {
	return config.IsFileTypeAllowed(filename)
}

type Handler struct {
	storage          *storage.Storage
	templates        *template.Template
	webhook          *webhook.Service
	sharePassLimiter *sharePasswordRateLimiter
	thumbnails       *preview.ThumbnailService
	emailHandler     *EmailHandler

	// Config cache
	tunnelConfigCache     *TunnelConfig
	tunnelConfigCacheTime time.Time
	tunnelConfigMu        sync.RWMutex
}

// sharePasswordRateLimiter tracks failed password attempts per share
type sharePasswordRateLimiter struct {
	attempts map[string]*shareAttempts // key: shareID:IP
	mu       sync.RWMutex
}

type shareAttempts struct {
	count    int
	lastFail time.Time
}

const (
	sharePasswordMaxAttempts = 5                // Max password attempts per share per IP
	sharePasswordWindow      = 15 * time.Minute // Reset window
)

func newSharePasswordRateLimiter() *sharePasswordRateLimiter {
	limiter := &sharePasswordRateLimiter{
		attempts: make(map[string]*shareAttempts),
	}
	// Cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.cleanup()
		}
	}()
	return limiter
}

func (l *sharePasswordRateLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for key, attempt := range l.attempts {
		if now.Sub(attempt.lastFail) > sharePasswordWindow {
			delete(l.attempts, key)
		}
	}
}

func (l *sharePasswordRateLimiter) isBlocked(shareID, ip string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	key := shareID + ":" + ip
	attempt, exists := l.attempts[key]
	if !exists {
		return false
	}
	// Check if window has passed
	if time.Since(attempt.lastFail) > sharePasswordWindow {
		return false
	}
	return attempt.count >= sharePasswordMaxAttempts
}

func (l *sharePasswordRateLimiter) recordFailure(shareID, ip string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := shareID + ":" + ip
	attempt, exists := l.attempts[key]
	if !exists || time.Since(attempt.lastFail) > sharePasswordWindow {
		l.attempts[key] = &shareAttempts{count: 1, lastFail: time.Now()}
		return 1
	}
	attempt.count++
	attempt.lastFail = time.Now()
	return attempt.count
}

func (l *sharePasswordRateLimiter) resetAttempts(shareID, ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := shareID + ":" + ip
	delete(l.attempts, key)
}

func New(s *storage.Storage, templatesDir string) (*Handler, error) {
	tmpl, err := template.ParseGlob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		return nil, err
	}

	// Initialize webhook service
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	webhookSvc := webhook.New(dataDir)

	// Initialize thumbnail service
	thumbSvc, err := preview.NewThumbnailService(dataDir)
	if err != nil {
		log.Printf("Warning: Failed to initialize thumbnail service: %v", err)
		// Continue without thumbnails - not critical
	}

	return &Handler{
		storage:          s,
		templates:        tmpl,
		webhook:          webhookSvc,
		sharePassLimiter: newSharePasswordRateLimiter(),
		thumbnails:       thumbSvc,
	}, nil
}

// Stop drains the handler's background services (webhook deliveries) on
// graceful shutdown.
func (h *Handler) Stop() {
	if h.webhook != nil {
		h.webhook.Stop()
	}
}

// SetEmailHandler sets the email handler for download notifications
func (h *Handler) SetEmailHandler(emailHandler *EmailHandler) {
	h.emailHandler = emailHandler
}

// getPrimaryBaseURL returns the base URL based on user's primary network setting
func (h *Handler) getPrimaryBaseURL(r *http.Request) string {
	// Reached via a public/tunnel host (Pangolin/Tailscale/custom domain)? Build
	// links from that host so they match how the user actually accessed the app.
	if u := utils.PreferredPublicBaseURL(r); u != "" {
		return u
	}

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
		// Check user config first — new field wins, legacy URL as fallback
		if tunnelCfg.CloudflareURL != "" {
			return strings.TrimSuffix(tunnelCfg.CloudflareURL, "/")
		}
		if tunnelCfg.URL != "" {
			return strings.TrimSuffix(tunnelCfg.URL, "/")
		}
		// Check auto-detected tunnel
		dataDir := os.Getenv("DATA_DIR")
		if dataDir == "" {
			dataDir = "./data"
		}
		tunnelFile := filepath.Join(dataDir, "tunnel_url.txt")
		if data, err := os.ReadFile(tunnelFile); err == nil {
			url := strings.TrimSpace(string(data))
			if url != "" && url != "token" {
				return strings.TrimSuffix(url, "/")
			}
		}
		// Fall back to env
		if tunnelURL := os.Getenv("TUNNEL_URL"); tunnelURL != "" {
			return strings.TrimSuffix(tunnelURL, "/")
		}
	}

	// Fallback to request-based URL
	return utils.GetBaseURL(r)
}

// API Handlers

// ShareFromPath creates a share from an existing file path without uploading
// This allows sharing files directly from the filesystem (e.g., ZimaOS file manager integration)
func (h *Handler) ShareFromPath(w http.ResponseWriter, r *http.Request) {
	var req models.ShareFromPathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate path
	if req.Path == "" {
		http.Error(w, "Path is required", http.StatusBadRequest)
		return
	}

	// Security: Check for path traversal attacks
	cleanPath := filepath.Clean(req.Path)
	if strings.Contains(cleanPath, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Resolve symlinks to get the real path for validation
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Check allowed base paths from environment (comma-separated)
	allowedPaths := os.Getenv("SHARE_ALLOWED_PATHS")
	if allowedPaths == "" {
		allowedPaths = "/DATA,/media,/home" // Default allowed paths for ZimaOS
	}

	pathAllowed := false
	for _, allowed := range strings.Split(allowedPaths, ",") {
		allowed = strings.TrimSpace(allowed)
		if strings.HasPrefix(resolvedPath, allowed+string(filepath.Separator)) || resolvedPath == allowed {
			pathAllowed = true
			break
		}
	}

	if !pathAllowed {
		http.Error(w, "Path not in allowed directories", http.StatusForbidden)
		return
	}

	// Check if file exists and get info
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to access file", http.StatusInternalServerError)
		}
		return
	}

	if fileInfo.IsDir() {
		http.Error(w, "Directories are not supported yet", http.StatusBadRequest)
		return
	}

	// Check file type using config-based validation
	config, _ := h.loadTunnelConfig()
	fileName := filepath.Base(cleanPath)
	if allowed, reason := config.IsExtensionAllowed(fileName); !allowed {
		http.Error(w, reason, http.StatusUnsupportedMediaType)
		return
	}

	// Generate unique ID
	id := uuid.New().String()[:8]
	ext := filepath.Ext(fileName)
	storedName := fmt.Sprintf("%s%s", id, ext)
	destPath := filepath.Join(h.storage.UploadsDir(), storedName)

	// Create symlink or copy file
	if req.UseSymlink {
		// Create symlink (faster, but file must remain accessible)
		if err := os.Symlink(cleanPath, destPath); err != nil {
			http.Error(w, "Failed to create symlink", http.StatusInternalServerError)
			return
		}
	} else {
		// Copy file (safer, independent of source)
		srcFile, err := os.Open(cleanPath)
		if err != nil {
			http.Error(w, "Failed to open source file", http.StatusInternalServerError)
			return
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			http.Error(w, "Failed to create destination file", http.StatusInternalServerError)
			return
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, srcFile); err != nil {
			os.Remove(destPath)
			http.Error(w, "Failed to copy file", http.StatusInternalServerError)
			return
		}
	}

	// Set defaults
	expiresIn := utils.ClampExpiryHours(req.ExpiresIn)
	if expiresIn <= 0 {
		expiresIn = 24 // Default 24 hours
	}

	// Hash password if provided
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		os.Remove(destPath)
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	// Detect MIME type
	mimeType := "application/octet-stream"
	if file, err := os.Open(destPath); err == nil {
		defer file.Close()
		buffer := make([]byte, 512)
		if n, err := file.Read(buffer); err == nil {
			mimeType = http.DetectContentType(buffer[:n])
		}
	}

	// Create share record
	share := &models.Share{
		ID:           id,
		FileName:     storedName,
		OriginalName: fileName,
		FileSize:     fileInfo.Size(),
		MimeType:     mimeType,
		Password:     hashedPassword,
		HasPassword:  req.Password != "",
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Hour),
		CreatedAt:    time.Now(),
		MaxDownloads: req.MaxDownloads,
		SourcePath:   cleanPath,
		IsSymlink:    req.UseSymlink,
	}

	if err := h.storage.Save(share); err != nil {
		os.Remove(destPath)
		http.Error(w, "Failed to save share", http.StatusInternalServerError)
		return
	}

	// Return response
	resp := share.ToResponse(fmt.Sprintf("%s/s/%s", h.getPrimaryBaseURL(r), share.ID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BrowseFiles returns directory contents for the file browser
func (h *Handler) BrowseFiles(w http.ResponseWriter, r *http.Request) {
	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		requestedPath = "/"
	}

	// Clean and validate path
	cleanPath := filepath.Clean(requestedPath)
	if strings.Contains(cleanPath, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Get allowed paths
	allowedPaths := os.Getenv("SHARE_ALLOWED_PATHS")
	if allowedPaths == "" {
		allowedPaths = "/DATA,/media,/home"
	}
	allowedList := strings.Split(allowedPaths, ",")
	for i := range allowedList {
		allowedList[i] = strings.TrimSpace(allowedList[i])
	}

	// If path is root, return allowed base directories
	if cleanPath == "/" || cleanPath == "" {
		type DirEntry struct {
			Name  string `json:"name"`
			Path  string `json:"path"`
			IsDir bool   `json:"is_dir"`
			Size  int64  `json:"size"`
		}

		var entries []DirEntry
		for _, allowed := range allowedList {
			if info, err := os.Stat(allowed); err == nil && info.IsDir() {
				entries = append(entries, DirEntry{
					Name:  filepath.Base(allowed),
					Path:  allowed,
					IsDir: true,
					Size:  0,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"path":    "/",
			"parent":  "",
			"entries": entries,
		})
		return
	}

	// Resolve symlinks before checking allowed paths
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Check if path is within allowed directories
	pathAllowed := false
	for _, allowed := range allowedList {
		if strings.HasPrefix(resolvedPath, allowed+string(filepath.Separator)) || resolvedPath == allowed {
			pathAllowed = true
			break
		}
	}

	if !pathAllowed {
		http.Error(w, "Path not in allowed directories", http.StatusForbidden)
		return
	}

	// Check if path exists
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Path not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to access path", http.StatusInternalServerError)
		}
		return
	}

	if !info.IsDir() {
		http.Error(w, "Path is not a directory", http.StatusBadRequest)
		return
	}

	// Read directory contents
	dirEntries, err := os.ReadDir(resolvedPath)
	if err != nil {
		http.Error(w, "Failed to read directory", http.StatusInternalServerError)
		return
	}

	type DirEntry struct {
		Name  string `json:"name"`
		Path  string `json:"path"`
		IsDir bool   `json:"is_dir"`
		Size  int64  `json:"size"`
	}

	var entries []DirEntry
	for _, entry := range dirEntries {
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		entryPath := filepath.Join(cleanPath, entry.Name())
		var size int64 = 0

		if info, err := entry.Info(); err == nil {
			size = info.Size()
		}

		entries = append(entries, DirEntry{
			Name:  entry.Name(),
			Path:  entryPath,
			IsDir: entry.IsDir(),
			Size:  size,
		})
	}

	// Sort: directories first, then files alphabetically
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	// Calculate parent path
	parent := filepath.Dir(cleanPath)
	isRoot := false
	for _, allowed := range allowedList {
		if cleanPath == allowed {
			isRoot = true
			break
		}
	}
	if isRoot {
		parent = "/"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"path":    cleanPath,
		"parent":  parent,
		"entries": entries,
	})
}

func (h *Handler) ListShares(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	baseURL := h.getPrimaryBaseURL(r)

	// Get shares based on user role
	var shares []*models.Share
	if user == nil || user.Role == models.RoleAdmin {
		// Admin or unauthenticated (legacy) sees all shares
		shares = h.storage.GetAll()
	} else {
		// Regular users only see their own shares
		shares = h.storage.GetSharesByUser(user.ID)
	}

	responses := make([]models.ShareResponse, len(shares))
	for i, share := range shares {
		responses[i] = share.ToResponse(fmt.Sprintf("%s/s/%s", baseURL, share.ID))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// GetStats returns share statistics (scoped by user role)
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	var shares []*models.Share
	if user == nil || user.Role == models.RoleAdmin {
		shares = h.storage.GetAll()
	} else {
		shares = h.storage.GetSharesByUser(user.ID)
	}

	stats := struct {
		TotalShares     int   `json:"total_shares"`
		TotalDownloads  int   `json:"total_downloads"`
		TotalSize       int64 `json:"total_size"`
		ProtectedShares int   `json:"protected_shares"`
		ExpiringSoon    int   `json:"expiring_soon"` // within 24 hours
	}{}

	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)

	for _, share := range shares {
		stats.TotalShares++
		stats.TotalDownloads += share.Downloads
		stats.TotalSize += share.FileSize
		if share.HasPassword {
			stats.ProtectedShares++
		}
		if share.ExpiresAt.Before(tomorrow) {
			stats.ExpiringSoon++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handler) DeleteShare(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Check ownership
	user := middleware.GetUserFromContext(r.Context())
	share, ok := h.storage.Get(id)
	if !ok {
		// Still try to delete in case it's expired but exists
		h.storage.Delete(id)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Check permission: admin can delete any share, others only their own
	if user != nil && user.Role != models.RoleAdmin {
		if share.UserID != "" && share.UserID != user.ID {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "You can only delete your own shares"})
			return
		}
	}

	if err := h.storage.Delete(id); err != nil {
		http.Error(w, "Failed to delete share", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetShareInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	share, ok := h.storage.Get(id)
	if !ok {
		http.Error(w, "Share not found or expired", http.StatusNotFound)
		return
	}

	// Check ownership: non-admin users can only view their own shares
	user := middleware.GetUserFromContext(r.Context())
	if user != nil && user.Role != models.RoleAdmin {
		if share.UserID != "" && share.UserID != user.ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	resp := share.ToResponse(fmt.Sprintf("%s/s/%s", h.getPrimaryBaseURL(r), share.ID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// UpdateShare modifies an existing share's expiry, max downloads, or password
func (h *Handler) UpdateShare(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get existing share
	share, ok := h.storage.Get(id)
	if !ok {
		http.Error(w, "Share not found or expired", http.StatusNotFound)
		return
	}

	// Check ownership (same pattern as DeleteShare)
	user := middleware.GetUserFromContext(r.Context())
	if user != nil && user.Role != models.RoleAdmin {
		if share.UserID != "" && share.UserID != user.ID {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "You can only edit your own shares"})
			return
		}
	}

	// Parse request
	var req models.UpdateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Apply updates
	if req.ExpiresInHours != nil {
		if *req.ExpiresInHours <= 0 {
			http.Error(w, "Expiry must be positive", http.StatusBadRequest)
			return
		}
		share.ExpiresAt = time.Now().Add(time.Duration(utils.ClampExpiryHours(*req.ExpiresInHours)) * time.Hour)
	}

	if req.MaxDownloads != nil {
		share.MaxDownloads = *req.MaxDownloads
	}

	if req.Password != nil {
		if *req.Password == "" {
			// Remove password
			share.Password = ""
			share.HasPassword = false
		} else {
			// Set new password
			hashed, err := auth.HashPassword(*req.Password)
			if err != nil {
				http.Error(w, "Failed to process password", http.StatusInternalServerError)
				return
			}
			share.Password = hashed
			share.HasPassword = true
		}
	}

	// Save (INSERT OR REPLACE preserves all other fields)
	if err := h.storage.Save(share); err != nil {
		http.Error(w, "Failed to update share", http.StatusInternalServerError)
		return
	}

	resp := share.ToResponse(fmt.Sprintf("%s/s/%s", h.getPrimaryBaseURL(r), share.ID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BulkDeleteShares deletes multiple shares at once
func (h *Handler) BulkDeleteShares(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	deleted := 0
	errors := 0

	for _, id := range req.IDs {
		share, ok := h.storage.Get(id)
		if !ok {
			errors++
			continue
		}

		// Check ownership (admin can delete all, users only their own)
		if user != nil && user.Role != models.RoleAdmin && share.UserID != user.ID {
			errors++
			continue
		}

		if err := h.storage.Delete(id); err != nil {
			errors++
			continue
		}
		deleted++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deleted": deleted,
		"errors":  errors,
	})
}

// Page Handlers

func (h *Handler) IndexPage(w http.ResponseWriter, r *http.Request) {
	h.templates.ExecuteTemplate(w, "index.html", nil)
}

func (h *Handler) SharePage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	share, ok := h.storage.Get(id)
	if !ok {
		h.templates.ExecuteTemplate(w, "not_found.html", nil)
		return
	}

	// Check if this is a folder share
	if share.IsDirectory {
		data := map[string]interface{}{
			"ID":          share.ID,
			"FolderName":  share.OriginalName,
			"TotalFiles":  share.TotalFiles,
			"TotalSize":   utils.FormatFileSize(share.TotalSize),
			"HasPassword": share.HasPassword,
			"ExpiresAt":   share.ExpiresAt.Format("02.01.2006 15:04"),
		}
		h.templates.ExecuteTemplate(w, "folder.html", data)
		return
	}

	// Determine media type for preview
	mediaType := getMediaType(share.MimeType)
	canPreview := isPreviewable(share.MimeType)

	// Get file extension for icon selection
	ext := strings.ToLower(filepath.Ext(share.OriginalName))

	data := map[string]interface{}{
		"ID":          share.ID,
		"FileName":    share.OriginalName,
		"FileSize":    utils.FormatFileSize(share.FileSize),
		"FileSizeRaw": share.FileSize,
		"HasPassword": share.HasPassword,
		"ExpiresAt":   share.ExpiresAt.Format("02.01.2006 15:04"),
		"MimeType":    share.MimeType,
		"MediaType":   string(mediaType),
		"CanPreview":  canPreview,
		"IsVideo":     mediaType == MediaTypeVideo,
		"IsAudio":     mediaType == MediaTypeAudio,
		"IsImage":     mediaType == MediaTypeImage,
		"IsPDF":       mediaType == MediaTypePDF,
		"IsText":      mediaType == MediaTypeText,
		"FileExt":     ext,
	}

	h.templates.ExecuteTemplate(w, "share.html", data)
}

// TunnelConfig represents the user's network and admin configuration
type TunnelConfig struct {
	Enabled        bool   `json:"enabled"`
	URL            string `json:"url"`            // Legacy field, kept for compatibility
	CloudflareURL  string `json:"cloudflareUrl"`  // Cloudflare Tunnel URL (manual override)
	TailscaleURL   string `json:"tailscaleUrl"`   // Tailscale Funnel URL
	EasyTierIP     string `json:"easytierIp"`     // EasyTier IP address
	CustomURL      string `json:"customUrl"`      // Custom URL (WireGuard, Reverse Proxy, etc.)
	LocalIP        string `json:"localIp"`        // Local network IP (manual override)
	PrimaryNetwork string `json:"primaryNetwork"` // Which network to use for share links: cloudflare, tailscale, easytier, custom, local
	// Enabled flags - when false, network is hidden from display and auto-detection is skipped
	// Default to true (nil/missing = enabled) for backwards compatibility
	CloudflareEnabled *bool `json:"cloudflareEnabled,omitempty"`
	TailscaleEnabled  *bool `json:"tailscaleEnabled,omitempty"`
	EasyTierEnabled   *bool `json:"easytierEnabled,omitempty"`
	CustomEnabled     *bool `json:"customEnabled,omitempty"`
	LocalEnabled      *bool `json:"localEnabled,omitempty"`
	// Legacy disabled flags - kept for backwards compatibility, will be converted to enabled flags
	CloudflareDisabled bool `json:"cloudflareDisabled,omitempty"`
	TailscaleDisabled  bool `json:"tailscaleDisabled,omitempty"`
	EasyTierDisabled   bool `json:"easytierDisabled,omitempty"`
	CustomDisabled     bool `json:"customDisabled,omitempty"`
	LocalDisabled      bool `json:"localDisabled,omitempty"`
	// Admin settings
	MaxFileSizeGB     int    `json:"maxFileSizeGB"`     // Maximum file size in GB (0 = use default 10 GB)
	AllowedExtensions string `json:"allowedExtensions"` // Comma-separated list of allowed extensions (empty = all except blocked)
	BlockedExtensions string `json:"blockedExtensions"` // Comma-separated list of blocked extensions (default security list if empty)
}

// IsCloudflareEnabled returns true if Cloudflare network is enabled
func (c *TunnelConfig) IsCloudflareEnabled() bool {
	if c.CloudflareEnabled != nil {
		return *c.CloudflareEnabled
	}
	// Fall back to legacy disabled flag (inverted)
	return !c.CloudflareDisabled
}

// IsTailscaleEnabled returns true if Tailscale network is enabled
func (c *TunnelConfig) IsTailscaleEnabled() bool {
	if c.TailscaleEnabled != nil {
		return *c.TailscaleEnabled
	}
	return !c.TailscaleDisabled
}

// IsEasyTierEnabled returns true if EasyTier network is enabled
func (c *TunnelConfig) IsEasyTierEnabled() bool {
	if c.EasyTierEnabled != nil {
		return *c.EasyTierEnabled
	}
	return !c.EasyTierDisabled
}

// IsCustomEnabled returns true if Custom network is enabled
func (c *TunnelConfig) IsCustomEnabled() bool {
	if c.CustomEnabled != nil {
		return *c.CustomEnabled
	}
	return !c.CustomDisabled
}

// IsLocalEnabled returns true if Local network is enabled
func (c *TunnelConfig) IsLocalEnabled() bool {
	if c.LocalEnabled != nil {
		return *c.LocalEnabled
	}
	return !c.LocalDisabled
}

// GetMaxFileSizeBytes returns the maximum file size in bytes
func (c *TunnelConfig) GetMaxFileSizeBytes() int64 {
	if c.MaxFileSizeGB <= 0 {
		return 10 << 30 // Default: 10 GB
	}
	return int64(c.MaxFileSizeGB) << 30
}

// GetBlockedExtensions returns the list of blocked extensions
func (c *TunnelConfig) GetBlockedExtensions() map[string]bool {
	// If custom blocked extensions are set, use those
	if c.BlockedExtensions != "" {
		blocked := make(map[string]bool)
		for _, ext := range strings.Split(c.BlockedExtensions, ",") {
			ext = strings.TrimSpace(strings.ToLower(ext))
			if ext != "" {
				if !strings.HasPrefix(ext, ".") {
					ext = "." + ext
				}
				blocked[ext] = true
			}
		}
		return blocked
	}
	// Return default blocked extensions from config package
	return config.DefaultBlockedExtensions
}

// GetAllowedExtensions returns the list of allowed extensions (empty = all allowed)
func (c *TunnelConfig) GetAllowedExtensions() map[string]bool {
	if c.AllowedExtensions == "" {
		return nil // nil means all extensions allowed (except blocked)
	}
	allowed := make(map[string]bool)
	for _, ext := range strings.Split(c.AllowedExtensions, ",") {
		ext = strings.TrimSpace(strings.ToLower(ext))
		if ext != "" {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			allowed[ext] = true
		}
	}
	return allowed
}

// IsExtensionAllowed checks if a file extension is allowed based on config
func (c *TunnelConfig) IsExtensionAllowed(filename string) (bool, string) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return true, "" // Files without extension are allowed
	}

	// Check blocked extensions first
	blocked := c.GetBlockedExtensions()
	if blocked[ext] {
		return false, fmt.Sprintf("File type %s is blocked for security reasons", ext)
	}

	// If allowed extensions are specified, check against whitelist
	allowed := c.GetAllowedExtensions()
	if allowed != nil && !allowed[ext] {
		return false, fmt.Sprintf("File type %s is not in the allowed extensions list", ext)
	}

	return true, ""
}

func (h *Handler) getDataDir() string {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	return dataDir
}

func (h *Handler) getTunnelConfigPath() string {
	return filepath.Join(h.getDataDir(), "tunnel_config.json")
}

// tunnelConfigCacheTTL defines how long the tunnel config is cached
const tunnelConfigCacheTTL = 30 * time.Second

func (h *Handler) loadTunnelConfig() (*TunnelConfig, error) {
	// Check cache first
	h.tunnelConfigMu.RLock()
	if h.tunnelConfigCache != nil && time.Since(h.tunnelConfigCacheTime) < tunnelConfigCacheTTL {
		config := h.tunnelConfigCache
		h.tunnelConfigMu.RUnlock()
		return config, nil
	}
	h.tunnelConfigMu.RUnlock()

	// Load from file
	data, err := os.ReadFile(h.getTunnelConfigPath())
	if err != nil {
		return &TunnelConfig{}, nil // Return empty config if file doesn't exist
	}
	var config TunnelConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return &TunnelConfig{}, nil
	}

	// Update cache
	h.tunnelConfigMu.Lock()
	h.tunnelConfigCache = &config
	h.tunnelConfigCacheTime = time.Now()
	h.tunnelConfigMu.Unlock()

	return &config, nil
}

func (h *Handler) saveTunnelConfig(config *TunnelConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(h.getTunnelConfigPath(), data, 0644)
	if err != nil {
		return err
	}

	// Invalidate cache on save
	h.tunnelConfigMu.Lock()
	h.tunnelConfigCache = config
	h.tunnelConfigCacheTime = time.Now()
	h.tunnelConfigMu.Unlock()

	return nil
}

// formatFileSize returns a human-readable file size string
func formatFileSize(size int64) string {
	return utils.FormatFileSize(size)
}
