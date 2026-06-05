// Package utils provides common utility functions used across CasaDrop
package utils

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

// trustedProxies holds the parsed TRUSTED_PROXY allow-list. When non-empty,
// X-Forwarded-For / X-Real-IP are only honored if the direct peer (RemoteAddr)
// is one of these networks. When empty (the default), forwarded headers are
// trusted — correct for the documented "always behind a reverse proxy" setup.
var (
	trustedProxies     []*net.IPNet
	trustedProxiesOnce sync.Once
)

func loadTrustedProxies() {
	raw := os.Getenv("TRUSTED_PROXY")
	if raw == "" {
		return
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.Contains(part, "/") {
			// Bare IP → /32 or /128
			if strings.Contains(part, ":") {
				part += "/128"
			} else {
				part += "/32"
			}
		}
		if _, network, err := net.ParseCIDR(part); err == nil {
			trustedProxies = append(trustedProxies, network)
		}
	}
}

// remoteIP returns the connecting peer's IP (RemoteAddr without port).
func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// peerIsTrustedProxy reports whether the direct peer is in the TRUSTED_PROXY set.
func peerIsTrustedProxy(r *http.Request) bool {
	if len(trustedProxies) == 0 {
		return false
	}
	ip := net.ParseIP(remoteIP(r))
	if ip == nil {
		return false
	}
	for _, n := range trustedProxies {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// GetClientIP extracts the client IP address from an HTTP request.
//
// If TRUSTED_PROXY is configured, X-Forwarded-For / X-Real-IP are honored only
// when the request actually arrives from one of those proxies — otherwise a
// client could spoof its IP and defeat per-IP rate limiting / lockout. If
// TRUSTED_PROXY is unset, forwarded headers are trusted (default for the
// always-proxied deployment model). Only the leftmost XFF entry is used.
func GetClientIP(r *http.Request) string {
	trustedProxiesOnce.Do(loadTrustedProxies)

	honorForwarded := len(trustedProxies) == 0 || peerIsTrustedProxy(r)
	if honorForwarded {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if idx := strings.Index(xff, ","); idx != -1 {
				return strings.TrimSpace(xff[:idx])
			}
			return strings.TrimSpace(xff)
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
	}
	return remoteIP(r)
}

// ClampExpiryHours bounds a user-supplied "expires in N hours" value so the
// subsequent time.Duration(N)*time.Hour multiplication can't overflow into a
// negative (already-expired) duration. Negative input is treated as 0.
func ClampExpiryHours(hours int) int {
	const maxHours = 100 * 365 * 24 // 100 years
	if hours < 0 {
		return 0
	}
	if hours > maxHours {
		return maxHours
	}
	return hours
}

// FormatFileSize formats a byte size into a human-readable string
func FormatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// GetBaseURL extracts the base URL from an HTTP request,
// handling reverse proxy headers (X-Forwarded-Proto, X-Forwarded-Host)
func GetBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Check X-Forwarded-Proto header (for reverse proxies)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := r.Host
	// Check X-Forwarded-Host header (for reverse proxies)
	if fwdHost := r.Header.Get("X-Forwarded-Host"); fwdHost != "" {
		host = fwdHost
	}

	return fmt.Sprintf("%s://%s", scheme, host)
}

// ValidateURL validates a URL string
// If requireHTTPS is true, only HTTPS URLs are accepted
// Empty URLs are considered valid (for optional fields)
func ValidateURL(urlStr string, requireHTTPS bool) error {
	if urlStr == "" {
		return nil // Empty URL is valid (optional field)
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format")
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL must include a host")
	}

	if requireHTTPS {
		if parsed.Scheme != "https" {
			return fmt.Errorf("URL must use HTTPS")
		}
	} else {
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("URL must use HTTP or HTTPS")
		}
	}

	return nil
}

// ValidateExternalWebhookURL rejects URLs that resolve to loopback,
// link-local, private, multicast, or unspecified addresses. Use this for
// webhook targets that are supplied by end users so the server can't be
// tricked into reaching internal services (SSRF).
//
// Note: this only blocks literal IP addresses. Hostnames that resolve to
// private IPs would still work. That's a trade-off — strict DNS pinning
// would break self-hosted receivers on the same LAN, which is a legitimate
// homelab use case for CasaDrop.
func ValidateExternalWebhookURL(urlStr string) error {
	if err := ValidateURL(urlStr, false); err != nil {
		return err
	}
	if urlStr == "" {
		return nil
	}
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format")
	}
	host := parsed.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if IsBlockedIP(ip) {
			return fmt.Errorf("webhook URL must not target a private or loopback address")
		}
	}
	return nil
}

// IsBlockedIP reports whether an IP must never be reached by server-initiated
// requests (SSRF targets): loopback, link-local, multicast, unspecified, or
// RFC1918/ULA private ranges.
func IsBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsPrivate()
}

// IsLocalHostname reports whether host (without port) refers to the local
// machine or a private LAN address — i.e. NOT a public/tunnel hostname.
func IsLocalHostname(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".local") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
	}
	return false // any other DNS name (ts.net, custom domain, …) is treated as public
}

// PreferredPublicBaseURL returns the request-derived base URL (honoring
// X-Forwarded-Host/Proto) when the client reached the server via a public or
// tunnel hostname (Pangolin, Tailscale, custom domain), so share links match
// the access path. It returns "" for local/LAN/loopback access, signalling the
// caller to fall back to its configured primary-network URL.
func PreferredPublicBaseURL(r *http.Request) string {
	base := GetBaseURL(r)
	u, err := url.Parse(base)
	if err != nil || IsLocalHostname(u.Hostname()) {
		return ""
	}
	return strings.TrimSuffix(base, "/")
}

// SanitizeFilename removes or replaces characters that could be problematic in filenames
func SanitizeFilename(name string) string {
	// Replace common problematic characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}

// TruncateString truncates a string to maxLen characters, adding "..." if truncated
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
