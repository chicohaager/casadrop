package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"casadrop/internal/auth"
	"casadrop/internal/middleware"
	"casadrop/internal/models"
)

// UserResponse represents a user in API responses (without sensitive data)
type UserResponse struct {
	ID          string      `json:"id"`
	Email       string      `json:"email"`
	Name        string      `json:"name"`
	Role        models.Role `json:"role"`
	IsActive    bool        `json:"isActive"`
	IsOIDCUser  bool        `json:"isOidcUser"`
	CreatedAt   time.Time   `json:"createdAt"`
	LastLoginAt *time.Time  `json:"lastLoginAt,omitempty"`
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Email    string      `json:"email"`
	Name     string      `json:"name"`
	Role     models.Role `json:"role"`
	Password string      `json:"password,omitempty"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	Email    string      `json:"email,omitempty"`
	Name     string      `json:"name,omitempty"`
	Role     models.Role `json:"role,omitempty"`
	Password string      `json:"password,omitempty"`
	IsActive *bool       `json:"isActive,omitempty"`
}

// toResponse converts a User to UserResponse
func toUserResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		Name:        user.Name,
		Role:        user.Role,
		IsActive:    user.IsActive,
		IsOIDCUser:  user.IsOIDCUser(),
		CreatedAt:   user.CreatedAt,
		LastLoginAt: user.LastLoginAt,
	}
}

// ListUsers returns all users (admin only)
// GET /api/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.storage.GetAllUsers()
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = toUserResponse(user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// CreateUser creates a new user (admin only)
// POST /api/users
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Email == "" {
		jsonError(w, "Email is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		jsonError(w, "Name is required", http.StatusBadRequest)
		return
	}
	if !models.IsValidRole(req.Role) {
		jsonError(w, "Invalid role", http.StatusBadRequest)
		return
	}

	// Check if email already exists
	existing, _ := h.storage.GetUserByEmail(req.Email)
	if existing != nil {
		jsonError(w, "Email already exists", http.StatusConflict)
		return
	}

	// Hash password if provided
	var passwordHash string
	if req.Password != "" {
		if len(req.Password) < 8 {
			jsonError(w, "Password must be at least 8 characters", http.StatusBadRequest)
			return
		}
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		passwordHash = hash
	}

	// Create user
	user := &models.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		Name:         req.Name,
		Role:         req.Role,
		PasswordHash: passwordHash,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
	}

	if err := h.storage.CreateUser(user); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toUserResponse(user))
}

// GetUser returns a specific user (admin only)
// GET /api/users/{id}
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	user, err := h.storage.GetUser(id)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toUserResponse(user))
}

// UpdateUser updates a user (admin only)
// PUT /api/users/{id}
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	user, err := h.storage.GetUser(id)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update fields if provided
	if req.Email != "" && req.Email != user.Email {
		// Check if new email already exists
		existing, _ := h.storage.GetUserByEmail(req.Email)
		if existing != nil && existing.ID != id {
			jsonError(w, "Email already exists", http.StatusConflict)
			return
		}
		user.Email = req.Email
	}

	if req.Name != "" {
		user.Name = req.Name
	}

	if req.Role != "" {
		if !models.IsValidRole(req.Role) {
			jsonError(w, "Invalid role", http.StatusBadRequest)
			return
		}
		user.Role = req.Role
	}

	if req.Password != "" {
		if len(req.Password) < 8 {
			jsonError(w, "Password must be at least 8 characters", http.StatusBadRequest)
			return
		}
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		user.PasswordHash = hash
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := h.storage.UpdateUser(user); err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toUserResponse(user))
}

// DeleteUser deletes a user (admin only)
// DELETE /api/users/{id}
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Prevent self-deletion
	currentUser := middleware.GetUserFromContext(r.Context())
	if currentUser != nil && currentUser.ID == id {
		jsonError(w, "Cannot delete your own account", http.StatusForbidden)
		return
	}

	user, err := h.storage.GetUser(id)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if err := h.storage.DeleteUser(id); err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetCurrentUser returns the current user's profile
// GET /api/me
func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := middleware.GetUserFromContext(r.Context())
	if sessionUser == nil {
		jsonError(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Try to get user from database
	var user *models.User
	var err error
	if sessionUser.ID != "" {
		user, err = h.storage.GetUser(sessionUser.ID)
	}

	// For legacy sessions or if user not found, return session info
	if err != nil || user == nil {
		// Use session role (defaults to admin for legacy sessions)
		role := sessionUser.Role
		if role == "" {
			role = models.RoleAdmin
		}
		email := sessionUser.Email
		if email == "" {
			email = "admin@localhost"
		}
		response := UserResponse{
			ID:       sessionUser.ID,
			Email:    email,
			Name:     "Admin",
			Role:     role,
			IsActive: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toUserResponse(user))
}

// UpdateCurrentUser updates the current user's profile
// PUT /api/me
func (h *Handler) UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := middleware.GetUserFromContext(r.Context())
	if sessionUser == nil {
		jsonError(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	user, err := h.storage.GetUser(sessionUser.ID)
	if err != nil || user == nil {
		jsonError(w, "User not found", http.StatusNotFound)
		return
	}

	var req struct {
		Name     string `json:"name,omitempty"`
		Password string `json:"password,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Users can only update their name and password (not role or email)
	if req.Name != "" {
		user.Name = req.Name
	}

	if req.Password != "" {
		if len(req.Password) < 8 {
			jsonError(w, "Password must be at least 8 characters", http.StatusBadRequest)
			return
		}
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		user.PasswordHash = hash
	}

	if err := h.storage.UpdateUser(user); err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toUserResponse(user))
}

// jsonError sends a JSON error response
func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
