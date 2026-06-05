package totp

import (
	"testing"
	"time"
)

// RFC 6238 test vector (SHA-1, secret "12345678901234567890").
func TestCodeAt_RFC6238(t *testing.T) {
	secret := b32.EncodeToString([]byte("12345678901234567890"))
	// T = 59s → counter 1 → known code 287082 (RFC 6238 appendix B).
	got, err := codeAt(secret, 59/period)
	if err != nil {
		t.Fatal(err)
	}
	if got != "287082" {
		t.Fatalf("RFC6238 vector: got %s, want 287082", got)
	}
}

func TestValidate_Roundtrip(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	counter := uint64(time.Now().Unix()) / period
	code, _ := codeAt(secret, counter)
	if !Validate(secret, code) {
		t.Fatal("valid current code rejected")
	}
	if Validate(secret, "12345") { // wrong length
		t.Fatal("short code accepted")
	}
	// A code from far in the past must be outside the ±1 window.
	old, _ := codeAt(secret, counter-100)
	if old != code && Validate(secret, old) {
		t.Fatal("stale code outside skew window accepted")
	}
}
