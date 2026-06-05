package handlers

import (
	"archive/zip"
	"bufio"
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
	"casadrop/internal/models"
	"casadrop/internal/utils"
)

// Maximum size for ZIP downloads (10 GB default)
const maxZipSize int64 = 10 << 30

// ShareFolder creates a share for an entire folder
func (h *Handler) ShareFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path         string `json:"path"`
		Password     string `json:"password"`
		ExpiresIn    int    `json:"expires_in"` // hours
		MaxDownloads int    `json:"max_downloads"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate path
	if req.Path == "" {
		http.Error(w, "Path is required", http.StatusBadRequest)
		return
	}

	// Security: Check for path traversal
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

	// Check allowed paths
	allowedPaths := os.Getenv("SHARE_ALLOWED_PATHS")
	if allowedPaths == "" {
		allowedPaths = "/DATA,/media,/home"
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

	// Check if path exists and is a directory
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Folder not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to access folder", http.StatusInternalServerError)
		}
		return
	}

	if !fileInfo.IsDir() {
		http.Error(w, "Path is not a directory", http.StatusBadRequest)
		return
	}

	// Scan directory and calculate total size
	var totalSize int64
	var totalFiles int
	var folderContents []*models.FolderEntry

	err = filepath.Walk(cleanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}

		// Skip hidden files
		if strings.HasPrefix(info.Name(), ".") && path != cleanPath {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Calculate relative path
		relPath, _ := filepath.Rel(cleanPath, path)
		if relPath == "." {
			return nil // Skip root
		}

		// Detect MIME type for files
		mimeType := ""
		if !info.IsDir() {
			mimeType = detectMimeType(path)
			totalSize += info.Size()
			totalFiles++
		}

		folderContents = append(folderContents, &models.FolderEntry{
			ID:           uuid.New().String()[:8],
			RelativePath: "/" + relPath,
			FileName:     info.Name(),
			FileSize:     info.Size(),
			MimeType:     mimeType,
			IsDirectory:  info.IsDir(),
		})

		return nil
	})

	if err != nil {
		http.Error(w, "Failed to scan folder", http.StatusInternalServerError)
		return
	}

	// Check size limit
	if totalSize > maxZipSize {
		http.Error(w, fmt.Sprintf("Folder too large (max %d GB for ZIP)", maxZipSize>>30), http.StatusBadRequest)
		return
	}

	// Generate share ID
	id := uuid.New().String()[:8]

	// Set defaults
	expiresIn := utils.ClampExpiryHours(req.ExpiresIn)
	if expiresIn <= 0 {
		expiresIn = 24
	}

	// Hash password if provided
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	// Create share record
	share := &models.Share{
		ID:           id,
		FileName:     "", // No single file
		OriginalName: filepath.Base(cleanPath),
		FileSize:     totalSize,
		MimeType:     "application/x-directory",
		Password:     hashedPassword,
		HasPassword:  req.Password != "",
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Hour),
		CreatedAt:    time.Now(),
		MaxDownloads: req.MaxDownloads,
		SourcePath:   cleanPath,
		IsDirectory:  true,
		TotalFiles:   totalFiles,
		TotalSize:    totalSize,
	}

	// Save share
	if err := h.storage.Save(share); err != nil {
		http.Error(w, "Failed to save share", http.StatusInternalServerError)
		return
	}

	// Save folder contents
	for _, entry := range folderContents {
		entry.ShareID = id
	}
	if err := h.storage.SaveFolderContents(id, folderContents); err != nil {
		h.storage.Delete(id)
		http.Error(w, "Failed to save folder contents", http.StatusInternalServerError)
		return
	}

	// Return response
	resp := share.ToResponse(fmt.Sprintf("%s/s/%s", h.getPrimaryBaseURL(r), share.ID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetFolderContents returns the contents of a folder share
func (h *Handler) GetFolderContents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	share, ok := h.storage.Get(id)
	if !ok {
		http.Error(w, "Share not found or expired", http.StatusNotFound)
		return
	}

	if !share.IsDirectory {
		http.Error(w, "Not a folder share", http.StatusBadRequest)
		return
	}

	// Check password
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

	// Get path parameter
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	// Get folder contents
	contents, err := h.storage.GetFolderContents(id, path)
	if err != nil {
		http.Error(w, "Failed to get folder contents", http.StatusInternalServerError)
		return
	}

	// Build response
	type FolderResponse struct {
		ShareID    string                `json:"share_id"`
		ShareName  string                `json:"share_name"`
		Path       string                `json:"path"`
		TotalFiles int                   `json:"total_files"`
		TotalSize  int64                 `json:"total_size"`
		Entries    []*models.FolderEntry `json:"entries"`
	}

	resp := FolderResponse{
		ShareID:    share.ID,
		ShareName:  share.OriginalName,
		Path:       path,
		TotalFiles: share.TotalFiles,
		TotalSize:  share.TotalSize,
		Entries:    contents,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// DownloadFolderFile downloads a single file from a folder share
func (h *Handler) DownloadFolderFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	share, ok := h.storage.Get(id)
	if !ok {
		http.Error(w, "Share not found or expired", http.StatusNotFound)
		return
	}

	if !share.IsDirectory {
		http.Error(w, "Not a folder share", http.StatusBadRequest)
		return
	}

	// Check download limit
	if share.MaxDownloads > 0 && share.Downloads >= share.MaxDownloads {
		http.Error(w, "Download limit reached", http.StatusForbidden)
		return
	}

	// Check password
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

	// Get file path
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "File path required", http.StatusBadRequest)
		return
	}

	// Security: Validate path - prevent path traversal attacks
	cleanPath := filepath.Clean(filePath)

	// Build full path
	fullPath := filepath.Join(share.SourcePath, cleanPath)

	// Resolve symlinks and get absolute paths for reliable comparison
	fullPathAbs, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	sourcePathAbs, err := filepath.EvalSymlinks(share.SourcePath)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Verify resolved path is within share source (must have trailing separator to prevent prefix attacks)
	if !strings.HasPrefix(fullPathAbs, sourcePathAbs+string(filepath.Separator)) && fullPathAbs != sourcePathAbs {
		http.Error(w, "Path traversal detected", http.StatusForbidden)
		return
	}

	// Open file using resolved path (not the original which could differ via symlinks)
	file, err := os.Open(fullPathAbs)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	if fileInfo.IsDir() {
		http.Error(w, "Cannot download directory", http.StatusBadRequest)
		return
	}

	// Set headers
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, strings.NewReplacer(`"`, `\"`, `\`, `\\`).Replace(fileInfo.Name())))
	// Detect MIME from the validated, symlink-resolved path — not the raw
	// input path, which could be swapped to escape the share dir (TOCTOU).
	w.Header().Set("Content-Type", detectMimeType(fullPathAbs))
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	if _, err := io.Copy(w, file); err != nil {
		log.Printf("Error streaming file: %v", err)
		return
	}
}

// DownloadFolderZip downloads the entire folder as a ZIP file
func (h *Handler) DownloadFolderZip(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	share, ok := h.storage.Get(id)
	if !ok {
		http.Error(w, "Share not found or expired", http.StatusNotFound)
		return
	}

	if !share.IsDirectory {
		http.Error(w, "Not a folder share", http.StatusBadRequest)
		return
	}

	// Check download limit
	if share.MaxDownloads > 0 && share.Downloads >= share.MaxDownloads {
		http.Error(w, "Download limit reached", http.StatusForbidden)
		return
	}

	// Check password
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

	// Verify source path still exists
	if _, err := os.Stat(share.SourcePath); os.IsNotExist(err) {
		http.Error(w, "Folder no longer exists", http.StatusNotFound)
		return
	}

	// Atomically increment download counter and check limit
	allowed, incErr := h.storage.IncrementDownloads(id)
	if incErr != nil {
		log.Printf("Error incrementing downloads for folder %s: %v", id, incErr)
	}
	if !allowed {
		http.Error(w, "Download limit reached", http.StatusGone)
		return
	}

	// Set headers for ZIP download
	zipName := share.OriginalName + ".zip"
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, strings.NewReplacer(`"`, `\"`, `\`, `\\`).Replace(zipName)))
	w.Header().Set("Content-Type", "application/zip")

	// Create buffered writer for better TCP performance
	bufferedWriter := bufio.NewWriterSize(w, 256*1024) // 256KB buffer
	defer bufferedWriter.Flush()

	// Create ZIP writer
	zipWriter := zip.NewWriter(bufferedWriter)
	defer zipWriter.Close()

	// Reusable buffer for file copying
	copyBuffer := make([]byte, 32*1024) // 32KB copy buffer

	// Enforce a max total uncompressed size per streamed ZIP. This guards
	// against a single request holding a server connection open for hours
	// while traversing a TB-scale directory. Configurable via env; 10 GB
	// is a sensible homelab default.
	maxZipBytes := getMaxFolderZipBytes()
	var streamedBytes int64
	errZipBudgetExceeded := fmt.Errorf("folder zip exceeded %d bytes budget", maxZipBytes)

	// Walk directory and add files to ZIP
	err := filepath.Walk(share.SourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}

		// Check for client disconnect
		select {
		case <-r.Context().Done():
			return r.Context().Err()
		default:
		}

		// Enforce byte budget (checked per entry, before we stream data)
		if !info.IsDir() && maxZipBytes > 0 && streamedBytes+info.Size() > maxZipBytes {
			return errZipBudgetExceeded
		}

		// Skip hidden files
		if strings.HasPrefix(info.Name(), ".") && path != share.SourcePath {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Calculate relative path for ZIP
		relPath, err := filepath.Rel(share.SourcePath, path)
		if err != nil {
			return nil
		}

		if relPath == "." {
			return nil
		}

		// Defense-in-depth against zip-slip: refuse any relative path that
		// escapes the share root. filepath.Walk does not follow symlinks by
		// default, so in practice relPath should never start with ".." — but
		// we never want an attacker-controlled archive entry writing outside
		// the extraction directory on the consumer side.
		if strings.HasPrefix(relPath, "..") || strings.Contains(relPath, string(filepath.Separator)+"..") {
			return nil
		}

		// Create ZIP header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return nil
		}
		// ZIP spec requires forward slashes regardless of host OS.
		header.Name = filepath.ToSlash(relPath)
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		// Create entry in ZIP
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return nil
		}

		// Write file contents - close file immediately after use to prevent resource leak
		if !info.IsDir() {
			if err := copyFileToZip(path, writer, copyBuffer); err != nil {
				log.Printf("Error writing %s to ZIP: %v", path, err)
			}
			streamedBytes += info.Size()
		}

		return nil
	})

	if err != nil {
		if err == errZipBudgetExceeded {
			log.Printf("ZIP budget exceeded for share %s (%d bytes)", id, maxZipBytes)
		} else {
			log.Printf("Error creating ZIP for share %s: %v", id, err)
		}
	}
}

// getMaxFolderZipBytes returns the configured max uncompressed size for
// folder ZIP downloads. Defaults to 10 GB, override with MAX_FOLDER_ZIP_GB.
func getMaxFolderZipBytes() int64 {
	const defaultGB int64 = 10
	raw := os.Getenv("MAX_FOLDER_ZIP_GB")
	if raw == "" {
		return defaultGB << 30
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return defaultGB << 30
	}
	// Clamp before shifting: n<<30 overflows int64 for n > ~8.5e9 and would
	// wrap negative, silently *disabling* the budget instead of enforcing it.
	const maxGB int64 = 1 << 20 // 1 PiB ceiling — far above any real folder
	if n > maxGB {
		n = maxGB
	}
	return n << 30
}

// copyFileToZip copies a file to the ZIP writer with proper resource cleanup
func copyFileToZip(path string, writer io.Writer, buffer []byte) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.CopyBuffer(writer, file, buffer)
	return err
}

// detectMimeType tries to detect MIME type from file
func detectMimeType(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return "application/octet-stream"
	}

	return http.DetectContentType(buffer[:n])
}
