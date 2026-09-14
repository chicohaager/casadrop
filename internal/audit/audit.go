// Package audit writes the activity log: one row per thing that happened to a
// share, a receive link or a login, with who did it and from where.
//
// The log answers the question a counter cannot — "who downloaded this file,
// and when?" — and it is the durable home for the security events the auth
// middleware used to print to stdout only.
package audit

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"casadrop/internal/middleware"
	"casadrop/internal/models"
	"casadrop/internal/utils"
)

// DefaultRetentionDays is how long rows are kept unless EVENT_RETENTION_DAYS
// says otherwise. Ninety days covers a quarter of "did the customer get it?"
// questions without keeping visitor addresses around forever.
const DefaultRetentionDays = 90

// retentionEnv is the environment variable that overrides the retention.
// 0 disables purging and keeps every row.
const retentionEnv = "EVENT_RETENTION_DAYS"

// purgeInterval is how often the retention sweep runs. Daily is plenty: the
// policy is measured in days, and the sweep is one indexed DELETE.
const purgeInterval = 24 * time.Hour

// maxUserAgentLen bounds what is stored per row. Browsers send ~120 bytes; a
// 4 KB header is a probe, not a browser, and does not deserve 4 KB of disk.
const maxUserAgentLen = 256

// Store is the slice of the storage layer the recorder needs. Kept minimal so a
// test can supply a fake without the whole SQLite backend.
type Store interface {
	RecordEvent(event *models.Event) error
	PurgeEventsBefore(cutoff time.Time) (int64, error)
}

// Recorder writes events and enforces retention. Zero-value is not usable;
// construct with New.
type Recorder struct {
	store     Store
	retention time.Duration // 0 = keep forever
	stop      chan struct{}
	stopOnce  sync.Once
}

// New returns a recorder with the given retention. A retention of 0 keeps rows
// forever. Call Start to run the retention sweep.
func New(store Store, retention time.Duration) *Recorder {
	return &Recorder{store: store, retention: retention, stop: make(chan struct{})}
}

// RetentionFromEnv reads EVENT_RETENTION_DAYS. Unset means the default; a
// value that is not a non-negative integer is an operator error and is
// returned as such rather than silently becoming "forever" or "default".
func RetentionFromEnv() (time.Duration, error) {
	raw := os.Getenv(retentionEnv)
	if raw == "" {
		return DefaultRetentionDays * 24 * time.Hour, nil
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < 0 {
		return 0, &InvalidRetentionError{Value: raw}
	}
	return time.Duration(days) * 24 * time.Hour, nil
}

// InvalidRetentionError reports a malformed EVENT_RETENTION_DAYS.
type InvalidRetentionError struct{ Value string }

func (e *InvalidRetentionError) Error() string {
	return retentionEnv + " must be a non-negative number of days, got " + strconv.Quote(e.Value)
}

// Retention returns the configured retention (0 = keep forever).
func (r *Recorder) Retention() time.Duration { return r.retention }

// Record writes an event that happened in the course of an HTTP request. The
// client address, user agent and — when the request is authenticated — the
// actor are taken from the request; the caller supplies what happened.
//
// A failure to record is logged, not returned: the download, upload or login
// that triggered the event has already happened and must not fail because the
// log could not be written.
func (r *Recorder) Record(req *http.Request, kind models.EventKind, event models.Event) {
	event.Kind = kind
	if req != nil {
		event.IP = utils.GetClientIP(req)
		event.UserAgent = truncate(req.Header.Get("User-Agent"), maxUserAgentLen)
		if user := middleware.GetUserFromContext(req.Context()); user != nil {
			event.ActorID = user.ID
			event.ActorEmail = user.Email
			// The single-admin session (env password / setup wizard) has no
			// user record behind it. It is still someone, not an anonymous
			// visitor, so name it by its role.
			if event.ActorID == "" && event.ActorEmail == "" {
				event.ActorID = string(user.Role)
			}
		}
	}
	r.write(&event)
}

// RecordSystem writes an event with no request behind it — an expiry sweep, a
// scheduled job. Actor, address and agent stay empty on purpose.
func (r *Recorder) RecordSystem(kind models.EventKind, event models.Event) {
	event.Kind = kind
	r.write(&event)
}

// AuthSink adapts the auth middleware's audit callback to the activity log. It
// maps the middleware's event vocabulary onto EventKind so the two never drift
// apart silently: an unmapped type lands as EventSecurity with the original
// name in Detail, never dropped.
func (r *Recorder) AuthSink(eventType middleware.AuditEventType, ip, userAgent, details string) {
	kind, ok := authKinds[eventType]
	if !ok {
		kind = models.EventSecurity
		details = string(eventType) + ": " + details
	}
	r.write(&models.Event{
		Kind:      kind,
		IP:        ip,
		UserAgent: truncate(userAgent, maxUserAgentLen),
		Detail:    details,
	})
}

var authKinds = map[middleware.AuditEventType]models.EventKind{
	middleware.AuditLoginSuccess:   models.EventLoginSuccess,
	middleware.AuditLoginFailed:    models.EventLoginFailed,
	middleware.AuditLoginLocked:    models.EventLoginLocked,
	middleware.AuditLogout:         models.EventLogout,
	middleware.AuditSetupComplete:  models.EventSetup,
	middleware.AuditSessionRevoked: models.EventSessionRevoked,
}

// Start runs the retention sweep until Stop is called. With retention 0 it
// returns immediately: there is nothing to sweep.
func (r *Recorder) Start() {
	if r.retention == 0 {
		return
	}
	go func() {
		r.purge()
		ticker := time.NewTicker(purgeInterval)
		defer ticker.Stop()
		for {
			select {
			case <-r.stop:
				return
			case <-ticker.C:
				r.purge()
			}
		}
	}()
}

// Stop ends the retention sweep. Safe to call more than once.
func (r *Recorder) Stop() {
	r.stopOnce.Do(func() { close(r.stop) })
}

// Purge runs one retention sweep now and reports how many rows went. Exposed
// so the policy can be tested without waiting a day.
func (r *Recorder) Purge() (int64, error) {
	if r.retention == 0 {
		return 0, nil
	}
	return r.store.PurgeEventsBefore(time.Now().Add(-r.retention))
}

func (r *Recorder) purge() {
	n, err := r.Purge()
	if err != nil {
		log.Printf("Activity log: retention sweep failed: %v", err)
		return
	}
	if n > 0 {
		log.Printf("Activity log: purged %d events older than %s", n, r.retention)
	}
}

func (r *Recorder) write(event *models.Event) {
	if err := r.store.RecordEvent(event); err != nil {
		log.Printf("Activity log: failed to record %s: %v", event.Kind, err)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
