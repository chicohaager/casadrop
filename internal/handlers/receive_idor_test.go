package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"casadrop/internal/middleware"
	"casadrop/internal/models"
)

// Regression test for the receive-link IDOR fix. The three read handlers
// (GetReceiveLink, GetReceivedFiles, DownloadReceivedFile) previously guarded
// with `link.UserID != "" && link.UserID != user.ID`, so an ownerless link
// (created under the shared-admin login, UserID == "") skipped the check and
// any authenticated Viewer/User could read the admin's received files. After
// the fix, ownerless links are admin-only (matching GetShareInfo).
func TestOwnerlessReceiveLinkIsAdminOnly(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	if err := handler.storage.SaveReceiveLink(&models.ReceiveLink{
		ID:   "ownerless",
		Name: "Drop",
		// UserID intentionally empty (shared-admin owned).
	}); err != nil {
		t.Fatalf("SaveReceiveLink: %v", err)
	}
	if err := handler.storage.SaveReceivedFile(&models.ReceivedFile{
		ID:            "file1",
		ReceiveLinkID: "ownerless",
		FileName:      "secret.pdf",
		OriginalName:  "secret.pdf",
		FileSize:      10,
	}); err != nil {
		t.Fatalf("SaveReceivedFile: %v", err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/receive-links/{id}", handler.GetReceiveLink).Methods("GET")
	router.HandleFunc("/api/receive-links/{id}/files", handler.GetReceivedFiles).Methods("GET")

	do := func(path string, user *middleware.SessionUser) int {
		req := httptest.NewRequest("GET", path, nil)
		if user != nil {
			req = req.WithContext(middleware.ContextWithUser(req.Context(), user))
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	nonAdmin := &middleware.SessionUser{ID: "intruder", Role: models.RoleUser}
	admin := &middleware.SessionUser{ID: "", Role: models.RoleAdmin}

	for _, path := range []string{"/api/receive-links/ownerless", "/api/receive-links/ownerless/files"} {
		if code := do(path, nonAdmin); code != http.StatusForbidden {
			t.Errorf("non-admin GET %s: got %d, want 403", path, code)
		}
		if code := do(path, admin); code != http.StatusOK {
			t.Errorf("admin GET %s: got %d, want 200", path, code)
		}
	}
}
