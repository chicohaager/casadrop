package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"casadrop/internal/middleware"
)

// TestStreamRateLimitSparesRangeRequests pins both halves of the /stream
// throttle: the requests that can burn a share's download budget are limited
// (one curl loop must not exhaust a max_downloads share before its recipient
// opens the link), and Range requests are not (a video player scrubbing through
// a file would otherwise start collecting 429s mid-playback).
//
// This lives in an internal test file because the guard is unexported and the
// distinction it makes — Range vs. no Range — is invisible from the outside
// without driving a real media file through the whole router.
func TestStreamRateLimitSparesRangeRequests(t *testing.T) {
	limiter := middleware.NewRateLimiter(3, time.Minute)
	defer limiter.Stop()

	var served int
	h := rateLimitStream(func(w http.ResponseWriter, r *http.Request) {
		served++
		w.WriteHeader(http.StatusOK)
	}, limiter)

	call := func(method string, withRange bool) int {
		req := httptest.NewRequest(method, "/stream/abc", nil)
		if withRange {
			req.Header.Set("Range", "bytes=0-1023")
		}
		req.RemoteAddr = "203.0.113.7:1234"
		rec := httptest.NewRecorder()
		h(rec, req)
		return rec.Code
	}

	// Far past the budget of 3: playback must never be throttled.
	for i := 0; i < 20; i++ {
		if code := call(http.MethodGet, true); code != http.StatusOK {
			t.Fatalf("range request %d: status %d, want 200 — seeking must not be rate-limited", i, code)
		}
	}
	if served != 20 {
		t.Errorf("range requests served = %d, want 20", served)
	}

	// A CORS preflight carries no payload and must not consume budget either.
	if code := call(http.MethodOptions, false); code != http.StatusOK {
		t.Errorf("OPTIONS preflight: status %d, want 200", code)
	}

	// Plain GETs are the ones that increment the counter: budget 3, so the
	// fourth must be refused.
	var last int
	for i := 0; i < 4; i++ {
		last = call(http.MethodGet, false)
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("4th counting request: status %d, want 429 — /stream is unthrottled again", last)
	}
}
