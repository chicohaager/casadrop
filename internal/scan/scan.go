// Package scan provides optional anti-malware scanning of uploaded files via
// clamd (ClamAV daemon), spoken over its raw INSTREAM protocol with no third-
// party dependency (same philosophy as internal/totp).
//
// It exists because receive links (/r/{id}/upload) accept files from anonymous,
// unauthenticated strangers on an internet-exposed instance. Scanning is opt-in:
// with no CLAMAV_ADDR configured the scanner is disabled and uploads behave as
// before. When it IS configured, the caller is expected to fail closed — reject
// the upload — on any scanner error, so a broken/unreachable scanner can never
// silently wave malware through.
package scan

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Scanner scans a file already written to disk.
//
// Return contract:
//   - nil                 → file is clean
//   - *InfectedError      → a signature matched; caller must delete + reject
//   - any other error     → scan could not be completed (infra/protocol);
//     caller must fail closed (delete + reject) since the file is unverified.
type Scanner interface {
	ScanFile(path string) error
}

// InfectedError reports that clamd matched a malware signature.
type InfectedError struct {
	Signature string
}

func (e *InfectedError) Error() string {
	return "malware detected: " + e.Signature
}

// IsInfected reports whether err is an *InfectedError (a real detection) as
// opposed to an infrastructure/protocol failure. Both cause a fail-closed
// rejection, but the caller uses this to pick the status code and message.
func IsInfected(err error) bool {
	_, ok := err.(*InfectedError)
	return ok
}

// Config controls scanner construction.
type Config struct {
	// Addr is the clamd address. Accepted forms:
	//   host:port          → TCP
	//   tcp://host:port    → TCP
	//   unix:/var/run/...  → Unix domain socket
	// Empty disables scanning (FromEnv returns a nil Scanner).
	Addr    string
	Timeout time.Duration
}

// FromEnv builds a Scanner from CLAMAV_ADDR (optionally CLAMAV_TIMEOUT, seconds).
// It returns (nil, nil) when CLAMAV_ADDR is unset — scanning disabled, not an
// error. A returned Scanner is safe for concurrent use (each call dials fresh).
func FromEnv() (Scanner, error) {
	addr := strings.TrimSpace(os.Getenv("CLAMAV_ADDR"))
	if addr == "" {
		return nil, nil
	}

	timeout := 30 * time.Second
	if v := strings.TrimSpace(os.Getenv("CLAMAV_TIMEOUT")); v != "" {
		secs, err := time.ParseDuration(v + "s")
		if err != nil {
			return nil, fmt.Errorf("invalid CLAMAV_TIMEOUT %q: %w", v, err)
		}
		timeout = secs
	}

	return NewClamd(Config{Addr: addr, Timeout: timeout}), nil
}
