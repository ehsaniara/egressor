package proxy

import (
	"net/http"
	"testing"
)

func TestFlattenHeaders(t *testing.T) {
	h := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"text/html", "application/json"},
	}

	flat := flattenHeaders(h)
	if flat["Content-Type"] != "application/json" {
		t.Errorf("expected application/json, got %s", flat["Content-Type"])
	}
	if flat["Accept"] != "text/html, application/json" {
		t.Errorf("expected joined accept, got %s", flat["Accept"])
	}
}

func TestStripHopByHop(t *testing.T) {
	h := http.Header{
		"Connection":       {"keep-alive"},
		"Content-Type":     {"application/json"},
		"Proxy-Connection": {"keep-alive"},
		"Keep-Alive":       {"timeout=5"},
		"X-Custom":         {"value"},
	}

	stripHopByHop(h)

	if h.Get("Connection") != "" {
		t.Error("Connection header should be stripped")
	}
	if h.Get("Proxy-Connection") != "" {
		t.Error("Proxy-Connection header should be stripped")
	}
	if h.Get("Keep-Alive") != "" {
		t.Error("Keep-Alive header should be stripped")
	}
	if h.Get("Content-Type") != "application/json" {
		t.Error("Content-Type should not be stripped")
	}
	if h.Get("X-Custom") != "value" {
		t.Error("X-Custom should not be stripped")
	}
}

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		body string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello[truncated]"},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc[truncated]"},
	}

	for _, tt := range tests {
		got := truncateBody(tt.body, tt.max)
		if got != tt.want {
			t.Errorf("truncateBody(%q, %d) = %q, want %q", tt.body, tt.max, got, tt.want)
		}
	}
}

func TestLimitWriter(t *testing.T) {
	var buf []byte
	w := &limitWriter{w: &byteWriter{buf: &buf}, max: 5}

	n, err := w.Write([]byte("hel"))
	if err != nil || n != 3 {
		t.Errorf("first write: n=%d err=%v", n, err)
	}

	n, err = w.Write([]byte("lo world"))
	if err != nil {
		t.Errorf("second write: err=%v", err)
	}
	// Should only write 2 more bytes ("lo") to reach max
	if string(buf) != "hello" {
		t.Errorf("expected 'hello', got %q", string(buf))
	}

	// Further writes should be discarded
	n, err = w.Write([]byte("more"))
	if err != nil || n != 4 {
		t.Errorf("overflow write: n=%d err=%v", n, err)
	}
	if string(buf) != "hello" {
		t.Errorf("expected 'hello' after overflow, got %q", string(buf))
	}
}

type byteWriter struct {
	buf *[]byte
}

func (w *byteWriter) Write(p []byte) (int, error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}
