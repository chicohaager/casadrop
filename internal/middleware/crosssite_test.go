package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCrossSiteGuard(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := CrossSiteGuard(next)

	mk := func(method, secFetch string, withCookie, withAuth bool) *http.Request {
		r := httptest.NewRequest(method, "/api/shares/x", nil)
		if secFetch != "" {
			r.Header.Set("Sec-Fetch-Site", secFetch)
		}
		if withCookie {
			r.AddCookie(&http.Cookie{Name: "casadrop_session", Value: "tok"})
		}
		if withAuth {
			r.Header.Set("Authorization", "Bearer x")
		}
		return r
	}

	cases := []struct {
		name string
		req  *http.Request
		want int
	}{
		{"cross-site cookie mutation blocked", mk("DELETE", "cross-site", true, false), http.StatusForbidden},
		{"same-origin cookie mutation allowed", mk("DELETE", "same-origin", true, false), http.StatusOK},
		{"no fetch-metadata (api client) allowed", mk("DELETE", "", true, false), http.StatusOK},
		{"cross-site with bearer (not cookie csrf) allowed", mk("DELETE", "cross-site", false, true), http.StatusOK},
		{"cross-site GET allowed (not state-changing)", mk("GET", "cross-site", true, false), http.StatusOK},
		{"cross-site no cookie allowed", mk("POST", "cross-site", false, false), http.StatusOK},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, c.req)
		if rec.Code != c.want {
			t.Errorf("%s: got %d, want %d", c.name, rec.Code, c.want)
		}
	}
}
