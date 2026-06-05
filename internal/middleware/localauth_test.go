package middleware

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"casadrop/internal/models"
	"casadrop/internal/totp"
)

type fakeUserStore struct{ users map[string]*models.User }

func (f *fakeUserStore) GetUserByEmail(email string) (*models.User, error) {
	return f.users[email], nil // (nil, nil) for unknown — mirrors real storage
}

func mkUser(t *testing.T, id, email string, role models.Role, password string, active bool) *models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return &models.User{ID: id, Email: email, Role: role, PasswordHash: string(hash), IsActive: active}
}

func newAuthWithUsers(t *testing.T, envPassword string, users ...*models.User) *AdminAuth {
	t.Helper()
	aa := NewAdminAuth(envPassword, t.TempDir())
	t.Cleanup(aa.Stop)
	store := &fakeUserStore{users: map[string]*models.User{}}
	for _, u := range users {
		store.users[u.Email] = u
	}
	aa.SetLocalUserStore(store)
	return aa
}

func TestResolveLogin_PerUserRole(t *testing.T) {
	viewer := mkUser(t, "u1", "viewer@example.com", models.RoleViewer, "correct-horse", true)
	aa := newAuthWithUsers(t, "adminpw", viewer)

	id, email, role, ok := aa.resolveLogin("viewer@example.com", "correct-horse")
	if !ok {
		t.Fatal("valid per-user credentials were rejected")
	}
	if id != "u1" || email != "viewer@example.com" || role != models.RoleViewer {
		t.Fatalf("wrong session identity: id=%q email=%q role=%q", id, email, role)
	}
}

func TestResolveLogin_WrongPasswordFails(t *testing.T) {
	user := mkUser(t, "u1", "user@example.com", models.RoleUser, "right", true)
	aa := newAuthWithUsers(t, "", user)

	if _, _, _, ok := aa.resolveLogin("user@example.com", "wrong"); ok {
		t.Fatal("wrong password must not authenticate")
	}
}

func TestResolveLogin_InactiveUserFails(t *testing.T) {
	user := mkUser(t, "u1", "ghost@example.com", models.RoleUser, "secret", false)
	aa := newAuthWithUsers(t, "", user)

	if _, _, _, ok := aa.resolveLogin("ghost@example.com", "secret"); ok {
		t.Fatal("inactive user must not authenticate")
	}
}

func TestResolveLogin_AdminPasswordFallback(t *testing.T) {
	aa := newAuthWithUsers(t, "adminpw") // no users

	// Blank email + admin password → admin session (backward compatible).
	_, _, role, ok := aa.resolveLogin("", "adminpw")
	if !ok || role != models.RoleAdmin {
		t.Fatalf("admin password fallback failed: ok=%v role=%q", ok, role)
	}

	if _, _, _, ok := aa.resolveLogin("", "nope"); ok {
		t.Fatal("wrong admin password must not authenticate")
	}
}

// A non-existent email must not fall through and silently mint an admin session
// just because the typed password happens not to match anything.
func TestResolveLogin_UnknownEmailNoSession(t *testing.T) {
	aa := newAuthWithUsers(t, "adminpw")
	if _, _, _, ok := aa.resolveLogin("nobody@example.com", "not-the-admin-pw"); ok {
		t.Fatal("unknown email with wrong password must be rejected")
	}
}

func TestAdminTOTPGate(t *testing.T) {
	aa := NewAdminAuth("adminpw", t.TempDir())
	t.Cleanup(aa.Stop)

	// Disabled by default → the 2FA gate must pass regardless of code.
	if aa.IsTOTPEnabled() {
		t.Fatal("2FA should be off by default")
	}
	if !aa.verifyTOTP("") {
		t.Fatal("gate must pass when 2FA is disabled")
	}

	// Enable with a real secret; a malformed code must now be rejected.
	secret, err := totpGenSecret(t)
	if err != nil {
		t.Fatal(err)
	}
	aa.config.TOTPSecret = secret
	aa.config.TOTPEnabled = true
	if !aa.IsTOTPEnabled() {
		t.Fatal("2FA should be enabled")
	}
	if aa.verifyTOTP("12345") {
		t.Fatal("malformed code must be rejected when 2FA is enabled")
	}
}

func totpGenSecret(t *testing.T) (string, error) { t.Helper(); return totp.GenerateSecret() }
