package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Host-filesystem browsing helpers, shared by /api/browse and
// /api/browse/thumbnail. Both endpoints are admin-only (see routes.go) and both
// must apply exactly the same SHARE_ALLOWED_PATHS rules — keeping the root list
// and the path resolution in one place is what stops the two from drifting.

// thumbnailableExts lists the extensions internal/preview can actually decode:
// the stdlib JPEG/PNG/GIF decoders plus BMP and TIFF from golang.org/x/image
// (registered in preview/thumbnails.go). WebP is deliberately absent — there is
// no pure-Go WebP decoder in this module's dependency set (golang.org/x/image
// no longer ships one), so a .webp entry keeps its file icon instead of firing
// a request that could only 415.
var thumbnailableExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".bmp": true, ".tif": true, ".tiff": true,
}

// isThumbnailableFile reports whether a host file can get a preview image.
func isThumbnailableFile(name string) bool {
	return thumbnailableExts[strings.ToLower(filepath.Ext(name))]
}

var (
	errHostPathInvalid = errors.New("invalid path")
	errHostPathDenied  = errors.New("path not in allowed directories")
)

// allowedHostRoots returns the configured SHARE_ALLOWED_PATHS roots.
// Each root is Clean()ed: a trailing slash in the env value ("/DATA/") would
// otherwise make both the equality check and the prefix check ("/DATA//…")
// fail and silently deny every path under that root.
func allowedHostRoots() []string {
	allowedPaths := os.Getenv("SHARE_ALLOWED_PATHS")
	if allowedPaths == "" {
		allowedPaths = "/DATA,/media,/home" // Default allowed paths for ZimaOS
	}
	roots := strings.Split(allowedPaths, ",")
	out := roots[:0]
	for _, r := range roots {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, filepath.Clean(r))
		}
	}
	return out
}

// hostPathAllowed reports whether an already symlink-resolved path sits at or
// below one of the SHARE_ALLOWED_PATHS roots. Shared by every handler that
// opens host paths (browse, thumbnail, share-from-path, share-folder) so the
// rule can't drift between them.
func hostPathAllowed(resolved string) bool {
	for _, allowed := range allowedHostRoots() {
		if resolved == allowed || strings.HasPrefix(resolved, allowed+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// resolveAllowedHostPath cleans and symlink-resolves a caller-supplied host
// path and confirms the *resolved* target sits at or below one of the allowed
// roots — resolving first is what keeps a symlink from pointing out of them.
func resolveAllowedHostPath(requested string) (string, error) {
	if requested == "" {
		return "", errHostPathInvalid
	}
	cleanPath := filepath.Clean(requested)
	if !filepath.IsAbs(cleanPath) || strings.Contains(cleanPath, "..") {
		return "", errHostPathInvalid
	}
	resolved, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		return "", errHostPathInvalid
	}
	if hostPathAllowed(resolved) {
		return resolved, nil
	}
	return "", errHostPathDenied
}

// HostThumbnail serves a cached preview for an image that already lives on the
// host filesystem, for the "share files and folders already on the server"
// browser. /thumbnail/{id} only works for shares that exist in the database, so
// the browse dialog had nothing to render previews from and showed a plain list.
//
// Admin-only, wired next to /api/browse: the same rationale as
// /api/share-from-path — a non-admin must not be able to read host files.
func (h *Handler) HostThumbnail(w http.ResponseWriter, r *http.Request) {
	if h.thumbnails == nil {
		http.Error(w, "Thumbnail service not available", http.StatusServiceUnavailable)
		return
	}

	resolved, err := resolveAllowedHostPath(r.URL.Query().Get("path"))
	switch {
	case errors.Is(err, errHostPathDenied):
		http.Error(w, "Path not in allowed directories", http.StatusForbidden)
		return
	case err != nil:
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(resolved)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if !info.Mode().IsRegular() {
		http.Error(w, "Not a regular file", http.StatusBadRequest)
		return
	}
	if !isThumbnailableFile(resolved) {
		http.Error(w, "Unsupported image type", http.StatusUnsupportedMediaType)
		return
	}

	// Cache key covers path + mtime + size: editing or replacing the file yields
	// a different key, so an edited photo can never serve its old thumbnail out
	// of data/thumbnails. Hashed because the key becomes a file name.
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%d|%d", resolved, info.ModTime().UnixNano(), info.Size()))
	key := "host-" + hex.EncodeToString(sum[:])

	thumbPath, err := h.thumbnails.GetThumbnail(key, resolved)
	if err != nil {
		// Not every "image" decodes (truncated file, wrong extension, CMYK JPEG
		// variants). Log it and answer 415 so the UI falls back to a type icon.
		log.Printf("Failed to generate host thumbnail for %s: %v", resolved, err)
		http.Error(w, "Failed to generate thumbnail", http.StatusUnsupportedMediaType)
		return
	}

	// private: this is admin-only content behind a session, it must not land in
	// a shared proxy cache. Short max-age because the URL is path-based and
	// therefore stable across edits (the cache key above is not).
	serveThumbnailFile(w, thumbPath, "private, max-age=60")
}
