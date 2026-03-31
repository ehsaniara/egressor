package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogger_Stdout(t *testing.T) {
	logger, err := NewLogger("json", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	if logger.writer != os.Stdout {
		t.Error("expected stdout writer for empty file path")
	}
}

func TestNewLogger_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	logger, err := NewLogger("json", path, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	sess := NewSession("127.0.0.1:1234", "example.com", "443")
	sess.Finish()
	logger.Log(sess)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	var logged Session
	if err := json.Unmarshal(data[:len(data)-1], &logged); err != nil {
		t.Fatalf("invalid JSON in log: %v", err)
	}
	if logged.TargetHost != "example.com" {
		t.Errorf("expected example.com, got %s", logged.TargetHost)
	}
}

func TestNewLogger_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "test.log")

	logger, err := NewLogger("json", path, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logger.Close()

	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}
}

func TestLogger_Rotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	// Small max size to trigger rotation quickly
	logger, err := NewLogger("json", path, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	// Write enough sessions to trigger rotation
	for i := 0; i < 10; i++ {
		sess := NewSession("127.0.0.1:1234", "example.com", "443")
		sess.Finish()
		logger.Log(sess)
	}

	// Check that rotated files exist
	entries, _ := os.ReadDir(dir)
	rotatedCount := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "audit.log.") {
			rotatedCount++
		}
	}
	if rotatedCount == 0 {
		t.Error("expected at least one rotated file")
	}

	// Current log should still exist and be writable
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("current log file should exist after rotation")
	}
}

func TestLogger_Close_Stdout(t *testing.T) {
	logger, _ := NewLogger("json", "", 0)
	if err := logger.Close(); err != nil {
		t.Errorf("closing stdout logger should not error: %v", err)
	}
}
