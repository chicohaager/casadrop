package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"casadrop/internal/models"
)

// makeSession creates a session for the given identity and returns its raw
// token and the opaque session ID the API exposes.
func makeSession(t *testing.T, aa *AdminAuth, userID, email string, role models.Role) (token, id string) {
	t.Helper()
	tok, err := aa.CreateSessionForUser("192.0.2.1", "test-agent", userID, email, role)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	aa.mu.RLock()
	defer aa.mu.RUnlock()
	s := aa.sessions[hashToken(tok)]
	if s.ID == "" {
		t.Fatal("session created without an ID")
	}
	return tok, s.ID
}

func listSessions(t *testing.T, aa *AdminAuth, caller *SessionUser, currentToken string) []SessionInfo {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/sessions", nil)
	if currentToken != "" {
		req.AddCookie(&http.Cookie{Name: "casadrop_session", Value: currentToken})
	}
	if caller != nil {
		req = req.WithContext(ContextWithUser(req.Context(), caller))
	}
	rr := httptest.NewRecorder()
	aa.ListSessionsHandler(rr, req)
	if rr.Code != 200 {
		t.Fatalf("list sessions: %d", rr.Code)
	}
	var resp struct {
		Sessions []SessionInfo `json:"sessions"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp.Sessions
}

func TestListSessionsMarksCurrentAndScopesByUser(t *testing.T) {
	aa := NewAdminAuth("", t.TempDir())
	t.Cleanup(aa.Stop)

	aliceTok, _ := makeSession(t, aa, "alice", "alice@example.com", models.RoleUser)
	makeSession(t, aa, "alice", "alice@example.com", models.RoleUser) // her second device
	makeSession(t, aa, "bob", "bob@example.com", models.RoleUser)

	alice := &SessionUser{ID: "alice", Email: "alice@example.com", Role: models.RoleUser}
	got := listSessions(t, aa, alice, aliceTok)

	if len(got) != 2 {
		t.Fatalf("alice should see her 2 sessions, not bob's; got %d", len(got))
	}
	current := 0
	for _, s := range got {
		if s.Current {
			current++
		}
		if s.UserEmail != "" {
			t.Error("a non-admin listing must not carry user emails")
		}
	}
	if current != 1 {
		t.Errorf("expected exactly one current session, got %d", current)
	}
}

func TestAdminSeesAllSessions(t *testing.T) {
	aa := NewAdminAuth("", t.TempDir())
	t.Cleanup(aa.Stop)
	makeSession(t, aa, "alice", "alice@example.com", models.RoleUser)
	makeSession(t, aa, "bob", "bob@example.com", models.RoleUser)
	adminTok, _ := makeSession(t, aa, "root", "root@example.com", models.RoleAdmin)

	got := listSessions(t, aa, &SessionUser{ID: "root", Email: "root@example.com", Role: models.RoleAdmin}, adminTok)
	if len(got) != 3 {
		t.Fatalf("admin should see all 3 sessions, got %d", len(got))
	}
	// The admin view carries emails so a stranger can be recognised.
	emails := 0
	for _, s := range got {
		if s.UserEmail != "" {
			emails++
		}
	}
	if emails != 3 {
		t.Errorf("admin view should include emails, got %d of 3", emails)
	}
}

func revoke(t *testing.T, aa *AdminAuth, caller *SessionUser, id string) int {
	t.Helper()
	req := httptest.NewRequest("DELETE", "/api/sessions/"+id, nil)
	req = req.WithContext(ContextWithUser(req.Context(), caller))
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/sessions/{id}", aa.RevokeSessionHandler).Methods("DELETE")
	router.ServeHTTP(rr, req)
	return rr.Code
}

func TestRevokeSessionOwnershipRules(t *testing.T) {
	aa := NewAdminAuth("", t.TempDir())
	t.Cleanup(aa.Stop)

	bobTok, bobID := makeSession(t, aa, "bob", "bob@example.com", models.RoleUser)
	alice := &SessionUser{ID: "alice", Email: "alice@example.com", Role: models.RoleUser}

	// Alice cannot revoke Bob's session, and the refusal is a 404 (no leak).
	if code := revoke(t, aa, alice, bobID); code != http.StatusNotFound {
		t.Errorf("alice revoking bob: %d, want 404", code)
	}
	if aa.getSession(bobTok) == nil {
		t.Error("bob's session must survive alice's attempt")
	}

	// Bob can revoke his own.
	bob := &SessionUser{ID: "bob", Email: "bob@example.com", Role: models.RoleUser}
	if code := revoke(t, aa, bob, bobID); code != http.StatusNoContent {
		t.Errorf("bob revoking his own: %d, want 204", code)
	}
	if aa.getSession(bobTok) != nil {
		t.Error("bob's session should be gone after he revoked it")
	}
}

func TestAdminCanRevokeAnySession(t *testing.T) {
	aa := NewAdminAuth("", t.TempDir())
	t.Cleanup(aa.Stop)
	bobTok, bobID := makeSession(t, aa, "bob", "bob@example.com", models.RoleUser)
	admin := &SessionUser{ID: "root", Role: models.RoleAdmin}

	if code := revoke(t, aa, admin, bobID); code != http.StatusNoContent {
		t.Errorf("admin revoking bob: %d, want 204", code)
	}
	if aa.getSession(bobTok) != nil {
		t.Error("admin should have been able to end bob's session")
	}
	// An unknown id is still 404, admin or not.
	if code := revoke(t, aa, admin, "no-such-session"); code != http.StatusNotFound {
		t.Errorf("admin revoking unknown: %d, want 404", code)
	}
}

func TestRevokeOtherSessionsSparesCurrent(t *testing.T) {
	aa := NewAdminAuth("", t.TempDir())
	t.Cleanup(aa.Stop)

	aliceTok1, _ := makeSession(t, aa, "alice", "alice@example.com", models.RoleUser)
	aliceTok2, _ := makeSession(t, aa, "alice", "alice@example.com", models.RoleUser)
	aliceTok3, _ := makeSession(t, aa, "alice", "alice@example.com", models.RoleUser)
	bobTok, _ := makeSession(t, aa, "bob", "bob@example.com", models.RoleUser)

	req := httptest.NewRequest("POST", "/api/sessions/revoke-others", nil)
	req.AddCookie(&http.Cookie{Name: "casadrop_session", Value: aliceTok1})
	req = req.WithContext(ContextWithUser(req.Context(), &SessionUser{ID: "alice", Role: models.RoleUser}))
	rr := httptest.NewRecorder()
	aa.RevokeOtherSessionsHandler(rr, req)
	if rr.Code != 200 {
		t.Fatalf("revoke-others: %d", rr.Code)
	}
	var resp struct {
		Revoked int `json:"revoked"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Revoked != 2 {
		t.Errorf("expected 2 of alice's other sessions revoked, got %d", resp.Revoked)
	}

	if aa.getSession(aliceTok1) == nil {
		t.Error("the current session must be spared")
	}
	if aa.getSession(aliceTok2) != nil || aa.getSession(aliceTok3) != nil {
		t.Error("alice's other sessions should be gone")
	}
	if aa.getSession(bobTok) == nil {
		t.Error("bob's session must never be touched by alice's revoke-others")
	}
}
