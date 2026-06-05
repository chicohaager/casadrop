package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"casadrop/internal/middleware"
	"casadrop/internal/models"
)

// Regression test for the IDOR fix: an ownerless share (UserID == "", e.g. one
// created under the shared-admin login or via receive-link auto-share) must NOT
// be accessible to a non-admin user. Before the fix the guard was
// `share.UserID != "" && share.UserID != user.ID`, so an empty owner matched
// everyone.
func TestOwnerlessShareIsAdminOnly(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	handler.storage.Save(&models.Share{
		ID:           "ownerless",
		FileName:     "f.txt",
		OriginalName: "f.txt",
		FileSize:     1,
		ExpiresAt:    time.Now().Add(time.Hour),
		CreatedAt:    time.Now(),
		// UserID intentionally empty.
	})

	router := mux.NewRouter()
	router.HandleFunc("/api/shares/{id}", handler.GetShareInfo).Methods("GET")
	router.HandleFunc("/api/shares/{id}", handler.DeleteShare).Methods("DELETE")

	do := func(method string, user *middleware.SessionUser) int {
		req := httptest.NewRequest(method, "/api/shares/ownerless", nil)
		if user != nil {
			req = req.WithContext(middleware.ContextWithUser(req.Context(), user))
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	nonAdmin := &middleware.SessionUser{ID: "intruder", Role: models.RoleUser}
	admin := &middleware.SessionUser{ID: "", Role: models.RoleAdmin}

	if code := do("GET", nonAdmin); code != http.StatusForbidden {
		t.Errorf("non-admin GET ownerless share: got %d, want 403", code)
	}
	if code := do("GET", admin); code != http.StatusOK {
		t.Errorf("admin GET ownerless share: got %d, want 200", code)
	}
	// Non-admin must not be able to delete it either.
	if code := do("DELETE", nonAdmin); code != http.StatusForbidden {
		t.Errorf("non-admin DELETE ownerless share: got %d, want 403", code)
	}
	// Admin can delete.
	if code := do("DELETE", admin); code != http.StatusNoContent {
		t.Errorf("admin DELETE ownerless share: got %d, want 204", code)
	}
}
