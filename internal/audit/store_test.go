package audit

import (
	"fmt"
	"sync"
	"testing"
)

func TestSessionStore_AddAndRecent(t *testing.T) {
	store := NewSessionStore(5)

	for i := 0; i < 3; i++ {
		store.Log(&Session{ID: fmt.Sprintf("sess_%d", i)})
	}

	recent := store.Recent(10)
	if len(recent) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(recent))
	}
	// Newest first
	if recent[0].ID != "sess_2" {
		t.Errorf("expected newest first (sess_2), got %s", recent[0].ID)
	}
	if recent[2].ID != "sess_0" {
		t.Errorf("expected oldest last (sess_0), got %s", recent[2].ID)
	}
}

func TestSessionStore_RingBufferWrap(t *testing.T) {
	store := NewSessionStore(3)

	for i := 0; i < 5; i++ {
		store.Log(&Session{ID: fmt.Sprintf("sess_%d", i)})
	}

	recent := store.Recent(10)
	if len(recent) != 3 {
		t.Fatalf("expected 3 sessions (capacity), got %d", len(recent))
	}
	// Should have sessions 4, 3, 2 (oldest 0, 1 evicted)
	if recent[0].ID != "sess_4" {
		t.Errorf("expected sess_4, got %s", recent[0].ID)
	}
	if recent[2].ID != "sess_2" {
		t.Errorf("expected sess_2, got %s", recent[2].ID)
	}
}

func TestSessionStore_GetByID(t *testing.T) {
	store := NewSessionStore(10)
	store.Log(&Session{ID: "target"})
	store.Log(&Session{ID: "other"})

	found := store.GetByID("target")
	if found == nil || found.ID != "target" {
		t.Error("expected to find session by ID")
	}

	notFound := store.GetByID("nonexistent")
	if notFound != nil {
		t.Error("expected nil for nonexistent ID")
	}
}

func TestSessionStore_Observer(t *testing.T) {
	store := NewSessionStore(10)
	var received []*Session

	store.OnSession(func(s *Session) {
		received = append(received, s)
	})

	store.Log(&Session{ID: "a"})
	store.Log(&Session{ID: "b"})

	if len(received) != 2 {
		t.Fatalf("expected 2 observer calls, got %d", len(received))
	}
	if received[0].ID != "a" || received[1].ID != "b" {
		t.Error("observer received wrong sessions")
	}
}

func TestSessionStore_Stats(t *testing.T) {
	store := NewSessionStore(10)

	store.Log(&Session{
		ID: "s1",
		Exchanges: []InterceptedExchange{
			{Blocked: true, DetectedFiles: []FileRef{{Path: "a.env"}}},
		},
	})
	store.Log(&Session{
		ID: "s2",
		Exchanges: []InterceptedExchange{
			{DetectedFiles: []FileRef{{Path: "b.go"}, {Path: "c.go"}}},
		},
	})

	stats := store.Stats()
	if stats.TotalSessions != 2 {
		t.Errorf("expected 2 total, got %d", stats.TotalSessions)
	}
	if stats.BlockedCount != 1 {
		t.Errorf("expected 1 blocked, got %d", stats.BlockedCount)
	}
	if stats.FileDetections != 3 {
		t.Errorf("expected 3 file detections, got %d", stats.FileDetections)
	}
}

func TestSessionStore_RecentLimit(t *testing.T) {
	store := NewSessionStore(10)
	for i := 0; i < 5; i++ {
		store.Log(&Session{ID: fmt.Sprintf("s%d", i)})
	}

	recent := store.Recent(2)
	if len(recent) != 2 {
		t.Fatalf("expected 2, got %d", len(recent))
	}
}

func TestSessionStore_DefaultCapacity(t *testing.T) {
	store := NewSessionStore(0)
	if store.capacity != defaultStoreCapacity {
		t.Errorf("expected default capacity %d, got %d", defaultStoreCapacity, store.capacity)
	}
}

func TestSessionStore_ConcurrentAccess(t *testing.T) {
	store := NewSessionStore(100)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			store.Log(&Session{ID: fmt.Sprintf("s%d", id)})
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Recent(10)
			store.Stats()
		}()
	}

	wg.Wait()
}
