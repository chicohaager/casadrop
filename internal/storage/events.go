package storage

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"casadrop/internal/models"
)

// sqliteTimeLayout is how every DATETIME column in shares.db is written: UTC,
// second precision, no zone suffix. Comparisons in SQL are string comparisons,
// so a bound time.Time (which the driver renders as RFC3339Nano) would sort
// wrongly against stored values. Format on the way in, always.
const sqliteTimeLayout = "2006-01-02 15:04:05"

// maxEventPageSize caps a single ListEvents page. The UI pages; the CSV export
// iterates pages. Nothing legitimately needs more rows per round trip.
const maxEventPageSize = 500

// RecordEvent appends one row to the activity log. A missing ID or timestamp is
// filled in here so callers can pass a bare Event; the caller's values win when
// present (the middleware sink already knows when its event happened).
func (s *SQLiteStorage) RecordEvent(event *models.Event) error {
	if event == nil {
		return fmt.Errorf("record event: nil event")
	}
	if event.Kind == "" {
		return fmt.Errorf("record event: kind is required")
	}
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.At.IsZero() {
		event.At = time.Now()
	}

	_, err := s.db.Exec(`
		INSERT INTO events (id, at, kind, actor_id, actor_email, share_id, link_id, ip, user_agent, detail)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.At.UTC().Format(sqliteTimeLayout), string(event.Kind),
		event.ActorID, event.ActorEmail, event.ShareID, event.LinkID,
		event.IP, event.UserAgent, event.Detail,
	)
	return err
}

// ListEvents returns one page of the activity log, newest first, plus the
// total number of rows matching the filter (so the UI can page without a
// second round trip). Limit is clamped to maxEventPageSize; 0 means "a page".
func (s *SQLiteStorage) ListEvents(filter models.EventFilter) ([]*models.Event, int, error) {
	where, args := buildEventWhere(filter)

	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM events"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > maxEventPageSize {
		limit = maxEventPageSize
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := s.db.Query(`
		SELECT id, at, kind, actor_id, actor_email, share_id, link_id, ip, user_agent, detail
		FROM events`+where+`
		ORDER BY at DESC, id DESC
		LIMIT ? OFFSET ?`, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	events := make([]*models.Event, 0, limit)
	for rows.Next() {
		e := &models.Event{}
		var kind string
		if err := rows.Scan(&e.ID, &e.At, &kind, &e.ActorID, &e.ActorEmail,
			&e.ShareID, &e.LinkID, &e.IP, &e.UserAgent, &e.Detail); err != nil {
			return nil, 0, err
		}
		e.Kind = models.EventKind(kind)
		events = append(events, e)
	}
	return events, total, rows.Err()
}

// PurgeEventsBefore deletes every row older than cutoff and reports how many
// went. Retention is the caller's policy; this only executes it.
func (s *SQLiteStorage) PurgeEventsBefore(cutoff time.Time) (int64, error) {
	res, err := s.db.Exec("DELETE FROM events WHERE at < ?", cutoff.UTC().Format(sqliteTimeLayout))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// buildEventWhere turns a filter into a WHERE clause and its bound arguments.
// Every clause is parameterised; the filter never reaches the SQL text.
func buildEventWhere(f models.EventFilter) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	add := func(clause string, arg interface{}) {
		clauses = append(clauses, clause)
		args = append(args, arg)
	}
	if f.ShareID != "" {
		add("share_id = ?", f.ShareID)
	}
	if f.LinkID != "" {
		add("link_id = ?", f.LinkID)
	}
	if f.Kind != "" {
		add("kind = ?", string(f.Kind))
	}
	if f.ActorID != "" {
		add("actor_id = ?", f.ActorID)
	}
	if !f.Since.IsZero() {
		add("at >= ?", f.Since.UTC().Format(sqliteTimeLayout))
	}
	if !f.Until.IsZero() {
		add("at < ?", f.Until.UTC().Format(sqliteTimeLayout))
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
