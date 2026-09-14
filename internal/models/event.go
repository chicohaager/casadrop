package models

import "time"

// EventKind names what happened. The values are stable identifiers: they are
// stored in the database, returned by the API and written to CSV exports, so
// renaming one is a migration, not a refactor.
type EventKind string

const (
	EventShareCreated    EventKind = "share.created"
	EventShareUpdated    EventKind = "share.updated"
	EventShareDeleted    EventKind = "share.deleted"
	EventShareExpired    EventKind = "share.expired"
	EventShareDownloaded EventKind = "share.downloaded"
	EventShareStreamed   EventKind = "share.streamed"
	EventReceiveUploaded EventKind = "receive.uploaded"
	EventLoginSuccess    EventKind = "auth.login"
	EventLoginFailed     EventKind = "auth.login_failed"
	EventLoginLocked     EventKind = "auth.locked"
	EventLogout          EventKind = "auth.logout"
	EventSetup           EventKind = "auth.setup"
	EventSessionRevoked  EventKind = "session.revoked"
	EventSecurity        EventKind = "security"
)

// Event is one row of the activity log: who did what, to which share, from
// where. Every field except ID, At and Kind is optional — a login failure has
// no share, an expiry sweep has no request.
type Event struct {
	ID         string    `json:"id"`
	At         time.Time `json:"at"`
	Kind       EventKind `json:"kind"`
	ActorID    string    `json:"actor_id,omitempty"`
	ActorEmail string    `json:"actor_email,omitempty"`
	ShareID    string    `json:"share_id,omitempty"`
	LinkID     string    `json:"link_id,omitempty"`
	IP         string    `json:"ip,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	Detail     string    `json:"detail,omitempty"`
}

// EventFilter narrows an activity-log query. Zero values mean "no constraint".
type EventFilter struct {
	ShareID string
	LinkID  string
	Kind    EventKind
	ActorID string
	Since   time.Time
	Until   time.Time
	Limit   int
	Offset  int
}
