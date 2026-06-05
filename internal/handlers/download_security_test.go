package handlers

import "testing"

// Locks in the anti-stored-XSS rule: only a strict media allow-list may be served
// inline; anything that can execute/render in our origin must be a download.
func TestInlineContentType(t *testing.T) {
	cases := []struct {
		mime       string
		wantType   string
		wantInline bool
	}{
		{"video/mp4", "video/mp4", true},
		{"audio/mpeg", "audio/mpeg", true},
		{"image/png", "image/png", true},
		{"image/jpeg", "image/jpeg", true},
		{"application/pdf", "application/pdf", true},
		// Dangerous-if-inline types must be forced to attachment/octet-stream.
		{"text/html", "application/octet-stream", false},
		{"image/svg+xml", "application/octet-stream", false},
		{"application/xml", "application/octet-stream", false},
		{"text/plain", "application/octet-stream", false},
		{"application/octet-stream", "application/octet-stream", false},
		{"", "application/octet-stream", false},
	}
	for _, c := range cases {
		gotType, gotInline := inlineContentType(c.mime)
		if gotType != c.wantType || gotInline != c.wantInline {
			t.Errorf("inlineContentType(%q) = (%q,%v), want (%q,%v)",
				c.mime, gotType, gotInline, c.wantType, c.wantInline)
		}
	}
}

// Locks in CR/LF stripping (response-header-injection defence) and quote escaping.
func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"normal.pdf", "normal.pdf"},
		{"with\r\nCRLF.txt", "withCRLF.txt"},
		{"tab\tand\x00null", "tabandnull"},
		{`quote".and\back`, `quote\".and\\back`},
	}
	for _, c := range cases {
		if got := sanitizeFilename(c.in); got != c.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
