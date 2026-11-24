package handlers

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Every path that writes an upload to the data disk goes through the helpers
// in this file: the free-space preflight, the 507 answer for a full disk, and
// placing a received multipart file without holding it on disk twice.
//
// Paths (2026-11-24): chunked upload (upload.go), single upload, multi upload,
// receive-link upload (receive.go), share-from-path in copy mode (handlers.go).

// spaceShortfall reports whether the filesystem holding dir lacks room for
// size more bytes, counting what running chunked uploads still have to send
// and chunkDiskReserve. need and free are returned for the message. When free
// space cannot be measured (non-Linux), it never reports a shortfall.
func spaceShortfall(dir string, size int64) (need, free int64, short bool) {
	free, ok := freeDiskBytes(dir)
	if !ok {
		return 0, 0, false
	}
	need = size + pendingChunkBytes() + chunkDiskReserve
	return need, free, free < need
}

// refuseIfNoSpace answers 507 in plain text (admin API) and returns true when
// the data disk cannot take size more bytes.
func (h *Handler) refuseIfNoSpace(w http.ResponseWriter, what string, size int64) bool {
	need, free, short := spaceShortfall(h.storage.UploadsDir(), size)
	if !short {
		return false
	}
	log.Printf("Upload refused (%s): needs %d bytes (incl. pending + reserve), %d free", what, need, free)
	http.Error(w, fmt.Sprintf("Not enough free disk space on the server: %s needed, %s free", formatGB(need), formatGB(free)), http.StatusInsufficientStorage)
	return true
}

// multipartTempDir is where ParseMultipartForm spills file parts. It lives on
// the data disk next to the uploads so a spilled part can be hard-linked into
// place instead of copied (see placeUploadedFile). Before, parts spilled to
// the container's /tmp and were then copied, so every upload sat on disk
// twice (three times for a receive link with auto-share).
const multipartTempName = ".multipart-tmp"

// SetupMultipartTempDir points os.TempDir (which mime/multipart uses for
// spilled parts) at a directory on the data disk and removes leftovers of a
// previous process. Called once at startup, before requests are accepted.
func SetupMultipartTempDir(uploadsDir string) (string, error) {
	dir := filepath.Join(uploadsDir, multipartTempName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	entries, _ := os.ReadDir(dir)
	removed := 0
	for _, e := range entries {
		// mime/multipart names its spill files "multipart-<random>".
		if strings.HasPrefix(e.Name(), "multipart-") {
			if os.Remove(filepath.Join(dir, e.Name())) == nil {
				removed++
			}
		}
	}
	if removed > 0 {
		log.Printf("Uploads: removed %d leftover multipart temp file(s)", removed)
	}
	if err := os.Setenv("TMPDIR", dir); err != nil {
		return "", err
	}
	return dir, nil
}

// placeUploadedFile puts a received multipart file at destPath and returns it
// opened read-write, plus its size. A part that mime/multipart spilled to disk
// is an *os.File (documented on multipart.FileHeader.Open); it is hard-linked
// into place, so the bytes exist once. Anything else (a part kept in memory,
// or a link across filesystems) is copied. destPath must not exist.
func placeUploadedFile(file multipart.File, destPath string) (*os.File, int64, error) {
	if f, ok := file.(*os.File); ok {
		if err := os.Link(f.Name(), destPath); err == nil {
			os.Remove(f.Name()) // net/http's own cleanup tolerates the missing file
			if err := os.Chmod(destPath, 0644); err != nil {
				os.Remove(destPath)
				return nil, 0, err
			}
			dest, err := os.OpenFile(destPath, os.O_RDWR, 0)
			if err != nil {
				os.Remove(destPath)
				return nil, 0, err
			}
			st, err := dest.Stat()
			if err != nil {
				dest.Close()
				os.Remove(destPath)
				return nil, 0, err
			}
			return dest, st.Size(), nil
		} else {
			log.Printf("Uploads: hard link of spilled part failed, copying instead: %v", err)
		}
	}
	dest, err := createShareFile(destPath)
	if err != nil {
		return nil, 0, err
	}
	size, err := io.Copy(dest, file)
	if err != nil {
		dest.Close()
		os.Remove(destPath)
		return nil, 0, err
	}
	return dest, size, nil
}

// linkOrCopy makes dst a second name for src (no extra disk space) and falls
// back to copying when a hard link is not possible.
func linkOrCopy(src, dst string) error {
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := createShareFile(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}
	return nil
}
