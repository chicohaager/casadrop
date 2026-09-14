package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"casadrop/internal/middleware"
	"casadrop/internal/models"
)

// defaultEventPageSize is what the activity-log UI asks for per page.
const defaultEventPageSize = 50

// maxEventExportRows bounds a CSV export. It is a safety rail against an
// unbounded response, not a feature: narrow the filter for more.
const maxEventExportRows = 100_000

// eventsResponse is the JSON shape of a listing: the page plus what the client
// needs to page further.
type eventsResponse struct {
	Events []*models.Event `json:"events"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// ListEvents answers GET /api/events (admin only, enforced in the router).
// Filters: share_id, link_id, kind, actor_id, since, until (RFC 3339),
// limit, offset.
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	filter, err := eventFilterFromQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.writeEventPage(w, filter)
}

// ShareEvents answers GET /api/shares/{id}/events for the share's owner or an
// admin. The share is looked up first so a foreign or unknown ID cannot be
// told apart by whether it has events.
func (h *Handler) ShareEvents(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	share, ok := h.storage.Get(id)
	if !ok {
		http.Error(w, "Share not found or expired", http.StatusNotFound)
		return
	}
	if user := middleware.GetUserFromContext(r.Context()); user != nil && user.Role != models.RoleAdmin {
		if share.UserID != user.ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	filter, err := eventFilterFromQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	filter.ShareID = id
	h.writeEventPage(w, filter)
}

// ExportEvents answers GET /api/events/export (admin only) with the filtered
// log as CSV, newest first. Same filters as ListEvents; limit/offset are
// ignored because the export is meant to be complete.
func (h *Handler) ExportEvents(w http.ResponseWriter, r *http.Request) {
	filter, err := eventFilterFromQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="casadrop-activity-%s.csv"`,
		time.Now().UTC().Format("20060102-150405")))

	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"at", "kind", "actor_id", "actor_email", "share_id", "link_id", "ip", "user_agent", "detail"}); err != nil {
		return
	}

	// Page through the store rather than asking for everything at once; the
	// store caps a page, and streaming keeps memory flat for large exports.
	filter.Limit = 500
	for exported := 0; exported < maxEventExportRows; {
		events, _, err := h.storage.ListEvents(filter)
		if err != nil {
			// Headers are out; the honest signal left is a truncated file. Log
			// the cause so it does not vanish.
			log.Printf("Activity log: export failed at offset %d: %v", filter.Offset, err)
			return
		}
		for _, e := range events {
			if err := cw.Write([]string{
				e.At.UTC().Format(time.RFC3339), string(e.Kind), e.ActorID, e.ActorEmail,
				e.ShareID, e.LinkID, e.IP, e.UserAgent, e.Detail,
			}); err != nil {
				return
			}
		}
		exported += len(events)
		if len(events) < filter.Limit {
			break
		}
		filter.Offset += len(events)
	}
	cw.Flush()
}

func (h *Handler) writeEventPage(w http.ResponseWriter, filter models.EventFilter) {
	events, total, err := h.storage.ListEvents(filter)
	if err != nil {
		http.Error(w, "Failed to read activity log", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eventsResponse{Events: events, Total: total, Limit: filter.Limit, Offset: filter.Offset})
}

// eventFilterFromQuery parses the query string into a filter. A malformed
// number or timestamp is a 400, not a silent "no filter": a client asking for
// "since=yesterday" must not receive the whole log and think it filtered.
func eventFilterFromQuery(r *http.Request) (models.EventFilter, error) {
	q := r.URL.Query()
	filter := models.EventFilter{
		ShareID: q.Get("share_id"),
		LinkID:  q.Get("link_id"),
		Kind:    models.EventKind(q.Get("kind")),
		ActorID: q.Get("actor_id"),
		Limit:   defaultEventPageSize,
	}

	var err error
	if filter.Since, err = parseOptionalTime(q.Get("since")); err != nil {
		return filter, fmt.Errorf("since: %w", err)
	}
	if filter.Until, err = parseOptionalTime(q.Get("until")); err != nil {
		return filter, fmt.Errorf("until: %w", err)
	}
	if filter.Limit, err = parseOptionalInt(q.Get("limit"), defaultEventPageSize); err != nil {
		return filter, fmt.Errorf("limit: %w", err)
	}
	if filter.Offset, err = parseOptionalInt(q.Get("offset"), 0); err != nil {
		return filter, fmt.Errorf("offset: %w", err)
	}
	if filter.Limit <= 0 || filter.Offset < 0 {
		return filter, fmt.Errorf("limit must be positive and offset non-negative")
	}
	return filter, nil
}

func parseOptionalTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected RFC 3339 timestamp, got %q", raw)
	}
	return t, nil
}

func parseOptionalInt(raw string, def int) (int, error) {
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("expected an integer, got %q", raw)
	}
	return n, nil
}
