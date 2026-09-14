package audit

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"casadrop/internal/middleware"
	"casadrop/internal/models"
)

type fakeStore struct {
	events     []*models.Event
	purgeCalls []time.Time
	failWrites bool
}

func (f *fakeStore) RecordEvent(e *models.Event) error {
	if f.failWrites {
		return errors.New("disk full")
	}
	f.events = append(f.events, e)
	return nil
}

func (f *fakeStore) PurgeEventsBefore(cutoff time.Time) (int64, error) {
	f.purgeCalls = append(f.purgeCalls, cutoff)
	return 3, nil
}

func TestRecordTakesActorAndAddressFromRequest(t *testing.T) {
	store := &fakeStore{}
	rec := New(store, 0)

	req := httptest.NewRequest("GET", "/d/abc", nil)
	req.RemoteAddr = "192.0.2.44:5555"
	req.Header.Set("User-Agent", "Mozilla/5.0 test")
	req = req.WithContext(middleware.ContextWithUser(req.Context(), &middleware.SessionUser{
		ID: "u1", Email: "user@example.com", Role: models.RoleUser,
	}))

	rec.Record(req, models.EventShareDownloaded, models.Event{ShareID: "abc", Detail: "file.txt"})

	if len(store.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(store.events))
	}
	e := store.events[0]
	if e.Kind != models.EventShareDownloaded || e.ShareID != "abc" || e.Detail != "file.txt" {
		t.Errorf("caller fields lost: %+v", e)
	}
	if e.IP != "192.0.2.44" {
		t.Errorf("ip = %q, want the peer address", e.IP)
	}
	if e.UserAgent != "Mozilla/5.0 test" {
		t.Errorf("user agent = %q", e.UserAgent)
	}
	if e.ActorID != "u1" || e.ActorEmail != "user@example.com" {
		t.Errorf("actor not taken from context: %+v", e)
	}
}

func TestSingleAdminSessionIsNamedByRole(t *testing.T) {
	store := &fakeStore{}
	rec := New(store, 0)

	req := httptest.NewRequest("POST", "/api/upload", nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), &middleware.SessionUser{Role: models.RoleAdmin}))
	rec.Record(req, models.EventShareCreated, models.Event{ShareID: "s"})

	if e := store.events[0]; e.ActorID != "admin" || e.ActorEmail != "" {
		t.Errorf("legacy admin session should be recorded as actor 'admin', got %+v", e)
	}
}

func TestRecordAnonymousRequestHasNoActor(t *testing.T) {
	store := &fakeStore{}
	rec := New(store, 0)

	req := httptest.NewRequest("GET", "/d/abc", nil)
	rec.Record(req, models.EventShareDownloaded, models.Event{ShareID: "abc"})

	if e := store.events[0]; e.ActorID != "" || e.ActorEmail != "" {
		t.Errorf("anonymous download must have no actor, got %+v", e)
	}
}

func TestUserAgentIsBounded(t *testing.T) {
	store := &fakeStore{}
	rec := New(store, 0)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", strings.Repeat("x", 4096))
	rec.Record(req, models.EventShareDownloaded, models.Event{})

	if got := len(store.events[0].UserAgent); got != maxUserAgentLen {
		t.Errorf("user agent stored with %d bytes, want %d", got, maxUserAgentLen)
	}
}

func TestWriteFailureIsLoggedNotFatal(t *testing.T) {
	rec := New(&fakeStore{failWrites: true}, 0)
	// Must not panic and must not return anything to the caller: the download
	// already happened.
	rec.RecordSystem(models.EventShareExpired, models.Event{ShareID: "gone"})
}

func TestAuthSinkMapsKnownAndUnknownTypes(t *testing.T) {
	store := &fakeStore{}
	rec := New(store, 0)

	rec.AuthSink(middleware.AuditLoginFailed, "192.0.2.1", "ua", "bad password for admin")
	rec.AuthSink(middleware.AuditCSRFViolation, "192.0.2.1", "ua", "POST /api/shares")

	if len(store.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(store.events))
	}
	if store.events[0].Kind != models.EventLoginFailed {
		t.Errorf("LOGIN_FAILED mapped to %s", store.events[0].Kind)
	}
	// An unmapped type must survive as a security event that still names itself.
	if store.events[1].Kind != models.EventSecurity || !strings.HasPrefix(store.events[1].Detail, "CSRF_VIOLATION: ") {
		t.Errorf("unmapped type lost its name: %+v", store.events[1])
	}
}

func TestEveryMiddlewareAuditTypeIsMappedOrDeliberatelyGeneric(t *testing.T) {
	// The mapping is the contract between two packages. New middleware types
	// should be added here on purpose, not fall through by accident.
	known := []middleware.AuditEventType{
		middleware.AuditLoginSuccess, middleware.AuditLoginFailed, middleware.AuditLoginLocked,
		middleware.AuditLogout, middleware.AuditSetupComplete, middleware.AuditSessionRevoked,
	}
	for _, k := range known {
		if _, ok := authKinds[k]; !ok {
			t.Errorf("%s has no EventKind mapping", k)
		}
	}
}

func TestPurgeHonoursRetention(t *testing.T) {
	store := &fakeStore{}
	rec := New(store, 90*24*time.Hour)

	before := time.Now()
	n, err := rec.Purge()
	if err != nil || n != 3 {
		t.Fatalf("Purge returned n=%d err=%v", n, err)
	}
	if len(store.purgeCalls) != 1 {
		t.Fatalf("expected one purge call, got %d", len(store.purgeCalls))
	}
	want := before.Add(-90 * 24 * time.Hour)
	if d := store.purgeCalls[0].Sub(want); d < 0 || d > time.Second {
		t.Errorf("cutoff %v is not 90 days before now (%v)", store.purgeCalls[0], want)
	}
}

func TestZeroRetentionNeverPurges(t *testing.T) {
	store := &fakeStore{}
	rec := New(store, 0)
	rec.Start() // returns without a goroutine
	rec.Stop()
	if n, _ := rec.Purge(); n != 0 || len(store.purgeCalls) != 0 {
		t.Errorf("retention 0 must never purge, got n=%d calls=%d", n, len(store.purgeCalls))
	}
}

func TestRetentionFromEnv(t *testing.T) {
	cases := []struct {
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{"", DefaultRetentionDays * 24 * time.Hour, false},
		{"0", 0, false},
		{"30", 30 * 24 * time.Hour, false},
		{"-1", 0, true},
		{"forever", 0, true},
	}
	for _, c := range cases {
		t.Setenv(retentionEnv, c.raw)
		got, err := RetentionFromEnv()
		if (err != nil) != c.wantErr {
			t.Errorf("%q: err=%v, wantErr=%v", c.raw, err, c.wantErr)
			continue
		}
		if !c.wantErr && got != c.want {
			t.Errorf("%q: got %v, want %v", c.raw, got, c.want)
		}
	}
}
