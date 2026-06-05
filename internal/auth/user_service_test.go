package auth

import (
	"testing"

	"casadrop/internal/models"
	"casadrop/internal/storage"
)

// newTestUserService builds a UserService backed by a real on-disk SQLite store
// in a temp dir, with auto-provisioning enabled and the default viewer role.
func newTestUserService(t *testing.T) *UserService {
	t.Helper()
	store, err := storage.NewSQLiteStorage(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return &UserService{
		storage:       store,
		defaultRole:   models.RoleViewer,
		autoProvision: true,
		provBuckets:   make(map[string]*provisionBucket),
	}
}

// An OIDC identity asserting an UNVERIFIED email must never be linked to an
// existing account with that email — that would be account takeover.
func TestFindOrCreateOIDCUser_UnverifiedEmailDoesNotLink(t *testing.T) {
	svc := newTestUserService(t)

	existing := &models.User{
		ID:       "admin-1",
		Email:    "admin@example.com",
		Name:     "Admin",
		Role:     models.RoleAdmin,
		IsActive: true,
	}
	if err := svc.storage.CreateUser(existing); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	attacker := &UserInfo{
		Subject:       "attacker-sub",
		Issuer:        "https://idp.example.com",
		Email:         "admin@example.com",
		EmailVerified: false,
		Name:          "Mallory",
	}

	user, err := svc.FindOrCreateOIDCUser(attacker)
	if err != nil {
		t.Fatalf("FindOrCreateOIDCUser: %v", err)
	}
	// Fail-closed: an unverified email must neither link to the existing admin
	// (takeover) nor provision a new account; the login is rejected.
	if user != nil {
		t.Fatalf("unverified email must not authenticate, got user %q (role %q)", user.ID, user.Role)
	}

	// The existing admin account must be untouched (no OIDC subject grafted on).
	reloaded, err := svc.storage.GetUserByEmail("admin@example.com")
	if err != nil {
		t.Fatalf("reload admin: %v", err)
	}
	if reloaded.OIDCSubject != "" {
		t.Fatalf("attacker OIDC subject was linked to admin account: %q", reloaded.OIDCSubject)
	}
}

// A verified email may link to an existing account (expected SSO behaviour).
func TestFindOrCreateOIDCUser_VerifiedEmailLinks(t *testing.T) {
	svc := newTestUserService(t)

	existing := &models.User{
		ID:       "user-1",
		Email:    "person@example.com",
		Name:     "Person",
		Role:     models.RoleUser,
		IsActive: true,
	}
	if err := svc.storage.CreateUser(existing); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	info := &UserInfo{
		Subject:       "person-sub",
		Issuer:        "https://idp.example.com",
		Email:         "person@example.com",
		EmailVerified: true,
	}

	user, err := svc.FindOrCreateOIDCUser(info)
	if err != nil {
		t.Fatalf("FindOrCreateOIDCUser: %v", err)
	}
	if user == nil || user.ID != existing.ID {
		t.Fatal("verified email should link to the existing account")
	}
	if user.OIDCSubject != "person-sub" {
		t.Fatalf("expected OIDC subject to be linked, got %q", user.OIDCSubject)
	}
}
