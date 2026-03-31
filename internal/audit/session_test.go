package audit

import (
	"strings"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	sess := NewSession("127.0.0.1:5000", "api.openai.com", "443")

	if sess.ClientAddr != "127.0.0.1:5000" {
		t.Errorf("expected client addr 127.0.0.1:5000, got %s", sess.ClientAddr)
	}
	if sess.TargetHost != "api.openai.com" {
		t.Errorf("expected target host api.openai.com, got %s", sess.TargetHost)
	}
	if sess.TargetPort != "443" {
		t.Errorf("expected target port 443, got %s", sess.TargetPort)
	}
	if !strings.HasPrefix(sess.ID, "sess_") {
		t.Errorf("expected session ID prefix sess_, got %s", sess.ID)
	}
	if sess.StartedAt.IsZero() {
		t.Error("expected non-zero start time")
	}
}

func TestSessionIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		sess := NewSession("", "", "")
		if ids[sess.ID] {
			t.Fatalf("duplicate session ID: %s", sess.ID)
		}
		ids[sess.ID] = true
	}
}

func TestSessionFinish(t *testing.T) {
	sess := NewSession("", "example.com", "443")
	time.Sleep(5 * time.Millisecond)
	sess.Finish()

	if sess.EndedAt.IsZero() {
		t.Error("expected non-zero end time")
	}
	if sess.DurationMs < 1 {
		t.Errorf("expected duration >= 1ms, got %d", sess.DurationMs)
	}
	if sess.EndedAt.Before(sess.StartedAt) {
		t.Error("end time before start time")
	}
}
