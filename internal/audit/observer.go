package audit

// SessionSink receives completed sessions for logging or processing.
type SessionSink interface {
	Log(s *Session)
}

// MultiSink fans out sessions to multiple sinks.
type MultiSink struct {
	sinks []SessionSink
}

func NewMultiSink(sinks ...SessionSink) *MultiSink {
	return &MultiSink{sinks: sinks}
}

func (m *MultiSink) Log(s *Session) {
	for _, sink := range m.sinks {
		sink.Log(s)
	}
}
