package handlers

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"casadrop/internal/audit"
	"casadrop/internal/middleware"
	"casadrop/internal/models"
)

// withActivityLog wires a real recorder into the handler, backed by the same
// SQLite store the handler uses, so a test reads back exactly what a download
// wrote — no fake in between.
func withActivityLog(t *testing.T, h *Handler) *audit.Recorder {
	t.Helper()
	rec := audit.New(h.storage, 0)
	h.SetAudit(rec)
	return rec
}

func saveShareWithFile(t *testing.T, h *Handler, id, name, content string) *models.Share {
	t.Helper()
	path := filepath.Join(h.storage.UploadsDir(), id+".txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	share := &models.Share{
		ID:           id,
		FileName:     id + ".txt",
		OriginalName: name,
		FileSize:     int64(len(content)),
		MimeType:     "text/plain",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		UserID:       "owner",
	}
	if err := h.storage.Save(share); err != nil {
		t.Fatalf("save share: %v", err)
	}
	return share
}

func eventsFor(t *testing.T, h *Handler, filter models.EventFilter) []*models.Event {
	t.Helper()
	events, _, err := h.storage.ListEvents(filter)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	return events
}

func TestDownloadRecordsWhoAndWhere(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	withActivityLog(t, h)
	saveShareWithFile(t, h, "dl-1", "report.txt", "hello")

	router := mux.NewRouter()
	router.HandleFunc("/d/{id}", h.DownloadFile)

	req := httptest.NewRequest("GET", "/d/dl-1", nil)
	req.RemoteAddr = "192.0.2.77:40000"
	req.Header.Set("User-Agent", "curl/8.0")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("download: %d %s", rr.Code, rr.Body.String())
	}

	events := eventsFor(t, h, models.EventFilter{ShareID: "dl-1"})
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Kind != models.EventShareDownloaded {
		t.Errorf("kind = %s", e.Kind)
	}
	if e.IP != "192.0.2.77" || e.UserAgent != "curl/8.0" || e.Detail != "report.txt" {
		t.Errorf("event lacks who/where: %+v", e)
	}
	if e.ActorID != "" {
		t.Errorf("anonymous download must not carry an actor: %+v", e)
	}
}

func TestDownloadRefusedAtLimitRecordsNothing(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	withActivityLog(t, h)
	share := saveShareWithFile(t, h, "dl-2", "capped.txt", "x")
	share.MaxDownloads = 1
	share.Downloads = 1
	if err := h.storage.Save(share); err != nil {
		t.Fatal(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/d/{id}", h.DownloadFile)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest("GET", "/d/dl-2", nil))
	if rr.Code == http.StatusOK {
		t.Fatalf("download over the limit answered 200")
	}
	if n := len(eventsFor(t, h, models.EventFilter{ShareID: "dl-2"})); n != 0 {
		t.Errorf("a refused download must not be logged as one, got %d events", n)
	}
}

func TestDeleteRecordsActor(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	withActivityLog(t, h)
	saveShareWithFile(t, h, "del-1", "gone.txt", "x")

	router := mux.NewRouter()
	router.HandleFunc("/api/shares/{id}", h.DeleteShare).Methods("DELETE")
	req := httptest.NewRequest("DELETE", "/api/shares/del-1", nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), &middleware.SessionUser{
		ID: "admin-1", Email: "admin@example.com", Role: models.RoleAdmin,
	}))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rr.Code)
	}

	events := eventsFor(t, h, models.EventFilter{ShareID: "del-1", Kind: models.EventShareDeleted})
	if len(events) != 1 || events[0].ActorEmail != "admin@example.com" || events[0].Detail != "gone.txt" {
		t.Errorf("delete not attributed: %+v", events)
	}
}

func TestExpiryRecordsSystemEvent(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	withActivityLog(t, h)
	share := saveShareWithFile(t, h, "exp-1", "old.txt", "x")
	share.ExpiresAt = time.Now().Add(-time.Hour)
	if err := h.storage.Save(share); err != nil {
		t.Fatal(err)
	}

	if !h.storage.RunExpiryCleanup() {
		t.Fatal("backend has no expiry sweep")
	}

	events := eventsFor(t, h, models.EventFilter{ShareID: "exp-1"})
	if len(events) != 1 || events[0].Kind != models.EventShareExpired || events[0].IP != "" {
		t.Errorf("expected one system expiry event, got %+v", events)
	}
}

func TestShareEventsOwnership(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	rec := withActivityLog(t, h)
	saveShareWithFile(t, h, "own-1", "mine.txt", "x")
	rec.RecordSystem(models.EventShareDownloaded, models.Event{ShareID: "own-1"})

	router := mux.NewRouter()
	router.HandleFunc("/api/shares/{id}/events", h.ShareEvents).Methods("GET")

	get := func(user *middleware.SessionUser, id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/api/shares/"+id+"/events", nil)
		if user != nil {
			req = req.WithContext(middleware.ContextWithUser(req.Context(), user))
		}
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}

	if rr := get(&middleware.SessionUser{ID: "intruder", Role: models.RoleUser}, "own-1"); rr.Code != http.StatusForbidden {
		t.Errorf("foreign user: %d, want 403", rr.Code)
	}
	if rr := get(&middleware.SessionUser{ID: "owner", Role: models.RoleUser}, "own-1"); rr.Code != http.StatusOK {
		t.Errorf("owner: %d, want 200", rr.Code)
	}
	rr := get(&middleware.SessionUser{ID: "root", Role: models.RoleAdmin}, "own-1")
	if rr.Code != http.StatusOK {
		t.Fatalf("admin: %d, want 200", rr.Code)
	}
	var page eventsResponse
	if err := json.NewDecoder(rr.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Events) != 1 || page.Events[0].ShareID != "own-1" {
		t.Errorf("unexpected page: %+v", page)
	}
	if rr := get(&middleware.SessionUser{ID: "root", Role: models.RoleAdmin}, "nope"); rr.Code != http.StatusNotFound {
		t.Errorf("unknown share: %d, want 404", rr.Code)
	}
}

func TestListEventsRejectsMalformedFilter(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	withActivityLog(t, h)

	for _, q := range []string{"since=yesterday", "limit=lots", "limit=0", "offset=-1"} {
		rr := httptest.NewRecorder()
		h.ListEvents(rr, httptest.NewRequest("GET", "/api/events?"+q, nil))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", q, rr.Code)
		}
	}
	// Positive control: a well-formed filter is accepted.
	rr := httptest.NewRecorder()
	h.ListEvents(rr, httptest.NewRequest("GET", "/api/events?since=2026-01-01T00:00:00Z&limit=10", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("valid filter: %d %s", rr.Code, rr.Body.String())
	}
}

func TestExportEventsIsCSV(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()
	rec := withActivityLog(t, h)
	rec.RecordSystem(models.EventShareDownloaded, models.Event{ShareID: "csv-1", IP: "192.0.2.9", Detail: `a "quoted" name`})
	rec.RecordSystem(models.EventLoginFailed, models.Event{IP: "192.0.2.9"})

	rr := httptest.NewRecorder()
	h.ExportEvents(rr, httptest.NewRequest("GET", "/api/events/export", nil))
	if rr.Code != http.StatusOK || !strings.HasPrefix(rr.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("export: %d %s", rr.Code, rr.Header().Get("Content-Type"))
	}
	rows, err := csv.NewReader(rr.Body).ReadAll()
	if err != nil {
		t.Fatalf("not parseable CSV: %v", err)
	}
	if len(rows) != 3 || rows[0][0] != "at" || rows[0][1] != "kind" {
		t.Fatalf("expected header + 2 rows, got %v", rows)
	}
	// Quoting survives the round trip.
	found := false
	for _, r := range rows[1:] {
		if r[8] == `a "quoted" name` {
			found = true
		}
	}
	if !found {
		t.Errorf("detail with quotes did not survive CSV: %v", rows)
	}
}
