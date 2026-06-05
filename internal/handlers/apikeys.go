package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"casadrop/internal/models"
)

// ListAPIKeys returns all API keys
func (h *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.storage.ListAPIKeys()
	if err != nil {
		http.Error(w, "Failed to list API keys", http.StatusInternalServerError)
		return
	}
	if keys == nil {
		keys = []map[string]interface{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

// CreateAPIKey generates a new API key
func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = "API Key"
	}
	if req.Role == "" {
		req.Role = string(models.RoleAdmin)
	}
	// Reject arbitrary role strings — an unvalidated role is stored verbatim
	// and later compared in RequireRole, so only known roles are allowed.
	if !models.IsValidRole(models.Role(req.Role)) {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	// Generate random key: cdp_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}
	keyStr := "cdp_" + hex.EncodeToString(rawKey)
	prefix := keyStr[:12] + "..."

	// Hash key for storage
	hash := sha256.Sum256([]byte(keyStr))
	keyHash := hex.EncodeToString(hash[:])

	id := uuid.New().String()[:8]

	if err := h.storage.CreateAPIKey(id, req.Name, keyHash, prefix, "", req.Role); err != nil {
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     id,
		"name":   req.Name,
		"key":    keyStr, // Only shown once!
		"prefix": prefix,
		"role":   req.Role,
	})
}

// DeleteAPIKey removes an API key
func (h *Handler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := h.storage.DeleteAPIKey(id); err != nil {
		http.Error(w, "Failed to delete API key", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
