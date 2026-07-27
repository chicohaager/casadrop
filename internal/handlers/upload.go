package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"casadrop/internal/auth"
	"casadrop/internal/middleware"
	"casadrop/internal/models"
	"casadrop/internal/utils"
)

// neverExpires is the sentinel stored in Share.ExpiresAt for shares
// that should not be auto-cleaned. The schema is NOT NULL, so we use
// a far-future date instead of nil. The cleanup worker compares against
// time.Now() so this naturally falls out of the expiry sweep.
var neverExpires = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

// expiresAtFromHours converts a UI-supplied hour count to a concrete
// ExpiresAt. expires_in <= 0 means "unbegrenzt" (no expiration); any
// positive value is treated as hours from now.
func expiresAtFromHours(hours int) time.Time {
	if hours <= 0 {
		return neverExpires
	}
	// Clamp to a sane maximum so a huge value can't overflow the int64
	// nanosecond duration and wrap to a past timestamp (born-expired share).
	return time.Now().Add(time.Duration(utils.ClampExpiryHours(hours)) * time.Hour)
}

// ChunkUpload represents an ongoing chunked upload
type ChunkUpload struct {
	ID             string
	FileName       string
	TotalSize      int64
	TotalChunks    int
	ChunksReceived map[int]bool
	ChunkSizes     map[int]int64 // per-index bytes on disk (for cumulative cap)
	ReceivedBytes  int64         // running sum of distinct chunk bytes written
	TempDir        string
	CreatedAt      time.Time
}

// chunkUploads stores ongoing chunked uploads
var (
	chunkUploads     = make(map[string]*ChunkUpload)
	chunkUploadsMu   sync.Mutex
	chunkCleanupOnce sync.Once
	chunkStopCh      = make(chan struct{})
	chunkStopOnce    sync.Once
)

// startChunkCleanupWorker starts a single background goroutine to clean up expired uploads.
// The goroutine exits when StopChunkCleanupWorker() is called.
func startChunkCleanupWorker() {
	chunkCleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(15 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					cleanupExpiredChunkUploads()
				case <-chunkStopCh:
					return
				}
			}
		}()
	})
}

// StopChunkCleanupWorker signals the cleanup goroutine to exit on graceful
// shutdown. Safe to call even if the worker was never started.
func StopChunkCleanupWorker() {
	chunkStopOnce.Do(func() {
		close(chunkStopCh)
	})
}

// cleanupExpiredChunkUploads removes uploads older than 24 hours
func cleanupExpiredChunkUploads() {
	chunkUploadsMu.Lock()
	defer chunkUploadsMu.Unlock()

	expiry := time.Now().Add(-24 * time.Hour)
	for id, upload := range chunkUploads {
		if upload.CreatedAt.Before(expiry) {
			os.RemoveAll(upload.TempDir)
			delete(chunkUploads, id)
			log.Printf("Cleaned up expired chunk upload: %s", id)
		}
	}
}

// UploadFile handles single file upload
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	// Get configurable max file size
	config, _ := h.loadTunnelConfig()
	maxSize := config.GetMaxFileSizeBytes()

	// Limit upload size
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	// 8 MB multipart memory budget; rest spills to disk (see receive.go)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		maxSizeGB := maxSize >> 30
		http.Error(w, fmt.Sprintf("File too large (max %d GB)", maxSizeGB), http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded. Please select a file.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check if file type is allowed (using config-based validation)
	if allowed, reason := config.IsExtensionAllowed(header.Filename); !allowed {
		http.Error(w, reason, http.StatusUnsupportedMediaType)
		return
	}

	// Per-user storage quota preflight (header.Size is the declared part size,
	// known before we write anything to disk).
	user := middleware.GetUserFromContext(r.Context())
	if user != nil {
		if over, err := h.quotaExceededBy(user.ID, header.Size); err != nil {
			http.Error(w, "Failed to check storage quota", http.StatusInternalServerError)
			return
		} else if over {
			http.Error(w, "Storage quota exceeded", http.StatusRequestEntityTooLarge)
			return
		}
	}

	// Parse form values
	password := r.FormValue("password")
	// expires_in semantics: positive = hours from now, 0/negative = unbegrenzt.
	// The frontend sends 0 when the user picks "Never (unlimited)".
	expiresInRaw, _ := strconv.Atoi(r.FormValue("expires_in"))
	hasExpiresIn := r.FormValue("expires_in") != ""
	expiresIn := expiresInRaw
	if !hasExpiresIn {
		expiresIn = 24 // legacy default when caller omitted the field
	}
	maxDownloads, _ := strconv.Atoi(r.FormValue("max_downloads"))

	// Generate unique ID and filename
	id := uuid.New().String()[:8]
	ext := filepath.Ext(header.Filename)
	storedName := fmt.Sprintf("%s%s", id, ext)

	// Save file
	destPath := filepath.Join(h.storage.UploadsDir(), storedName)
	dest, err := os.Create(destPath)
	if err != nil {
		log.Printf("Failed to create file %s: %v", destPath, err)
		http.Error(w, "Failed to save file. Please check server disk space.", http.StatusInternalServerError)
		return
	}
	defer dest.Close()

	size, err := io.Copy(dest, file)
	if err != nil {
		os.Remove(destPath)
		log.Printf("Failed to write file %s: %v", destPath, err)
		http.Error(w, "Failed to save file. Upload interrupted.", http.StatusInternalServerError)
		return
	}

	// Detect MIME type server-side (don't trust client-supplied Content-Type)
	dest.Sync()
	dest.Seek(0, 0)
	mimeType := detectMimeFromFile(dest, header.Filename)

	// Hash password if provided
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		os.Remove(destPath)
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	// user was resolved above for the quota preflight; reuse it for ownership.

	// Create share record
	share := &models.Share{
		ID:           id,
		FileName:     storedName,
		OriginalName: header.Filename,
		FileSize:     size,
		MimeType:     mimeType,
		Password:     hashedPassword,
		HasPassword:  password != "",
		ExpiresAt:    expiresAtFromHours(expiresIn),
		CreatedAt:    time.Now(),
		MaxDownloads: maxDownloads,
	}

	// Set user ownership if available
	if user != nil {
		share.UserID = user.ID
		share.UserEmail = user.Email
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

// InitChunkUpload initializes a new chunked upload
func (h *Handler) InitChunkUpload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileName    string `json:"fileName"`
		TotalSize   int64  `json:"totalSize"`
		TotalChunks int    `json:"totalChunks"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate
	if req.FileName == "" || req.TotalSize <= 0 || req.TotalChunks <= 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Check file type
	config, _ := h.loadTunnelConfig()
	if allowed, reason := config.IsExtensionAllowed(req.FileName); !allowed {
		http.Error(w, reason, http.StatusUnsupportedMediaType)
		return
	}

	// Check size limit
	maxSize := config.GetMaxFileSizeBytes()
	if req.TotalSize > maxSize {
		maxSizeGB := maxSize >> 30
		http.Error(w, fmt.Sprintf("File too large (max %d GB)", maxSizeGB), http.StatusRequestEntityTooLarge)
		return
	}

	// Per-user storage quota preflight — reject before any chunk is written.
	if user := middleware.GetUserFromContext(r.Context()); user != nil {
		if over, err := h.quotaExceededBy(user.ID, req.TotalSize); err != nil {
			http.Error(w, "Failed to check storage quota", http.StatusInternalServerError)
			return
		} else if over {
			http.Error(w, "Storage quota exceeded", http.StatusRequestEntityTooLarge)
			return
		}
	}

	// Generate upload ID
	uploadID := uuid.New().String()

	// Create temp directory for chunks
	tempDir := filepath.Join(h.storage.UploadsDir(), "chunks", uploadID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}

	upload := &ChunkUpload{
		ID:             uploadID,
		FileName:       req.FileName,
		TotalSize:      req.TotalSize,
		TotalChunks:    req.TotalChunks,
		ChunksReceived: make(map[int]bool),
		ChunkSizes:     make(map[int]int64),
		TempDir:        tempDir,
		CreatedAt:      time.Now(),
	}

	chunkUploadsMu.Lock()
	chunkUploads[uploadID] = upload
	chunkUploadsMu.Unlock()

	// Start cleanup worker (only once)
	startChunkCleanupWorker()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"uploadId": uploadID})
}

// UploadChunk receives a single chunk
func (h *Handler) UploadChunk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uploadID := vars["uploadId"]

	// Get upload info with lock
	chunkUploadsMu.Lock()
	upload, ok := chunkUploads[uploadID]
	if !ok {
		chunkUploadsMu.Unlock()
		http.Error(w, "Upload not found", http.StatusNotFound)
		return
	}
	// Copy needed values before unlocking
	totalChunks := upload.TotalChunks
	tempDir := upload.TempDir
	chunkUploadsMu.Unlock()

	// Parse chunk index from query
	chunkIndex, err := strconv.Atoi(r.URL.Query().Get("index"))
	if err != nil || chunkIndex < 0 || chunkIndex >= totalChunks {
		http.Error(w, "Invalid chunk index", http.StatusBadRequest)
		return
	}

	// Stream chunk data directly to file (memory efficient)
	chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", chunkIndex))
	chunkFile, err := os.Create(chunkPath)
	if err != nil {
		http.Error(w, "Failed to create chunk file", http.StatusInternalServerError)
		return
	}

	// Use buffered copy with size limit
	written, err := io.Copy(chunkFile, io.LimitReader(r.Body, 10<<20)) // 10MB max per chunk
	chunkFile.Close()
	if err != nil {
		os.Remove(chunkPath)
		http.Error(w, "Failed to save chunk", http.StatusInternalServerError)
		return
	}

	if written == 0 {
		os.Remove(chunkPath)
		http.Error(w, "Empty chunk received", http.StatusBadRequest)
		return
	}

	// Enforce a cumulative on-disk cap against the declared TotalSize. Without
	// this, InitChunkUpload's size/quota preflight (which trusts the declared
	// TotalSize) is meaningless: a client could declare TotalSize:1 with a huge
	// TotalChunks and stream 10 MB per index to disk until Finalize finally
	// rechecks — an authenticated disk-exhaustion / quota+size-cap bypass.
	chunkUploadsMu.Lock()
	current, ok := chunkUploads[uploadID]
	if !ok {
		chunkUploadsMu.Unlock()
		os.Remove(chunkPath)
		http.Error(w, "Upload not found", http.StatusNotFound)
		return
	}
	// Re-uploading an index replaces its bytes, so discount the previous size.
	newTotal := current.ReceivedBytes - current.ChunkSizes[chunkIndex] + written
	if newTotal > current.TotalSize {
		chunkUploadsMu.Unlock()
		os.Remove(chunkPath)
		http.Error(w, "Uploaded data exceeds declared file size", http.StatusRequestEntityTooLarge)
		return
	}
	current.ReceivedBytes = newTotal
	current.ChunkSizes[chunkIndex] = written
	current.ChunksReceived[chunkIndex] = true
	receivedCount := len(current.ChunksReceived)
	chunkUploadsMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"received": receivedCount,
		"total":    totalChunks,
	})
}

// FinalizeChunkUpload combines chunks and creates the share
func (h *Handler) FinalizeChunkUpload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uploadID := vars["uploadId"]

	// Snapshot everything we need under the lock. Reading upload.ChunksReceived
	// after unlocking would race with any in-flight UploadChunk calls that had
	// already resolved the map entry before we deleted it.
	chunkUploadsMu.Lock()
	upload, ok := chunkUploads[uploadID]
	var receivedCount int
	if ok {
		receivedCount = len(upload.ChunksReceived)
		delete(chunkUploads, uploadID)
	}
	chunkUploadsMu.Unlock()

	if !ok {
		http.Error(w, "Upload not found", http.StatusNotFound)
		return
	}

	// Verify all chunks received
	if receivedCount != upload.TotalChunks {
		os.RemoveAll(upload.TempDir)
		http.Error(w, fmt.Sprintf("Missing chunks: got %d of %d", receivedCount, upload.TotalChunks), http.StatusBadRequest)
		return
	}

	// Parse form data
	var req struct {
		Password     string `json:"password"`
		ExpiresIn    int    `json:"expires_in"`
		MaxDownloads int    `json:"max_downloads"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Try form values as fallback
		req.Password = r.FormValue("password")
		req.ExpiresIn, _ = strconv.Atoi(r.FormValue("expires_in"))
		req.MaxDownloads, _ = strconv.Atoi(r.FormValue("max_downloads"))
	}

	// expires_in semantics: positive = hours from now, 0/negative = unbegrenzt.
	// The chunk upload's JSON body / form fallback may legitimately carry 0.
	// Generate share ID
	id := uuid.New().String()[:8]
	ext := filepath.Ext(upload.FileName)
	storedName := fmt.Sprintf("%s%s", id, ext)
	destPath := filepath.Join(h.storage.UploadsDir(), storedName)

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		os.RemoveAll(upload.TempDir)
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}

	// Combine chunks in order using streaming (memory efficient)
	var totalSize int64
	copyBuffer := make([]byte, 256*1024) // 256KB buffer for efficient copying

	for i := 0; i < upload.TotalChunks; i++ {
		chunkPath := filepath.Join(upload.TempDir, fmt.Sprintf("chunk_%d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			destFile.Close()
			os.Remove(destPath)
			os.RemoveAll(upload.TempDir)
			http.Error(w, fmt.Sprintf("Failed to read chunk %d", i), http.StatusInternalServerError)
			return
		}

		n, err := io.CopyBuffer(destFile, chunkFile, copyBuffer)
		chunkFile.Close()

		if err != nil {
			destFile.Close()
			os.Remove(destPath)
			os.RemoveAll(upload.TempDir)
			http.Error(w, "Failed to write file", http.StatusInternalServerError)
			return
		}
		totalSize += n
	}
	destFile.Close()

	// Cleanup temp directory
	os.RemoveAll(upload.TempDir)

	// Per-user storage quota re-check with the ACTUAL assembled size — a client
	// could understate totalSize at init, so we verify against real bytes here
	// and roll back the assembled file if it would exceed the quota.
	if user := middleware.GetUserFromContext(r.Context()); user != nil {
		if over, err := h.quotaExceededBy(user.ID, totalSize); err != nil {
			os.Remove(destPath)
			http.Error(w, "Failed to check storage quota", http.StatusInternalServerError)
			return
		} else if over {
			os.Remove(destPath)
			http.Error(w, "Storage quota exceeded", http.StatusRequestEntityTooLarge)
			return
		}
	}

	// Hash password if provided
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		os.Remove(destPath)
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	// Detect MIME type
	mimeType := utils.UndetectedMimeType
	if f, err := os.Open(destPath); err == nil {
		mimeType = detectMimeFromFile(f, upload.FileName)
		f.Close()
	} else {
		log.Printf("MIME detection: cannot open %q: %v", destPath, err)
	}

	// Get user from context
	user := middleware.GetUserFromContext(r.Context())

	// Create share record
	share := &models.Share{
		ID:           id,
		FileName:     storedName,
		OriginalName: upload.FileName,
		FileSize:     totalSize,
		MimeType:     mimeType,
		Password:     hashedPassword,
		HasPassword:  req.Password != "",
		ExpiresAt:    expiresAtFromHours(req.ExpiresIn),
		CreatedAt:    time.Now(),
		MaxDownloads: req.MaxDownloads,
	}

	if user != nil {
		share.UserID = user.ID
		share.UserEmail = user.Email
	}

	if err := h.storage.Save(share); err != nil {
		os.Remove(destPath)
		http.Error(w, "Failed to save share", http.StatusInternalServerError)
		return
	}

	resp := share.ToResponse(fmt.Sprintf("%s/s/%s", h.getPrimaryBaseURL(r), share.ID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// UploadMultipleFiles handles multiple file uploads in a single request
func (h *Handler) UploadMultipleFiles(w http.ResponseWriter, r *http.Request) {
	// Get configurable max file size
	config, _ := h.loadTunnelConfig()
	maxSize := config.GetMaxFileSizeBytes()

	// Limit upload size
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	// 8 MB multipart memory budget; rest spills to disk
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		maxSizeGB := maxSize >> 30
		http.Error(w, fmt.Sprintf("Files too large (max %d GB total)", maxSizeGB), http.StatusRequestEntityTooLarge)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		// Fallback: try single file field
		files = r.MultipartForm.File["file"]
	}
	if len(files) == 0 {
		http.Error(w, "No files uploaded. Please select at least one file.", http.StatusBadRequest)
		return
	}

	// Limit number of files per upload (prevent DoS)
	const maxFilesPerUpload = 50
	if len(files) > maxFilesPerUpload {
		http.Error(w, fmt.Sprintf("Too many files. Maximum %d files per upload.", maxFilesPerUpload), http.StatusBadRequest)
		return
	}

	// Parse form values (shared for all files)
	password := r.FormValue("password")
	// expires_in semantics: positive = hours from now, 0/negative = unbegrenzt.
	expiresInRaw, _ := strconv.Atoi(r.FormValue("expires_in"))
	hasExpiresIn := r.FormValue("expires_in") != ""
	expiresIn := expiresInRaw
	if !hasExpiresIn {
		expiresIn = 24
	}
	maxDownloads, _ := strconv.Atoi(r.FormValue("max_downloads"))

	// Hash password once (shared for all files)
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	baseURL := h.getPrimaryBaseURL(r)
	var responses []models.ShareResponse
	var errors []string
	successCount := 0
	failedCount := 0

	// Get user from context for ownership
	user := middleware.GetUserFromContext(r.Context())

	// Quota context, computed once. quotaUsed is a running tally we grow as we
	// reserve each accepted file's bytes, so a single multi-file request can't
	// collectively blow past the cap. quotaCap == 0 means unlimited.
	var quotaCap, quotaUsed int64
	if user != nil {
		// Fail CLOSED on storage errors — matching the single-file path
		// (quotaExceededBy). Swallowing the error here (the old `err == nil`
		// pattern) left quotaCap=0 (unlimited) on a transient DB/lock error,
		// letting a whole batch bypass a nearly-full quota.
		u, err := h.storage.GetUser(user.ID)
		if err != nil {
			http.Error(w, "Failed to check storage quota", http.StatusInternalServerError)
			return
		}
		if u != nil {
			quotaCap = u.QuotaBytes
		}
		if quotaCap > 0 {
			used, err := h.storage.GetUserUsage(user.ID)
			if err != nil {
				http.Error(w, "Failed to check storage quota", http.StatusInternalServerError)
				return
			}
			quotaUsed = used
		}
	}

	for _, fileHeader := range files {
		// Check file type using config-based validation
		if allowed, reason := config.IsExtensionAllowed(fileHeader.Filename); !allowed {
			errors = append(errors, fmt.Sprintf("%s: %s", fileHeader.Filename, reason))
			failedCount++
			continue
		}

		// Per-user storage quota (reserve this file's bytes on acceptance).
		if quotaCap > 0 && quotaUsed+fileHeader.Size > quotaCap {
			errors = append(errors, fmt.Sprintf("%s: storage quota exceeded", fileHeader.Filename))
			failedCount++
			continue
		}
		quotaUsed += fileHeader.Size

		// Open uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to open", fileHeader.Filename))
			failedCount++
			continue
		}

		// Generate unique ID
		id := uuid.New().String()[:8]
		ext := filepath.Ext(fileHeader.Filename)
		storedName := fmt.Sprintf("%s%s", id, ext)
		destPath := filepath.Join(h.storage.UploadsDir(), storedName)

		// Save file
		dest, err := os.Create(destPath)
		if err != nil {
			file.Close()
			errors = append(errors, fmt.Sprintf("%s: failed to save", fileHeader.Filename))
			failedCount++
			continue
		}

		size, err := io.Copy(dest, file)
		file.Close()

		if err != nil {
			dest.Close()
			os.Remove(destPath)
			errors = append(errors, fmt.Sprintf("%s: failed to write", fileHeader.Filename))
			failedCount++
			continue
		}

		// Detect MIME type server-side (don't trust client-supplied Content-Type)
		dest.Sync()
		dest.Seek(0, 0)
		detectedMime := detectMimeFromFile(dest, fileHeader.Filename)
		dest.Close()

		// Create share record
		share := &models.Share{
			ID:           id,
			FileName:     storedName,
			OriginalName: fileHeader.Filename,
			FileSize:     size,
			MimeType:     detectedMime,
			Password:     hashedPassword,
			HasPassword:  password != "",
			ExpiresAt:    expiresAtFromHours(expiresIn),
			CreatedAt:    time.Now(),
			MaxDownloads: maxDownloads,
		}

		// Set user ownership if available
		if user != nil {
			share.UserID = user.ID
			share.UserEmail = user.Email
		}

		if err := h.storage.Save(share); err != nil {
			os.Remove(destPath)
			errors = append(errors, fmt.Sprintf("%s: failed to save share", fileHeader.Filename))
			failedCount++
			continue
		}

		responses = append(responses, share.ToResponse(fmt.Sprintf("%s/s/%s", baseURL, share.ID)))
		successCount++
	}

	// Return response
	resp := models.MultiUploadResponse{
		Shares:  responses,
		Success: successCount,
		Failed:  failedCount,
		Errors:  errors,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// detectMimeFromFile reads the sniff window from an already-open file and
// returns the refined type.
//
// When nothing could be read it returns utils.UndetectedMimeType and logs why,
// instead of inventing a plausible value. Both former defaults were wrong in
// their own way: "application/octet-stream" is *also* a genuine sniff result
// ("I looked and recognised nothing"), so a row written that way is
// indistinguishable from an examined one and the migration would later promote
// it by file name alone; and http.DetectContentType(nil) cheerfully answers
// "text/plain; charset=utf-8", recording a positive identification of a file
// nobody managed to read.
func detectMimeFromFile(f *os.File, fileName string) string {
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if n == 0 {
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("MIME detection: cannot read %q: %v", fileName, err)
		}
		return utils.UndetectedMimeType
	}
	return utils.DetectMimeType(buf[:n], fileName)
}
