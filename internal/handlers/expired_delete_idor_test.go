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

// Regression test: deleting an EXPIRED share must still honor ownership. The old
// DeleteShare fell through to an unconditional storage.Delete when Get returned
// !ok (which it does for expired rows), so any authenticated user — including a
// viewer who can't own shares at all — could delete another user's expired share
// (and its file) by ID. The fix returns 404 instead and leaves reaping to the
// background cleanup.
func TestDeleteExpiredShareRequiresOwnership(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	// A share owned by "victim" that has already expired.
	handler.storage.Save(&models.Share{
		ID:           "expired1",
		FileName:     "f.txt",
		OriginalName: "f.txt",
		FileSize:     1,
		UserID:       "victim",
		ExpiresAt:    time.Now().Add(-time.Hour),
		CreatedAt:    time.Now().Add(-2 * time.Hour),
	})

	router := mux.NewRouter()
	router.HandleFunc("/api/shares/{id}", handler.DeleteShare).Methods("DELETE")

	del := func(user *middleware.SessionUser) int {
		req := httptest.NewRequest("DELETE", "/api/shares/expired1", nil)
		req = req.WithContext(middleware.ContextWithUser(req.Context(), user))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	// An unrelated non-admin must NOT be able to delete the victim's expired
	// share. Before the fix this returned 204; now it's a clean 404.
	if code := del(&middleware.SessionUser{ID: "intruder", Role: models.RoleUser}); code != http.StatusNotFound {
		t.Errorf("intruder DELETE expired share: got %d, want 404", code)
	}
	// A viewer (cannot own shares) likewise gets 404, not a silent delete.
	if code := del(&middleware.SessionUser{ID: "peeker", Role: models.RoleViewer}); code != http.StatusNotFound {
		t.Errorf("viewer DELETE expired share: got %d, want 404", code)
	}
}
