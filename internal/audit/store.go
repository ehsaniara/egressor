package audit

import "sync"

const defaultStoreCapacity = 1000

// SessionObserver is called when a new session is logged.
type SessionObserver func(s *Session)

// SessionStore is a thread-safe in-memory ring buffer of recent sessions.
type SessionStore struct {
	mu        sync.RWMutex
	sessions  []*Session
	capacity  int
	head      int
	count     int
	observers []SessionObserver
}

func NewSessionStore(capacity int) *SessionStore {
	if capacity <= 0 {
		capacity = defaultStoreCapacity
	}
	return &SessionStore{
		sessions: make([]*Session, capacity),
		capacity: capacity,
	}
}

// Log adds a session to the ring buffer and notifies observers.
func (s *SessionStore) Log(sess *Session) {
	s.mu.Lock()
	s.sessions[s.head] = sess
	s.head = (s.head + 1) % s.capacity
	if s.count < s.capacity {
		s.count++
	}
	observers := make([]SessionObserver, len(s.observers))
	copy(observers, s.observers)
	s.mu.Unlock()

	for _, fn := range observers {
		fn(sess)
	}
}

// OnSession registers a callback for new sessions.
func (s *SessionStore) OnSession(fn SessionObserver) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observers = append(s.observers, fn)
}

// Recent returns the most recent N sessions, newest first.
func (s *SessionStore) Recent(limit int) []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > s.count {
		limit = s.count
	}

	result := make([]*Session, limit)
	for i := 0; i < limit; i++ {
		idx := (s.head - 1 - i + s.capacity) % s.capacity
		result[i] = s.sessions[idx]
	}
	return result
}

// GetByID looks up a session by ID.
func (s *SessionStore) GetByID(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := 0; i < s.count; i++ {
		idx := (s.head - 1 - i + s.capacity) % s.capacity
		if s.sessions[idx].ID == id {
			return s.sessions[idx]
		}
	}
	return nil
}

// Stats returns aggregate statistics.
type StoreStats struct {
	TotalSessions  int `json:"total_sessions"`
	BlockedCount   int `json:"blocked_count"`
	FileDetections int `json:"file_detections"`
}

func (s *SessionStore) Stats() StoreStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var stats StoreStats
	stats.TotalSessions = s.count

	for i := 0; i < s.count; i++ {
		idx := (s.head - 1 - i + s.capacity) % s.capacity
		sess := s.sessions[idx]
		for _, ex := range sess.Exchanges {
			if ex.Blocked {
				stats.BlockedCount++
			}
			stats.FileDetections += len(ex.DetectedFiles)
		}
	}
	return stats
}
