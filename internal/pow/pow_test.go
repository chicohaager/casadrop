package pow

import (
	"crypto/sha256"
	"strconv"
	"testing"
	"time"
)

// solve mirrors what the browser does: find a counter whose
// SHA-256(challenge + "." + counter) has >= bits leading zero bits.
func solve(challenge string, bits int) string {
	for i := 0; ; i++ {
		sol := strconv.Itoa(i)
		sum := sha256.Sum256([]byte(challenge + "." + sol))
		if leadingZeroBits(sum[:]) >= bits {
			return sol
		}
	}
}

func newTestManager(t *testing.T, bits int) *Manager {
	t.Helper()
	m, err := New(bits, time.Minute)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(m.Stop)
	return m
}

func TestIssueSolveVerify(t *testing.T) {
	m := newTestManager(t, 8) // small enough to solve fast in a test
	ch, err := m.Issue()
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	sol := solve(ch, m.Bits())
	if err := m.Verify(ch, sol); err != nil {
		t.Fatalf("Verify valid solution: %v", err)
	}
}

func TestVerifyRejectsWrongSolution(t *testing.T) {
	m := newTestManager(t, 12)
	ch, _ := m.Issue()
	if err := m.Verify(ch, "0"); err != ErrUnsolved {
		t.Fatalf("expected ErrUnsolved, got %v", err)
	}
}

func TestVerifyRejectsTamperedChallenge(t *testing.T) {
	m := newTestManager(t, 8)
	ch, _ := m.Issue()
	// Flip the difficulty to try to make it trivially solvable — the HMAC
	// covers the bits field, so the signature must no longer match.
	tampered := ch[:len(ch)-40] // corrupt the signature tail
	if err := m.Verify(tampered, "0"); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestVerifyRejectsReplay(t *testing.T) {
	m := newTestManager(t, 8)
	ch, _ := m.Issue()
	sol := solve(ch, m.Bits())
	if err := m.Verify(ch, sol); err != nil {
		t.Fatalf("first verify: %v", err)
	}
	if err := m.Verify(ch, sol); err != ErrReplay {
		t.Fatalf("expected ErrReplay on reuse, got %v", err)
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	m := newTestManager(t, 8)
	// Issue "in the past" by moving the clock seam.
	orig := nowUnix
	nowUnix = func() int64 { return orig() - 3600 } // 1h ago
	ch, _ := m.Issue()
	nowUnix = orig
	sol := solve(ch, m.Bits())
	if err := m.Verify(ch, sol); err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestFromEnvDisabledByDefault(t *testing.T) {
	t.Setenv("RECEIVE_POW_BITS", "")
	m, err := FromEnv()
	if err != nil || m != nil {
		t.Fatalf("expected disabled (nil,nil), got (%v,%v)", m, err)
	}
}
