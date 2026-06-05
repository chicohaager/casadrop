package utils

import (
	"net"
	"net/http/httptest"
	"testing"
)

func TestGetClientIP(t *testing.T) {
	// Trust the loopback peer so the X-Forwarded-* cases (whose RemoteAddr is
	// 127.0.0.1) exercise the honored path. Fail-closed cases use a different,
	// untrusted RemoteAddr. We set the package var directly and consume the
	// sync.Once so GetClientIP doesn't reload from the environment.
	_, loop, _ := net.ParseCIDR("127.0.0.0/8")
	trustedProxies = []*net.IPNet{loop}
	trustedProxiesOnce.Do(func() {})

	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4"},
			remoteAddr: "127.0.0.1:12345",
			expected:   "1.2.3.4",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4, 5.6.7.8"},
			remoteAddr: "127.0.0.1:12345",
			expected:   "1.2.3.4",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "9.10.11.12"},
			remoteAddr: "127.0.0.1:12345",
			expected:   "9.10.11.12",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:54321",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			headers:    map[string]string{"X-Forwarded-For": "1.1.1.1", "X-Real-IP": "2.2.2.2"},
			remoteAddr: "127.0.0.1:12345",
			expected:   "1.1.1.1",
		},
		{
			name:       "fail-closed: XFF ignored from untrusted (non-proxy) peer",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4"},
			remoteAddr: "203.0.113.9:4444",
			expected:   "203.0.113.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := GetClientIP(req)
			if result != tt.expected {
				t.Errorf("GetClientIP() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1572864, "1.50 MB"},
		{1073741824, "1.00 GB"},
		{1610612736, "1.50 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatFileSize(tt.size)
			if result != tt.expected {
				t.Errorf("FormatFileSize(%d) = %q, want %q", tt.size, result, tt.expected)
			}
		})
	}
}

func TestGetBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		headers  map[string]string
		expected string
	}{
		{
			name:     "HTTP request",
			host:     "example.com",
			headers:  map[string]string{},
			expected: "http://example.com",
		},
		{
			name:     "X-Forwarded-Proto HTTPS",
			host:     "example.com",
			headers:  map[string]string{"X-Forwarded-Proto": "https"},
			expected: "https://example.com",
		},
		{
			name:     "X-Forwarded-Host",
			host:     "internal.local",
			headers:  map[string]string{"X-Forwarded-Host": "public.example.com"},
			expected: "http://public.example.com",
		},
		{
			name:     "Both forwarded headers",
			host:     "internal.local",
			headers:  map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "public.example.com"},
			expected: "https://public.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://"+tt.host+"/", nil)
			req.Host = tt.host
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := GetBaseURL(req)
			if result != tt.expected {
				t.Errorf("GetBaseURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		requireHTTPS bool
		wantErr      bool
	}{
		{"Empty URL is valid", "", false, false},
		{"Valid HTTP URL", "http://example.com", false, false},
		{"Valid HTTPS URL", "https://example.com", false, false},
		{"HTTPS URL when required", "https://example.com", true, false},
		{"HTTP URL when HTTPS required", "http://example.com", true, true},
		{"Invalid scheme", "ftp://example.com", false, true},
		{"Missing host", "http://", false, true},
		{"Invalid URL", "not a url", false, true},
		{"URL with path", "https://example.com/path", false, false},
		{"URL with port", "https://example.com:8443", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, tt.requireHTTPS)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q, %v) error = %v, wantErr %v", tt.url, tt.requireHTTPS, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.txt", "normal.txt"},
		{"file/with/slashes.txt", "file_with_slashes.txt"},
		{"file\\with\\backslashes.txt", "file_with_backslashes.txt"},
		{"file:with:colons.txt", "file_with_colons.txt"},
		{"file*with*stars.txt", "file_with_stars.txt"},
		{"file?with?questions.txt", "file_with_questions.txt"},
		{"file\"with\"quotes.txt", "file_with_quotes.txt"},
		{"file<with>brackets.txt", "file_with_brackets.txt"},
		{"file|with|pipes.txt", "file_with_pipes.txt"},
		{"all/\\:*?\"<>|bad.txt", "all_________bad.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is too long", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestIsLocalHostname(t *testing.T) {
	local := []string{"localhost", "127.0.0.1", "192.168.1.50", "10.0.0.5", "172.16.0.1", "zimaos.local", ""}
	public := []string{"casadrop.example.com", "node.example.ts.net", "example.com", "8.8.8.8"}
	for _, h := range local {
		if !IsLocalHostname(h) {
			t.Errorf("IsLocalHostname(%q) = false, want true", h)
		}
	}
	for _, h := range public {
		if IsLocalHostname(h) {
			t.Errorf("IsLocalHostname(%q) = true, want false", h)
		}
	}
}

func TestPreferredPublicBaseURL(t *testing.T) {
	// Public host via X-Forwarded-Host → used for links.
	r := httptest.NewRequest("GET", "http://backend/", nil)
	r.Header.Set("X-Forwarded-Host", "casadrop.example.com")
	r.Header.Set("X-Forwarded-Proto", "https")
	if got := PreferredPublicBaseURL(r); got != "https://casadrop.example.com" {
		t.Errorf("public XFH: got %q", got)
	}
	// Local/LAN host → empty (caller falls back to primary network).
	r2 := httptest.NewRequest("GET", "http://192.168.1.50:8086/", nil)
	if got := PreferredPublicBaseURL(r2); got != "" {
		t.Errorf("LAN host: got %q, want empty", got)
	}
}
