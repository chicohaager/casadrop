package scan

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// chunkSize is the INSTREAM chunk payload size. clamd reads a 4-byte big-endian
// length followed by that many bytes; a zero length terminates the stream.
const chunkSize = 64 * 1024

// clamdScanner talks to clamd over its INSTREAM protocol. It holds no
// connection state — every ScanFile dials a fresh connection, so the scanner
// is safe for concurrent use across upload handlers.
type clamdScanner struct {
	network string // "tcp" or "unix"
	address string
	timeout time.Duration
}

// NewClamd returns a Scanner backed by clamd at cfg.Addr.
func NewClamd(cfg Config) Scanner {
	network, address := parseAddr(cfg.Addr)
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &clamdScanner{network: network, address: address, timeout: timeout}
}

// parseAddr maps the CLAMAV_ADDR forms onto (network, address) for net.Dial.
func parseAddr(addr string) (network, address string) {
	switch {
	case strings.HasPrefix(addr, "unix:"):
		return "unix", strings.TrimPrefix(addr, "unix:")
	case strings.HasPrefix(addr, "tcp://"):
		return "tcp", strings.TrimPrefix(addr, "tcp://")
	default:
		return "tcp", addr
	}
}

func (c *clamdScanner) ScanFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("clamav: open %s: %w", path, err)
	}
	defer f.Close()

	conn, err := net.DialTimeout(c.network, c.address, c.timeout)
	if err != nil {
		return fmt.Errorf("clamav: dial %s/%s: %w", c.network, c.address, err)
	}
	defer conn.Close()

	// Bound the whole exchange so a wedged clamd can't hang an upload handler.
	if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return fmt.Errorf("clamav: set deadline: %w", err)
	}

	// The 'z'-prefixed command form is null-terminated (INSTREAM streams binary
	// data that may contain newlines, so the newline-terminated 'n' form is
	// unsafe here).
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return fmt.Errorf("clamav: write command: %w", err)
	}

	if err := streamChunks(conn, f); err != nil {
		return fmt.Errorf("clamav: stream: %w", err)
	}

	return parseResponse(conn)
}

// streamChunks writes the file to clamd as length-prefixed chunks, terminated
// by a zero-length chunk.
func streamChunks(w io.Writer, r io.Reader) error {
	buf := make([]byte, chunkSize)
	var lenBuf [4]byte
	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			binary.BigEndian.PutUint32(lenBuf[:], uint32(n))
			if _, err := w.Write(lenBuf[:]); err != nil {
				return err
			}
			if _, err := w.Write(buf[:n]); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	// Zero-length chunk signals end of stream.
	binary.BigEndian.PutUint32(lenBuf[:], 0)
	_, err := w.Write(lenBuf[:])
	return err
}

// parseResponse reads clamd's verdict. Clean → "stream: OK"; a detection →
// "stream: <Signature> FOUND"; anything else is a protocol/infra error.
func parseResponse(r io.Reader) error {
	line, err := bufio.NewReader(r).ReadString('\x00')
	if err != nil && err != io.EOF {
		return fmt.Errorf("clamav: read response: %w", err)
	}
	line = strings.TrimRight(line, "\x00\n ")

	switch {
	case strings.HasSuffix(line, "OK"):
		return nil
	case strings.HasSuffix(line, "FOUND"):
		// Format: "stream: <Signature> FOUND"
		sig := strings.TrimSpace(strings.TrimPrefix(line, "stream:"))
		sig = strings.TrimSpace(strings.TrimSuffix(sig, "FOUND"))
		if sig == "" {
			sig = "unknown"
		}
		return &InfectedError{Signature: sig}
	default:
		return fmt.Errorf("clamav: unexpected response %q", line)
	}
}
