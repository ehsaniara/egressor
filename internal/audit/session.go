// Package audit provides session logging, storage, and observation
// for intercepted proxy traffic.
package audit

import (
	"crypto/rand"
	"fmt"
	"time"
)

type Session struct {
	ID         string                `json:"session_id"`
	StartedAt  time.Time             `json:"started_at"`
	EndedAt    time.Time             `json:"ended_at,omitempty"`
	DurationMs int64                 `json:"duration_ms,omitempty"`
	ClientAddr string                `json:"client_addr"`
	TargetHost string                `json:"target_host"`
	TargetPort string                `json:"target_port"`
	DialStatus string                `json:"dial_status,omitempty"`
	Error      string                `json:"error,omitempty"`
	Exchanges  []InterceptedExchange `json:"exchanges,omitempty"`
}

// FileRef represents a file reference detected in an API payload.
type FileRef struct {
	Path   string `json:"path"`
	Source string `json:"source"` // how it was detected: "json_field", "text_pattern"
}

// InterceptedExchange records a single HTTP request/response pair captured via TLS interception.
type InterceptedExchange struct {
	Timestamp       time.Time         `json:"timestamp"`
	Method          string            `json:"method"`
	URL             string            `json:"url"`
	RequestHeaders  map[string]string `json:"request_headers,omitempty"`
	RequestBody     string            `json:"request_body,omitempty"`
	DetectedFiles   []FileRef         `json:"detected_files,omitempty"`
	Blocked         bool              `json:"blocked,omitempty"`
	BlockReason     string            `json:"block_reason,omitempty"`
	StatusCode      int               `json:"status_code"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
	ResponseBody    string            `json:"response_body,omitempty"`
}

func NewSession(clientAddr, targetHost, targetPort string) *Session {
	return &Session{
		ID:         newSessionID(),
		StartedAt:  time.Now(),
		ClientAddr: clientAddr,
		TargetHost: targetHost,
		TargetPort: targetPort,
	}
}

func (s *Session) Finish() {
	s.EndedAt = time.Now()
	s.DurationMs = s.EndedAt.Sub(s.StartedAt).Milliseconds()
}

func newSessionID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("sess_%x", b)
}
