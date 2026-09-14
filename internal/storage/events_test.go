package storage

import (
	"testing"
	"time"

	"casadrop/internal/models"
)

func recordAt(t *testing.T, store *Storage, at time.Time, kind models.EventKind, shareID string) *models.Event {
	t.Helper()
	e := &models.Event{At: at, Kind: kind, ShareID: shareID, IP: "192.0.2.10", UserAgent: "test-agent"}
	if err := store.RecordEvent(e); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	return e
}

func TestRecordEventFillsIDAndTime(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	e := &models.Event{Kind: models.EventShareDownloaded, ShareID: "abc"}
	if err := store.RecordEvent(e); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if e.ID == "" {
		t.Error("expected an ID to be assigned")
	}
	if e.At.IsZero() {
		t.Error("expected a timestamp to be assigned")
	}

	events, total, err := store.ListEvents(models.EventFilter{ShareID: "abc"})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if total != 1 || len(events) != 1 {
		t.Fatalf("expected exactly one event, got total=%d len=%d", total, len(events))
	}
	got := events[0]
	if got.Kind != models.EventShareDownloaded || got.ShareID != "abc" {
		t.Errorf("round trip lost fields: %+v", got)
	}
	// Second precision is what the column stores; the round trip must not drift
	// beyond that.
	if d := got.At.Sub(e.At.UTC()); d > time.Second || d < -time.Second {
		t.Errorf("timestamp drifted by %v", d)
	}
}

func TestRecordEventRequiresKind(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	if err := store.RecordEvent(&models.Event{ShareID: "abc"}); err == nil {
		t.Error("expected an error for an event without a kind")
	}
	if err := store.RecordEvent(nil); err == nil {
		t.Error("expected an error for a nil event")
	}
}

func TestListEventsFiltersAndPages(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	base := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	recordAt(t, store, base.Add(-3*time.Hour), models.EventShareCreated, "s1")
	recordAt(t, store, base.Add(-2*time.Hour), models.EventShareDownloaded, "s1")
	recordAt(t, store, base.Add(-1*time.Hour), models.EventShareDownloaded, "s2")
	recordAt(t, store, base, models.EventLoginFailed, "")

	t.Run("by share", func(t *testing.T) {
		events, total, err := store.ListEvents(models.EventFilter{ShareID: "s1"})
		if err != nil {
			t.Fatal(err)
		}
		if total != 2 || len(events) != 2 {
			t.Fatalf("expected 2 events for s1, got total=%d len=%d", total, len(events))
		}
		// Newest first.
		if events[0].Kind != models.EventShareDownloaded || events[1].Kind != models.EventShareCreated {
			t.Errorf("expected newest first, got %s then %s", events[0].Kind, events[1].Kind)
		}
	})

	t.Run("by kind", func(t *testing.T) {
		_, total, err := store.ListEvents(models.EventFilter{Kind: models.EventShareDownloaded})
		if err != nil {
			t.Fatal(err)
		}
		if total != 2 {
			t.Errorf("expected 2 downloads, got %d", total)
		}
	})

	t.Run("by time window", func(t *testing.T) {
		// Since is inclusive, Until exclusive — exactly the two middle rows.
		events, total, err := store.ListEvents(models.EventFilter{
			Since: base.Add(-2 * time.Hour),
			Until: base,
		})
		if err != nil {
			t.Fatal(err)
		}
		if total != 2 || len(events) != 2 {
			t.Fatalf("expected 2 events in window, got total=%d len=%d", total, len(events))
		}
		// The window is bound as a formatted string; a time.Time bound raw would
		// compare as RFC3339 text and silently return the wrong rows. Guard it:
		// the same window expressed in a non-UTC zone must give the same answer.
		berlin := time.FixedZone("CEST", 2*3600)
		_, totalLocal, err := store.ListEvents(models.EventFilter{
			Since: base.Add(-2 * time.Hour).In(berlin),
			Until: base.In(berlin),
		})
		if err != nil {
			t.Fatal(err)
		}
		if totalLocal != total {
			t.Errorf("zone-dependent result: UTC=%d local=%d", total, totalLocal)
		}
	})

	t.Run("paging", func(t *testing.T) {
		page1, total, err := store.ListEvents(models.EventFilter{Limit: 3})
		if err != nil {
			t.Fatal(err)
		}
		page2, _, err := store.ListEvents(models.EventFilter{Limit: 3, Offset: 3})
		if err != nil {
			t.Fatal(err)
		}
		if total != 4 || len(page1) != 3 || len(page2) != 1 {
			t.Errorf("paging: total=%d page1=%d page2=%d", total, len(page1), len(page2))
		}
	})
}

func TestPurgeEventsBefore(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	now := time.Now()
	recordAt(t, store, now.Add(-100*24*time.Hour), models.EventShareDownloaded, "old")
	recordAt(t, store, now.Add(-10*24*time.Hour), models.EventShareDownloaded, "recent")

	// Positive control before the purge: both rows are there.
	if _, total, _ := store.ListEvents(models.EventFilter{}); total != 2 {
		t.Fatalf("expected 2 rows before purge, got %d", total)
	}

	deleted, err := store.PurgeEventsBefore(now.Add(-90 * 24 * time.Hour))
	if err != nil {
		t.Fatalf("PurgeEventsBefore: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 row purged, got %d", deleted)
	}
	events, total, _ := store.ListEvents(models.EventFilter{})
	if total != 1 || events[0].ShareID != "recent" {
		t.Errorf("wrong row survived: total=%d first=%+v", total, events[0])
	}
}
