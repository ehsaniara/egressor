package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("{}"), 0o644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ListenAddress != "127.0.0.1:8080" {
		t.Errorf("expected default listen address, got %s", cfg.ListenAddress)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("expected json format, got %s", cfg.Logging.Format)
	}
	if cfg.Intercept.MaxBodySize != 65536 {
		t.Errorf("expected default max body size 65536, got %d", cfg.Intercept.MaxBodySize)
	}
	if cfg.Logging.MaxSizeMB != 2 {
		t.Errorf("expected default max log size 2MB, got %d", cfg.Logging.MaxSizeMB)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := `
listen_address: "0.0.0.0:9090"
policy:
  deny_file_patterns:
    - "*.env"
    - "*.pem"
intercept:
  log_body: true
  max_body_size: 1048576
logging:
  file: /tmp/test.log
  max_size_mb: 5
`
	os.WriteFile(path, []byte(yaml), 0o644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ListenAddress != "0.0.0.0:9090" {
		t.Errorf("expected 0.0.0.0:9090, got %s", cfg.ListenAddress)
	}
	if len(cfg.Policy.DenyFilePatterns) != 2 {
		t.Errorf("expected 2 deny patterns, got %d", len(cfg.Policy.DenyFilePatterns))
	}
	if !cfg.Intercept.LogBody {
		t.Error("expected log_body true")
	}
	if cfg.Intercept.MaxBodySize != 1048576 {
		t.Errorf("expected 1048576, got %d", cfg.Intercept.MaxBodySize)
	}
	if cfg.Logging.MaxSizeMB != 5 {
		t.Errorf("expected 5, got %d", cfg.Logging.MaxSizeMB)
	}
}

func TestLoad_ExpandHome(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := `
intercept:
  ca_cert: ~/.egressor/ca.pem
  ca_key: ~/.egressor/ca-key.pem
`
	os.WriteFile(path, []byte(yaml), 0o644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".egressor", "ca.pem")
	if cfg.Intercept.CACert != expected {
		t.Errorf("expected %s, got %s", expected, cfg.Intercept.CACert)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("{{invalid"), 0o644)

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")

	cfg := &Config{
		ListenAddress: "127.0.0.1:8080",
		Policy: PolicyConfig{
			DenyFilePatterns: []string{"*.env", "*.pem"},
		},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Load it back
	data, _ := os.ReadFile(path)
	if len(data) == 0 {
		t.Fatal("expected non-empty file")
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("failed to reload: %v", err)
	}
	if len(loaded.Policy.DenyFilePatterns) != 2 {
		t.Errorf("expected 2 patterns after reload, got %d", len(loaded.Policy.DenyFilePatterns))
	}
}
