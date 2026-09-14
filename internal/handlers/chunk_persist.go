package handlers

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// manifestName is the file inside a chunk upload's temp directory that records
// what the upload is and which chunks have arrived. It is what lets an upload
// survive a browser reload or a server restart: the registry is rebuilt from
// these on startup, and the client asks "which chunks do you already have?"
// before resuming.
const manifestName = "manifest.json"

// chunkManifest is the on-disk form of a ChunkUpload. It holds only what is
// needed to resume — the transient maps are rebuilt from the chunk files on
// disk, which are the source of truth for what actually landed.
type chunkManifest struct {
	ID          string `json:"id"`
	FileName    string `json:"file_name"`
	TotalSize   int64  `json:"total_size"`
	TotalChunks int    `json:"total_chunks"`
	UserID      string `json:"user_id,omitempty"`
	CreatedAt   int64  `json:"created_at_unix"`
}

// writeManifest persists the upload's immutable facts once, at init. The chunk
// files themselves record progress, so the manifest never needs rewriting as
// chunks arrive — one write per upload, not one per chunk.
func writeManifest(upload *ChunkUpload) error {
	m := chunkManifest{
		ID:          upload.ID,
		FileName:    upload.FileName,
		TotalSize:   upload.TotalSize,
		TotalChunks: upload.TotalChunks,
		UserID:      upload.UserID,
		CreatedAt:   upload.CreatedAt.Unix(),
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	// 0600: the manifest names a file a user is uploading; it is not world-readable.
	return os.WriteFile(filepath.Join(upload.TempDir, manifestName), data, 0600)
}

// RestoreChunkUploads rebuilds the in-memory registry from any manifests left
// on disk by a previous process, so an upload interrupted by a restart can be
// resumed instead of silently vanishing. Chunk progress is read back from the
// chunk files, which cannot lie about what was flushed. Called once at startup,
// before the server accepts requests.
func RestoreChunkUploads(uploadsDir string) {
	chunksRoot := filepath.Join(uploadsDir, "chunks")
	entries, err := os.ReadDir(chunksRoot)
	if err != nil {
		return // no chunks directory yet is the normal fresh-start case
	}

	restored := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tempDir := filepath.Join(chunksRoot, entry.Name())
		upload, err := loadUploadFromDisk(tempDir)
		if err != nil {
			// A directory without a readable manifest is debris from a version
			// that predated persistence, or a half-written init. Leave it for
			// the age-based cleanup rather than guessing.
			continue
		}
		chunkUploadsMu.Lock()
		chunkUploads[upload.ID] = upload
		chunkUploadsMu.Unlock()
		restored++
	}
	if restored > 0 {
		log.Printf("Chunked uploads: restored %d interrupted upload(s) from disk", restored)
		startChunkCleanupWorker()
	}
}

// loadUploadFromDisk reconstructs one ChunkUpload from its temp directory: the
// manifest for the facts, the chunk files for the progress.
func loadUploadFromDisk(tempDir string) (*ChunkUpload, error) {
	data, err := os.ReadFile(filepath.Join(tempDir, manifestName))
	if err != nil {
		return nil, err
	}
	var m chunkManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	upload := &ChunkUpload{
		ID:             m.ID,
		FileName:       m.FileName,
		TotalSize:      m.TotalSize,
		TotalChunks:    m.TotalChunks,
		UserID:         m.UserID,
		ChunksReceived: make(map[int]bool),
		ChunkSizes:     make(map[int]int64),
		TempDir:        tempDir,
		CreatedAt:      time.Unix(m.CreatedAt, 0),
	}

	// The chunk files are the truth about what actually flushed. A chunk index
	// beyond TotalChunks (a tampered or corrupt name) is ignored.
	files, _ := os.ReadDir(tempDir)
	for _, f := range files {
		if f.IsDir() || !strings.HasPrefix(f.Name(), "chunk_") {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimPrefix(f.Name(), "chunk_"))
		if err != nil || idx < 0 || idx >= m.TotalChunks {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		upload.ChunksReceived[idx] = true
		upload.ChunkSizes[idx] = info.Size()
		upload.ReceivedBytes += info.Size()
	}
	return upload, nil
}

// receivedIndices returns the sorted chunk indices already on disk for an
// upload, for the resume-status response. Caller holds no lock; it reads a
// snapshot copy so it must be called with the registry lock held or on a copy.
func receivedIndices(upload *ChunkUpload) []int {
	out := make([]int, 0, len(upload.ChunksReceived))
	for idx := range upload.ChunksReceived {
		out = append(out, idx)
	}
	sortInts(out)
	return out
}

// sortInts is a tiny insertion sort — the slice is a set of chunk indices,
// small and nearly ordered, so pulling in sort for it is not worth the import.
func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}
