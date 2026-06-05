// Package totp implements RFC 6238 time-based one-time passwords (SHA-1, 6
// digits, 30s period) with zero external dependencies — used for optional
// two-factor authentication on the local admin login.
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	digits = 6
	period = 30 // seconds
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateSecret returns a new random base32-encoded TOTP secret (160 bits).
func GenerateSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return b32.EncodeToString(b), nil
}

// ProvisioningURI builds the otpauth:// URI for authenticator-app enrollment.
func ProvisioningURI(secret, account, issuer string) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(digits))
	q.Set("period", fmt.Sprint(period))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// GenerateCode returns the TOTP code for the current 30s step. Useful for
// tooling and tests; the login path uses Validate / ValidateWithCounter.
func GenerateCode(secret string) (string, error) {
	return codeAt(secret, uint64(time.Now().Unix())/period)
}

func codeAt(secret string, counter uint64) (string, error) {
	key, err := b32.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", err
	}
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	h := hmac.New(sha1.New, key)
	h.Write(buf[:])
	sum := h.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	val := (uint32(sum[offset]&0x7f) << 24) |
		(uint32(sum[offset+1]) << 16) |
		(uint32(sum[offset+2]) << 8) |
		uint32(sum[offset+3])
	return fmt.Sprintf("%0*d", digits, val%1_000_000), nil
}

// Validate reports whether code is a valid TOTP for secret, accepting the
// current 30s step plus one step of clock skew on either side. The comparison
// is constant-time.
func Validate(secret, code string) bool {
	_, ok := ValidateWithCounter(secret, code)
	return ok
}

// ValidateWithCounter is like Validate but also returns the step counter that
// matched. Callers can persist the last consumed counter and reject any code
// whose counter is <= the stored value, giving single-use (anti-replay)
// semantics within the 90s acceptance window (RFC 6238 §5.2).
func ValidateWithCounter(secret, code string) (uint64, bool) {
	code = strings.TrimSpace(code)
	if len(code) != digits {
		return 0, false
	}
	counter := uint64(time.Now().Unix()) / period
	// Try the current step first so a correct "now" code is preferred over an
	// adjacent step, minimising spurious replay rejections of the next code.
	for _, c := range []uint64{counter, counter - 1, counter + 1} {
		want, err := codeAt(secret, c)
		if err != nil {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return c, true
		}
	}
	return 0, false
}
