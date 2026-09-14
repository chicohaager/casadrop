package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type recordedAudit struct {
	eventType AuditEventType
	ip, ua    string
	details   string
}

// The sink is the bridge from the process log to the activity log. It must
// receive exactly what was logged, and its absence must not break logging.
func TestAuditForwardsToSink(t *testing.T) {
	aa := NewAdminAuth("adminpw", t.TempDir())
	t.Cleanup(aa.Stop)

	// No sink registered: must not panic, must not recurse.
	aa.audit(AuditLoginFailed, "192.0.2.1", "ua", "before sink")

	var got []recordedAudit
	aa.SetAuditSink(func(eventType AuditEventType, ip, userAgent, details string) {
		got = append(got, recordedAudit{eventType, ip, userAgent, details})
	})

	aa.audit(AuditLoginSuccess, "192.0.2.2", "Mozilla/5.0", "admin login")

	if len(got) != 1 {
		t.Fatalf("expected the sink to receive 1 event after registration, got %d", len(got))
	}
	want := recordedAudit{AuditLoginSuccess, "192.0.2.2", "Mozilla/5.0", "admin login"}
	if got[0] != want {
		t.Errorf("sink received %+v, want %+v", got[0], want)
	}
}

// A real login path must reach the sink, not just the audit helper — otherwise
// a future refactor could route a handler around it without any test noticing.
func TestLoginFailureReachesSink(t *testing.T) {
	user := mkUser(t, "u1", "user@example.com", "user", "right", true)
	aa := newAuthWithUsers(t, "", user)

	var kinds []AuditEventType
	aa.SetAuditSink(func(eventType AuditEventType, _, _, _ string) {
		kinds = append(kinds, eventType)
	})

	if _, _, _, ok := aa.resolveLogin("user@example.com", "wrong"); ok {
		t.Fatal("wrong password must not authenticate")
	}
	// resolveLogin itself does not audit; the handlers do. Drive the JSON
	// login handler the way the UI does.
	rr := performJSONLogin(t, aa, "user@example.com", "wrong")
	if rr.Code == 200 {
		t.Fatalf("wrong password answered 200")
	}
	found := false
	for _, k := range kinds {
		if k == AuditLoginFailed {
			found = true
		}
	}
	if !found {
		t.Errorf("expected %s in sink, got %v", AuditLoginFailed, kinds)
	}
}

func performJSONLogin(t *testing.T, aa *AdminAuth, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	body := strings.NewReader(`{"email":` + strconv.Quote(email) + `,"password":` + strconv.Quote(password) + `}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.5:12345"
	rr := httptest.NewRecorder()
	aa.LoginHandler(rr, req)
	return rr
}
