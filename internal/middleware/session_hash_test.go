package middleware

import (
	"os"
	"strings"
	"testing"
)

// Verifies session tokens are stored hashed: the raw bearer token validates, but
// it never appears in the in-memory map keys or in sessions.json on disk.
func TestSessionTokenHashedAtRest(t *testing.T) {
	aa, cleanup := setupTestAdminAuth(t)
	defer cleanup()

	raw, err := aa.CreateSession("1.2.3.4", "ua")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Raw token must validate (lookup hashes it internally).
	if aa.getSession(raw) == nil {
		t.Fatal("getSession(raw) returned nil — hashed lookup broken")
	}
	if !aa.validSession(raw) {
		t.Fatal("validSession(raw) false")
	}

	// The map must be keyed by the hash, not the raw token.
	if _, ok := aa.sessions[raw]; ok {
		t.Fatal("session map is keyed by the RAW token — must be the hash")
	}
	if _, ok := aa.sessions[hashToken(raw)]; !ok {
		t.Fatal("session map is not keyed by the token hash")
	}

	// sessions.json must not contain the raw token.
	data, err := os.ReadFile(aa.sessionsPath())
	if err != nil {
		t.Fatalf("read sessions file: %v", err)
	}
	if strings.Contains(string(data), raw) {
		t.Fatal("sessions.json contains the RAW token — must persist only the hash")
	}
	if !strings.Contains(string(data), hashToken(raw)) {
		t.Fatal("sessions.json does not contain the token hash")
	}

	// Logout invalidates by raw token.
	aa.InvalidateSession(raw)
	if aa.getSession(raw) != nil {
		t.Fatal("session still valid after InvalidateSession")
	}
}
