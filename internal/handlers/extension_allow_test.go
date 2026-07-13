package handlers

import "testing"

// TestIsExtensionAllowed_TrailingBypass guards the blocklist against a trailing
// space or dot slipping a blocked type through. filepath.Ext("evil.exe ") is
// ".exe " and filepath.Ext("evil.exe.") is ".", neither of which matched the
// blocked key ".exe" before the fix — and Windows strips trailing dots, so
// "evil.exe." is really executable. This is the handlers.TunnelConfig used by
// the actual upload path.
func TestIsExtensionAllowed_TrailingBypass(t *testing.T) {
	cfg := &TunnelConfig{} // default blocklist includes .exe

	cases := []struct {
		filename string
		allowed  bool
	}{
		{"program.exe", false},
		{"program.exe ", false},  // trailing space
		{"program.exe.", false},  // trailing dot
		{"program.exe. ", false}, // trailing dot + space
		{"document.pdf", true},
		{"README", true},
	}

	for _, c := range cases {
		got, _ := cfg.IsExtensionAllowed(c.filename)
		if got != c.allowed {
			t.Errorf("IsExtensionAllowed(%q) = %v, want %v", c.filename, got, c.allowed)
		}
	}
}
