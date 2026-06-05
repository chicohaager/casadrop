package middleware

import (
	"testing"

	"casadrop/internal/totp"
)

// TestVerifyTOTP_AntiReplay verifies that a valid 2FA code is single-use within
// its acceptance window (regression guard for the anti-replay fix).
func TestVerifyTOTP_AntiReplay(t *testing.T) {
	aa := NewAdminAuth("pw", t.TempDir())
	defer aa.Stop()

	secret, err := totp.GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	aa.config = &AdminConfig{PasswordHash: "x", SetupDone: true, TOTPEnabled: true, TOTPSecret: secret}

	code, err := totp.GenerateCode(secret)
	if err != nil {
		t.Fatal(err)
	}

	if !aa.verifyTOTP(code) {
		t.Fatal("first use of a valid code must succeed")
	}
	if aa.verifyTOTP(code) {
		t.Fatal("replay of the same code must be rejected")
	}
	if aa.verifyTOTP("000000") {
		t.Fatal("an invalid code must be rejected")
	}
}

// TestVerifyTOTP_DisabledPasses confirms verifyTOTP is a no-op (returns true)
// when 2FA is not configured, so the gate doesn't block password-only logins.
func TestVerifyTOTP_DisabledPasses(t *testing.T) {
	aa := NewAdminAuth("pw", t.TempDir())
	defer aa.Stop()
	aa.config = &AdminConfig{PasswordHash: "x", SetupDone: true}
	if !aa.verifyTOTP("") {
		t.Fatal("verifyTOTP must pass when 2FA is disabled")
	}
}
