// Package pow implements a dependency-free, stateless Hashcash-style proof of
// work used to throttle abuse of public receive-link uploads.
//
// A challenge is HMAC-signed with a per-process secret so the server keeps no
// per-challenge state to issue it; only a small in-memory set of already-spent
// challenges is kept to stop replay within the (short) validity window. The
// client must find a solution whose SHA-256(challenge + "." + solution) has at
// least `bits` leading zero bits — cheap for one human upload, expensive to
// repeat at bot scale. It needs no external service and keeps the strict
// `script-src 'self'` CSP intact (the browser solves it with WebCrypto).
package pow

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalid  = errors.New("pow: invalid challenge")
	ErrExpired  = errors.New("pow: challenge expired")
	ErrReplay   = errors.New("pow: challenge already used")
	ErrUnsolved = errors.New("pow: solution does not meet difficulty")
)

// Manager issues and verifies proof-of-work challenges.
type Manager struct {
	secret []byte
	bits   int
	ttl    time.Duration

	mu       sync.Mutex
	used     map[string]time.Time // spent challenge -> issue time (replay guard)
	stop     chan struct{}
	stopOnce sync.Once
}

// New returns a Manager requiring `bits` leading zero bits, with challenges
// valid for ttl. It starts a cleanup goroutine (drain via Stop).
func New(bits int, ttl time.Duration) (*Manager, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	m := &Manager{
		secret: secret,
		bits:   bits,
		ttl:    ttl,
		used:   make(map[string]time.Time),
		stop:   make(chan struct{}),
	}
	go m.cleanupLoop()
	return m, nil
}

// FromEnv builds a Manager from RECEIVE_POW_BITS (leading zero bits required).
// Returns (nil, nil) when unset or <= 0 — proof of work disabled (opt-in).
func FromEnv() (*Manager, error) {
	raw := strings.TrimSpace(os.Getenv("RECEIVE_POW_BITS"))
	if raw == "" {
		return nil, nil
	}
	bits, err := strconv.Atoi(raw)
	if err != nil || bits <= 0 {
		return nil, nil
	}
	if bits > 32 {
		bits = 32 // guardrail: keep human-solvable
	}
	return New(bits, 10*time.Minute)
}

// Bits reports the configured difficulty (leading zero bits).
func (m *Manager) Bits() int { return m.bits }

// Issue returns a fresh signed challenge string:
//
//	<nonceB64>.<unixSeconds>.<bits>.<hmacHex>
func (m *Manager) Issue() (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(nonce) + "." +
		strconv.FormatInt(nowUnix(), 10) + "." +
		strconv.Itoa(m.bits)
	return payload + "." + m.sign(payload), nil
}

// Verify checks the challenge's signature, freshness, single-use, and that the
// solution meets the required difficulty. It consumes the challenge on the
// first valid call (replay-protected).
func (m *Manager) Verify(challenge, solution string) error {
	parts := strings.Split(challenge, ".")
	if len(parts) != 4 {
		return ErrInvalid
	}
	payload := parts[0] + "." + parts[1] + "." + parts[2]
	if subtle.ConstantTimeCompare([]byte(parts[3]), []byte(m.sign(payload))) != 1 {
		return ErrInvalid
	}

	ts, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return ErrInvalid
	}
	if nowUnix()-ts > int64(m.ttl.Seconds()) {
		return ErrExpired
	}
	bits, err := strconv.Atoi(parts[2])
	if err != nil || bits <= 0 {
		return ErrInvalid
	}

	// Single-use: reject a challenge already spent within its window.
	m.mu.Lock()
	if _, seen := m.used[challenge]; seen {
		m.mu.Unlock()
		return ErrReplay
	}
	m.used[challenge] = time.Unix(ts, 0)
	m.mu.Unlock()

	sum := sha256.Sum256([]byte(challenge + "." + solution))
	if leadingZeroBits(sum[:]) < bits {
		return ErrUnsolved
	}
	return nil
}

func (m *Manager) sign(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(m.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cutoff := time.Unix(nowUnix(), 0).Add(-m.ttl)
			m.mu.Lock()
			for c, ts := range m.used {
				if ts.Before(cutoff) {
					delete(m.used, c)
				}
			}
			m.mu.Unlock()
		case <-m.stop:
			return
		}
	}
}

// Stop drains the cleanup goroutine.
func (m *Manager) Stop() {
	if m == nil {
		return
	}
	m.stopOnce.Do(func() { close(m.stop) })
}

// leadingZeroBits counts the number of leading zero bits in b.
func leadingZeroBits(b []byte) int {
	count := 0
	for _, by := range b {
		if by == 0 {
			count += 8
			continue
		}
		for bit := 7; bit >= 0; bit-- {
			if by&(1<<uint(bit)) == 0 {
				count++
			} else {
				return count
			}
		}
		break
	}
	return count
}

// nowUnix is a seam so tests aren't time-flaky; wall clock in production.
var nowUnix = func() int64 { return time.Now().Unix() }
