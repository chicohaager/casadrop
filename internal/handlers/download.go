package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"casadrop/internal/auth"
	"casadrop/internal/utils"
)

// MediaType represents the type of media for preview purposes
type MediaType string

const (
	MediaTypeVideo   MediaType = "video"
	MediaTypeAudio   MediaType = "audio"
	MediaTypeImage   MediaType = "image"
	MediaTypePDF     MediaType = "pdf"
	MediaTypeText    MediaType = "text"
	MediaTypeUnknown MediaType = "unknown"
)

// getMediaType determines the media type from MIME type
func getMediaType(mimeType string) MediaType {
	switch {
	case strings.HasPrefix(mimeType, "video/"):
		return MediaTypeVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return MediaTypeAudio
	case strings.HasPrefix(mimeType, "image/"):
		return MediaTypeImage
	case mimeType == "application/pdf":
		return MediaTypePDF
	case strings.HasPrefix(mimeType, "text/") || mimeType == "application/json" || mimeType == "application/xml":
		return MediaTypeText
	default:
		return MediaTypeUnknown
	}
}

// isPreviewable checks if the file can be previewed in browser
func isPreviewable(mimeType string) bool {
	mediaType := getMediaType(mimeType)
	return mediaType != MediaTypeUnknown
}

// inlineContentType decides how a stored file may be served to the browser.
// Only a strict allow-list of media types is served inline with its real type;
// everything else — notably text/html, image/svg+xml and xml, which can execute
// script or render phishing UI in the app's own origin — is forced to an
// attachment download with a neutral content type. This prevents stored-XSS and
// same-origin phishing via uploaded files served from /stream.
func inlineContentType(mimeType string) (contentType string, inline bool) {
	switch {
	case strings.HasPrefix(mimeType, "video/"),
		strings.HasPrefix(mimeType, "audio/"),
		mimeType == "application/pdf":
		return mimeType, true
	case strings.HasPrefix(mimeType, "image/") && mimeType != "image/svg+xml":
		return mimeType, true
	default:
		return "application/octet-stream", false
	}
}

// sanitizeFilename produces a safe value for the quoted filename in a
// Content-Disposition header: it strips CR/LF and other control characters
// (response-header-injection defence) and escapes backslash and double-quote.
func sanitizeFilename(name string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(cleaned)
}

// DownloadFile handles file download requests
func (h *Handler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	share, ok := h.storage.Get(id)
	if !ok {
		http.Error(w, "Share not found or expired", http.StatusNotFound)
		return
	}

	// Check max downloads
	if share.MaxDownloads > 0 && share.Downloads >= share.MaxDownloads {
		http.Error(w, "Download limit reached", http.StatusForbidden)
		return
	}

	// Check password if required
	if share.HasPassword {
		clientIP := utils.GetClientIP(r)

		// Rate limit check for share passwords
		if h.sharePassLimiter.isBlocked(id, clientIP) {
			http.Error(w, "Too many password attempts. Please try again later.", http.StatusTooManyRequests)
			return
		}

		password := r.URL.Query().Get("password")
		if password == "" {
			password = r.Header.Get("X-Password")
		}
		if !auth.CheckPassword(password, share.Password) {
			attempts := h.sharePassLimiter.recordFailure(id, clientIP)
			if attempts >= sharePasswordMaxAttempts {
				http.Error(w, "Too many password attempts. Please try again later.", http.StatusTooManyRequests)
			} else {
				http.Error(w, "Invalid password", http.StatusUnauthorized)
			}
			return
		}
		// Correct password - reset attempts
		h.sharePassLimiter.resetAttempts(id, clientIP)
	}

	filePath := filepath.Join(h.storage.UploadsDir(), share.FileName)
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Atomically increment download counter and check limit
	allowed, err := h.storage.IncrementDownloads(id)
	if err != nil {
		log.Printf("Error incrementing downloads for %s: %v", id, err)
	}
	if !allowed {
		file.Close()
		http.Error(w, "Download limit reached", http.StatusGone)
		return
	}

	// Get updated share for webhook
	updatedShare, _ := h.storage.Get(id)

	// Send webhook notification
	clientIP := utils.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")
	if updatedShare != nil {
		h.webhook.NotifyDownload(updatedShare, clientIP, userAgent)

		// Check if download limit was just reached
		if updatedShare.MaxDownloads > 0 && updatedShare.Downloads >= updatedShare.MaxDownloads {
			h.webhook.NotifyLimitReached(updatedShare)
		}

		// Send email download notification if enabled
		if h.emailHandler != nil && h.emailHandler.IsEnabled() {
			go h.emailHandler.NotifyDownload(id, updatedShare.OriginalName)
		}
	}

	// Set headers for download
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, sanitizeFilename(share.OriginalName)))
	w.Header().Set("Content-Type", share.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(share.FileSize, 10))

	if _, err := io.Copy(w, file); err != nil {
		log.Printf("Error streaming file: %v", err)
		return
	}
}

// StreamFile serves files for inline viewing/streaming with Range request support
// This enables video seeking, audio scrubbing, and efficient media playback
func (h *Handler) StreamFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	share, ok := h.storage.Get(id)
	if !ok {
		http.Error(w, "Share not found or expired", http.StatusNotFound)
		return
	}

	// Check max downloads (for streams, only count first request, not range requests)
	if share.MaxDownloads > 0 && share.Downloads >= share.MaxDownloads {
		http.Error(w, "Download limit reached", http.StatusForbidden)
		return
	}

	// Check password if required
	if share.HasPassword {
		clientIP := utils.GetClientIP(r)

		// Rate limit check for share passwords
		if h.sharePassLimiter.isBlocked(id, clientIP) {
			http.Error(w, "Too many password attempts. Please try again later.", http.StatusTooManyRequests)
			return
		}

		password := r.URL.Query().Get("password")
		if password == "" {
			password = r.Header.Get("X-Password")
		}
		if password == "" {
			// Check cookie for password (set by share page)
			if cookie, err := r.Cookie("share_auth_" + id); err == nil {
				password = cookie.Value
			}
		}
		if !auth.CheckPassword(password, share.Password) {
			attempts := h.sharePassLimiter.recordFailure(id, clientIP)
			if attempts >= sharePasswordMaxAttempts {
				http.Error(w, "Too many password attempts. Please try again later.", http.StatusTooManyRequests)
			} else {
				http.Error(w, "Invalid password", http.StatusUnauthorized)
			}
			return
		}
		// Correct password - reset attempts
		h.sharePassLimiter.resetAttempts(id, clientIP)
	}

	filePath := filepath.Join(h.storage.UploadsDir(), share.FileName)

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Only increment download counter on first request (not range requests)
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		allowed, incErr := h.storage.IncrementDownloads(id)
		if incErr != nil {
			log.Printf("Error incrementing downloads for %s: %v", id, incErr)
		}
		if !allowed {
			file.Close()
			http.Error(w, "Download limit reached", http.StatusGone)
			return
		}

		// Send webhook notification
		updatedShare, _ := h.storage.Get(id)
		clientIP := utils.GetClientIP(r)
		userAgent := r.Header.Get("User-Agent")
		if updatedShare != nil {
			h.webhook.NotifyDownload(updatedShare, clientIP, userAgent)
			if updatedShare.MaxDownloads > 0 && updatedShare.Downloads >= updatedShare.MaxDownloads {
				h.webhook.NotifyLimitReached(updatedShare)
			}
		}
	}

	// Set headers for viewing. Only whitelisted media types are served inline;
	// anything that could execute in our origin (html/svg/xml/text/unknown) is
	// forced to an attachment download with a neutral content type.
	ctype, inline := inlineContentType(share.MimeType)
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, sanitizeFilename(share.OriginalName)))
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Accept-Ranges", "bytes")

	// CORS: only allow same-origin requests (no wildcard)
	origin := r.Header.Get("Origin")
	if origin != "" {
		// Exact-match the origin scheme+host against the request host.
		// strings.Contains is unsafe: "https://evil.com/attacker.example.com" would match host "attacker.example.com".
		if originURL, err := url.Parse(origin); err == nil && originURL.Host == r.Host {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Range, X-Password")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
			w.Header().Set("Vary", "Origin")
		}
	}

	// Handle OPTIONS preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Use http.ServeContent for proper Range request handling
	// This enables video seeking, audio scrubbing, and resume downloads
	http.ServeContent(w, r, share.OriginalName, fileInfo.ModTime(), file)
}
