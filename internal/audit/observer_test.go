package audit_test

import (
	"testing"

	"github.com/ehsaniara/egressor/internal/audit"
	"github.com/ehsaniara/egressor/internal/audit/auditfakes"
)

func TestMultiSink(t *testing.T) {
	s1 := &auditfakes.FakeSessionSink{}
	s2 := &auditfakes.FakeSessionSink{}
	multi := audit.NewMultiSink(s1, s2)

	sess := &audit.Session{ID: "test"}
	multi.Log(sess)

	if s1.LogCallCount() != 1 {
		t.Errorf("sink 1: expected 1 call, got %d", s1.LogCallCount())
	}
	if s1.LogArgsForCall(0).ID != "test" {
		t.Error("sink 1 received wrong session")
	}
	if s2.LogCallCount() != 1 {
		t.Errorf("sink 2: expected 1 call, got %d", s2.LogCallCount())
	}
	if s2.LogArgsForCall(0).ID != "test" {
		t.Error("sink 2 received wrong session")
	}
}

func TestMultiSink_Empty(t *testing.T) {
	multi := audit.NewMultiSink()
	multi.Log(&audit.Session{ID: "test"}) // should not panic
}

func TestMultiSink_MultipleCallsTracked(t *testing.T) {
	fake := &auditfakes.FakeSessionSink{}
	multi := audit.NewMultiSink(fake)

	multi.Log(&audit.Session{ID: "a"})
	multi.Log(&audit.Session{ID: "b"})
	multi.Log(&audit.Session{ID: "c"})

	if fake.LogCallCount() != 3 {
		t.Errorf("expected 3 calls, got %d", fake.LogCallCount())
	}
	if fake.LogArgsForCall(0).ID != "a" {
		t.Errorf("call 0: expected 'a', got %s", fake.LogArgsForCall(0).ID)
	}
	if fake.LogArgsForCall(2).ID != "c" {
		t.Errorf("call 2: expected 'c', got %s", fake.LogArgsForCall(2).ID)
	}
}
