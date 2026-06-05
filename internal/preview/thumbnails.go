package preview

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/image/draw"
)

// ThumbnailService generates and caches image thumbnails
type ThumbnailService struct {
	cacheDir  string
	maxWidth  int
	maxHeight int
	mu        sync.RWMutex
}

// NewThumbnailService creates a new thumbnail service
func NewThumbnailService(dataDir string) (*ThumbnailService, error) {
	cacheDir := filepath.Join(dataDir, "thumbnails")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	return &ThumbnailService{
		cacheDir:  cacheDir,
		maxWidth:  300,
		maxHeight: 300,
	}, nil
}

// GetThumbnail returns the path to a thumbnail, generating it if needed
func (s *ThumbnailService) GetThumbnail(shareID, filePath string) (string, error) {
	thumbPath := filepath.Join(s.cacheDir, shareID+".jpg")

	// Check if thumbnail already exists
	s.mu.RLock()
	if _, err := os.Stat(thumbPath); err == nil {
		s.mu.RUnlock()
		return thumbPath, nil
	}
	s.mu.RUnlock()

	// Generate thumbnail
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if _, err := os.Stat(thumbPath); err == nil {
		return thumbPath, nil
	}

	return s.generateThumbnail(filePath, thumbPath)
}

// generateThumbnail creates a thumbnail from the source image
func (s *ThumbnailService) generateThumbnail(srcPath, dstPath string) (string, error) {
	// Open source file
	file, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Guard against decompression bombs: read just the header to get the pixel
	// dimensions and refuse images that would blow up memory on a full decode
	// (a small file can declare a huge canvas). Best-effort — if the header
	// can't be parsed we fall through to the normal decode, which will error.
	const maxPixels = 50 * 1000 * 1000 // 50 megapixels
	if cfg, _, cfgErr := image.DecodeConfig(file); cfgErr == nil {
		if int64(cfg.Width)*int64(cfg.Height) > maxPixels {
			return "", fmt.Errorf("image too large to thumbnail: %dx%d", cfg.Width, cfg.Height)
		}
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	// Decode image
	var img image.Image
	ext := strings.ToLower(filepath.Ext(srcPath))

	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	case ".png":
		img, err = png.Decode(file)
	default:
		// Try to decode as any format
		img, _, err = image.Decode(file)
	}

	if err != nil {
		return "", err
	}

	// Calculate thumbnail dimensions
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	newWidth, newHeight := s.calculateDimensions(origWidth, origHeight)

	// Create thumbnail
	thumb := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.ApproxBiLinear.Scale(thumb, thumb.Bounds(), img, bounds, draw.Over, nil)

	// Save thumbnail
	out, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if err := jpeg.Encode(out, thumb, &jpeg.Options{Quality: 85}); err != nil {
		os.Remove(dstPath)
		return "", err
	}

	return dstPath, nil
}

// calculateDimensions calculates thumbnail dimensions maintaining aspect ratio
func (s *ThumbnailService) calculateDimensions(origWidth, origHeight int) (int, int) {
	if origWidth <= s.maxWidth && origHeight <= s.maxHeight {
		return origWidth, origHeight
	}

	ratio := float64(origWidth) / float64(origHeight)

	var newWidth, newHeight int
	if ratio > float64(s.maxWidth)/float64(s.maxHeight) {
		// Width is the limiting factor
		newWidth = s.maxWidth
		newHeight = int(float64(s.maxWidth) / ratio)
	} else {
		// Height is the limiting factor
		newHeight = s.maxHeight
		newWidth = int(float64(s.maxHeight) * ratio)
	}

	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	return newWidth, newHeight
}

// DeleteThumbnail removes a cached thumbnail
func (s *ThumbnailService) DeleteThumbnail(shareID string) error {
	thumbPath := filepath.Join(s.cacheDir, shareID+".jpg")
	return os.Remove(thumbPath)
}

// ClearCache removes all cached thumbnails
func (s *ThumbnailService) ClearCache() error {
	return os.RemoveAll(s.cacheDir)
}

// IsImage checks if a file is an image that can have a thumbnail
func IsImage(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/") &&
		!strings.Contains(mimeType, "svg") // SVG doesn't need thumbnails
}
