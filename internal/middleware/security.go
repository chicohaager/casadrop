package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeaders fügt wichtige Security-Header hinzu
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verhindert MIME-Type Sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Verhindert Clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// XSS-Schutz (für ältere Browser)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer-Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// HSTS: only emit over HTTPS (direct TLS or via a TLS-terminating proxy)
		// so we never pin HTTP-only LAN deployments into an unreachable state.
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Content-Security-Policy (ES modules eliminate need for unsafe-inline scripts)
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
				"img-src 'self' data: blob: https://api.qrserver.com; "+
				"font-src 'self' https://fonts.gstatic.com; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")

		// Permissions-Policy (moderne Alternative zu Feature-Policy)
		w.Header().Set("Permissions-Policy",
			"accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")

		next.ServeHTTP(w, r)
	})
}

// CrossSiteGuard is a defense-in-depth CSRF mitigation for cookie-authenticated,
// state-changing requests. Modern browsers send the Sec-Fetch-Site fetch-metadata
// header; when it explicitly reports a cross-site context we refuse the request.
// This complements (does not replace) the SameSite=Strict session cookie and the
// CSRF token on the login form. Non-browser API clients (which authenticate with
// an Authorization bearer/API key and send no session cookie, and no Sec-Fetch-Site
// header) are unaffected.
func CrossSiteGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				// Only cookie-borne auth is subject to browser CSRF; an attacker
				// cannot set the Authorization header cross-site.
				if _, err := r.Cookie("casadrop_session"); err == nil && r.Header.Get("Authorization") == "" {
					http.Error(w, "Cross-site request blocked", http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// MaxBodySize returns middleware that limits the size of request bodies.
// Requests with Content-Length exceeding maxBytes are rejected immediately.
// All request bodies are wrapped with http.MaxBytesReader as a safety net.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodySizeSkipPaths returns body-size-limiting middleware that skips
// requests whose path starts with any of the given prefixes (e.g. upload routes).
func MaxBodySizeSkipPaths(maxBytes int64, skipPrefixes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, prefix := range skipPrefixes {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}
			if r.ContentLength > maxBytes {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
