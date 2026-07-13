package scan

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeClamd is a minimal INSTREAM server that reassembles the streamed file,
// then replies with a caller-supplied verdict. It lets us verify the client's
// protocol framing without a real clamd.
type fakeClamd struct {
	ln       net.Listener
	reply    string // e.g. "stream: OK\x00" or "stream: Eicar FOUND\x00"
	received chan []byte
}

func newFakeClamd(t *testing.T, reply string) *fakeClamd {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	f := &fakeClamd{ln: ln, reply: reply, received: make(chan []byte, 1)}
	go f.serve()
	t.Cleanup(func() { ln.Close() })
	return f
}

func (f *fakeClamd) addr() string { return f.ln.Addr().String() }

func (f *fakeClamd) serve() {
	conn, err := f.ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	br := bufio.NewReader(conn)

	// Read the command line up to the null terminator ("zINSTREAM\0").
	if _, err := br.ReadString('\x00'); err != nil {
		return
	}

	// Reassemble length-prefixed chunks until the zero-length terminator.
	var payload []byte
	for {
		var lenBuf [4]byte
		if _, err := io.ReadFull(br, lenBuf[:]); err != nil {
			return
		}
		n := binary.BigEndian.Uint32(lenBuf[:])
		if n == 0 {
			break
		}
		chunk := make([]byte, n)
		if _, err := io.ReadFull(br, chunk); err != nil {
			return
		}
		payload = append(payload, chunk...)
	}
	f.received <- payload
	conn.Write([]byte(f.reply))
}

func writeTemp(t *testing.T, data string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "upload.bin")
	if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return p
}

func TestScanFile_Clean(t *testing.T) {
	f := newFakeClamd(t, "stream: OK\x00")
	s := NewClamd(Config{Addr: f.addr(), Timeout: 2 * time.Second})

	if err := s.ScanFile(writeTemp(t, "hello world")); err != nil {
		t.Fatalf("expected clean, got %v", err)
	}
	// Verify framing reassembled the exact bytes we streamed.
	got := <-f.received
	if string(got) != "hello world" {
		t.Fatalf("server received %q, want %q", got, "hello world")
	}
}

func TestScanFile_Infected(t *testing.T) {
	f := newFakeClamd(t, "stream: Eicar-Test-Signature FOUND\x00")
	s := NewClamd(Config{Addr: f.addr(), Timeout: 2 * time.Second})

	err := s.ScanFile(writeTemp(t, "x5O!P%..."))
	if !IsInfected(err) {
		t.Fatalf("expected InfectedError, got %v", err)
	}
	if sig := err.(*InfectedError).Signature; sig != "Eicar-Test-Signature" {
		t.Fatalf("signature = %q, want Eicar-Test-Signature", sig)
	}
}

func TestScanFile_LargeFileChunking(t *testing.T) {
	// Larger than one chunk (64 KiB) to exercise multi-chunk framing.
	data := strings.Repeat("A", chunkSize*2+123)
	f := newFakeClamd(t, "stream: OK\x00")
	s := NewClamd(Config{Addr: f.addr(), Timeout: 5 * time.Second})

	if err := s.ScanFile(writeTemp(t, data)); err != nil {
		t.Fatalf("expected clean, got %v", err)
	}
	if got := <-f.received; len(got) != len(data) {
		t.Fatalf("server received %d bytes, want %d", len(got), len(data))
	}
}

func TestScanFile_UnreachableIsError(t *testing.T) {
	// Nothing listening → dial fails → non-infected error (caller fail-closes).
	s := NewClamd(Config{Addr: "127.0.0.1:1", Timeout: 500 * time.Millisecond})
	err := s.ScanFile(writeTemp(t, "data"))
	if err == nil {
		t.Fatal("expected error for unreachable clamd")
	}
	if IsInfected(err) {
		t.Fatal("unreachable clamd must not be reported as an infection")
	}
}

func TestFromEnv_DisabledWhenUnset(t *testing.T) {
	t.Setenv("CLAMAV_ADDR", "")
	s, err := FromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != nil {
		t.Fatal("expected nil scanner when CLAMAV_ADDR unset")
	}
}

func TestParseAddr(t *testing.T) {
	cases := map[string][2]string{
		"clamav:3310":            {"tcp", "clamav:3310"},
		"tcp://clamav:3310":      {"tcp", "clamav:3310"},
		"unix:/var/run/clamd.sk": {"unix", "/var/run/clamd.sk"},
	}
	for in, want := range cases {
		net, addr := parseAddr(in)
		if net != want[0] || addr != want[1] {
			t.Errorf("parseAddr(%q) = (%q,%q), want (%q,%q)", in, net, addr, want[0], want[1])
		}
	}
}
