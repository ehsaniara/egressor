package proxy

import (
	"testing"

	"github.com/ehsaniara/egressor/internal/audit/auditfakes"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		target   string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{"example.com:443", "example.com", "443", false},
		{"api.openai.com:8080", "api.openai.com", "8080", false},
		{"example.com", "example.com", "443", false},
		{"192.168.1.1:443", "192.168.1.1", "443", false},
		{":443", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			host, port, err := parseTarget(tt.target)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if host != tt.wantHost {
				t.Errorf("host: expected %q, got %q", tt.wantHost, host)
			}
			if port != tt.wantPort {
				t.Errorf("port: expected %q, got %q", tt.wantPort, port)
			}
		})
	}
}

func TestNewServer(t *testing.T) {
	fake := &auditfakes.FakeSessionSink{}
	server := NewServer("127.0.0.1:0", fake, nil)

	if server.Address() != "127.0.0.1:0" {
		t.Errorf("expected address 127.0.0.1:0, got %s", server.Address())
	}
	if server.IsRunning() {
		t.Error("server should not be running before start")
	}
}

func TestServer_StartStop(t *testing.T) {
	fake := &auditfakes.FakeSessionSink{}
	server := NewServer("127.0.0.1:0", fake, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if !server.IsRunning() {
		t.Error("expected server to be running after start")
	}

	server.Stop()
}

func TestServer_DoubleStart(t *testing.T) {
	fake := &auditfakes.FakeSessionSink{}
	server := NewServer("127.0.0.1:0", fake, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("first start: %v", err)
	}
	defer server.Stop()

	if err := server.Start(); err == nil {
		t.Error("expected error on double start")
	}
}
