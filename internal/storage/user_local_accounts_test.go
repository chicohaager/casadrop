package storage

import (
	"testing"
	"time"

	"casadrop/internal/models"
)

// TestCreateMultipleLocalUsers guards the UNIQUE(oidc_subject, oidc_issuer)
// trap: local accounts carry no OIDC identity, and writing "" into those
// columns made the *second* local account collide with the first (SQLite treats
// two NULLs as distinct but two empty strings as equal). The regression made
// every user beyond the first fail with a bare HTTP 500.
func TestCreateMultipleLocalUsers(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	local := []*models.User{
		{ID: "u1", Email: "one@example.com", Name: "One", Role: models.RoleUser,
			PasswordHash: "h1", IsActive: true, CreatedAt: time.Now().UTC()},
		{ID: "u2", Email: "two@example.com", Name: "Two", Role: models.RoleViewer,
			PasswordHash: "h2", IsActive: true, CreatedAt: time.Now().UTC()},
		{ID: "u3", Email: "three@example.com", Name: "Three", Role: models.RoleAdmin,
			PasswordHash: "h3", IsActive: true, CreatedAt: time.Now().UTC()},
	}

	for _, u := range local {
		if err := store.CreateUser(u); err != nil {
			t.Fatalf("CreateUser(%s, role=%s): %v", u.Email, u.Role, err)
		}
	}

	users, err := store.GetAllUsers()
	if err != nil {
		t.Fatalf("GetAllUsers: %v", err)
	}
	if len(users) != len(local) {
		t.Fatalf("expected %d local users, got %d", len(local), len(users))
	}

	// The OIDC columns must read back as "" even though they are stored as NULL,
	// so callers keep seeing a plain string.
	for _, u := range users {
		if u.OIDCSubject != "" || u.OIDCIssuer != "" {
			t.Errorf("%s: expected empty OIDC identity, got subject=%q issuer=%q",
				u.Email, u.OIDCSubject, u.OIDCIssuer)
		}
	}

	// Updating a local user must not resurrect the empty-string collision.
	u := local[1]
	u.Name = "Two Renamed"
	if err := store.UpdateUser(u); err != nil {
		t.Fatalf("UpdateUser(%s): %v", u.Email, err)
	}
}

// TestLookupsPreserveQuota guards a silent quota wipe: the OIDC login path loads
// a user via GetUserByOIDC/GetUserByEmail and hands that same struct straight to
// UpdateUser (to refresh last_login_at). UpdateUser persists every column, so a
// lookup that forgets to SELECT quota_bytes writes back 0 — promoting the
// account to "unlimited" on every single sign-on.
func TestLookupsPreserveQuota(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	const quota = int64(5 << 30) // 5 GiB
	sso := &models.User{ID: "q1", Email: "sso@example.com", Name: "SSO",
		Role: models.RoleUser, OIDCSubject: "sub-1", OIDCIssuer: "https://idp.example.com",
		IsActive: true, CreatedAt: time.Now().UTC(), QuotaBytes: quota}
	if err := store.CreateUser(sso); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	lookups := map[string]func() (*models.User, error){
		"GetUserByOIDC":  func() (*models.User, error) { return store.GetUserByOIDC("sub-1", "https://idp.example.com") },
		"GetUserByEmail": func() (*models.User, error) { return store.GetUserByEmail("sso@example.com") },
		"GetUser":        func() (*models.User, error) { return store.GetUser("q1") },
	}

	for name, lookup := range lookups {
		got, err := lookup()
		if err != nil || got == nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got.QuotaBytes != quota {
			t.Errorf("%s returned QuotaBytes=%d, want %d", name, got.QuotaBytes, quota)
		}

		// Simulate the login write-back and confirm the quota survives it.
		now := time.Now().UTC()
		got.LastLoginAt = &now
		if err := store.UpdateUser(got); err != nil {
			t.Fatalf("%s: UpdateUser: %v", name, err)
		}
		after, err := store.GetUser("q1")
		if err != nil || after == nil {
			t.Fatalf("%s: reload: %v", name, err)
		}
		if after.QuotaBytes != quota {
			t.Errorf("%s: quota wiped by the login write-back: got %d, want %d",
				name, after.QuotaBytes, quota)
		}
	}
}

// TestOIDCIdentityStillUnique makes sure the NULL fix did not weaken the real
// constraint: two accounts from the same issuer with the same subject must
// still be rejected, and a genuine OIDC identity must remain lookup-able.
func TestOIDCIdentityStillUnique(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	first := &models.User{ID: "o1", Email: "sso1@example.com", Name: "SSO One",
		Role: models.RoleUser, OIDCSubject: "sub-123", OIDCIssuer: "https://idp.example.com",
		IsActive: true, CreatedAt: time.Now().UTC()}
	if err := store.CreateUser(first); err != nil {
		t.Fatalf("CreateUser(first OIDC user): %v", err)
	}

	dup := &models.User{ID: "o2", Email: "sso2@example.com", Name: "SSO Dup",
		Role: models.RoleUser, OIDCSubject: "sub-123", OIDCIssuer: "https://idp.example.com",
		IsActive: true, CreatedAt: time.Now().UTC()}
	if err := store.CreateUser(dup); err == nil {
		t.Fatal("expected duplicate (subject, issuer) to be rejected, got nil error")
	}

	found, err := store.GetUserByOIDC("sub-123", "https://idp.example.com")
	if err != nil {
		t.Fatalf("GetUserByOIDC: %v", err)
	}
	if found == nil || found.ID != "o1" {
		t.Fatalf("expected to find the OIDC user o1, got %+v", found)
	}
}
