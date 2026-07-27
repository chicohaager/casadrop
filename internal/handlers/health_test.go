package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestReadyzReportsUnreadableStorage covers the only branch that distinguishes
// /readyz from /healthz — and which nothing exercised before.
//
// It also pins the fix to a proxy signal: Ping used to be db.Ping(), which the
// modernc driver answers with `select 1` without reading a page of shares.db.
// A database that had gone away still reported ready. The probe now reads a
// real row, so a closed (or deleted, or truncated) database fails it.
func TestReadyzReportsUnreadableStorage(t *testing.T) {
	h, store, _ := setupRealTemplateHandler(t)

	// Healthy first, so a later 503 cannot be blamed on the harness.
	rec := httptest.NewRecorder()
	h.Readyz(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthy /readyz: status %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); body != "ready" {
		t.Errorf("healthy /readyz body = %q, want %q", body, "ready")
	}

	// Now take the storage away underneath it.
	store.Close()

	rec = httptest.NewRecorder()
	h.Readyz(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/readyz with unusable storage: status %d, want 503 — the probe is not measuring the database", rec.Code)
	}
	if body := rec.Body.String(); body != "storage unavailable" {
		t.Errorf("/readyz body = %q, want %q", body, "storage unavailable")
	}

	// Liveness must NOT follow readiness down: that split is the whole reason
	// the container healthcheck can stay on /healthz without risking a restart
	// loop when the database is merely busy.
	rec = httptest.NewRecorder()
	h.Healthz(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("/healthz with unusable storage: status %d, want 200 (liveness is not readiness)", rec.Code)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Errorf("/healthz body = %q, want %q", body, "ok")
	}
}

// TestReadyzQueriesTheRealTable proves the probe touches application data
// rather than answering from thin air: dropping the table it reads must turn
// readiness red. `select 1` — what db.Ping() issues — would sail straight
// through this.
//
// LIMITATION, measured and recorded rather than glossed over: this check cannot
// detect a database file clobbered *underneath* an open connection. SQLite
// answers the query from its warm page cache, so a shares.db overwritten with
// garbage still reports ready until something forces an uncached read. A probe
// on an already-open handle is inherently a weaker witness than reopening the
// file, and reopening on every probe is not something a 30-second healthcheck
// should do. What this does cover: a closed pool, a missing/renamed table, a
// lock that outlives the timeout, and I/O errors on any page not already cached.
func TestReadyzQueriesTheRealTable(t *testing.T) {
	h, store, _ := setupRealTemplateHandler(t)
	defer store.Close()

	rec := httptest.NewRecorder()
	h.Readyz(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("baseline /readyz: status %d, want 200", rec.Code)
	}

	if err := store.DropSharesTableForTest(); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	rec = httptest.NewRecorder()
	h.Readyz(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/readyz after dropping the shares table: status %d, want 503 — the probe is not querying application data", rec.Code)
	}
}
