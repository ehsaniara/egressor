package ca

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateToPath(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.pem")
	keyPath := filepath.Join(dir, "ca-key.pem")

	authority, err := GenerateToPath(certPath, keyPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if authority.Cert == nil {
		t.Fatal("expected non-nil certificate")
	}
	if authority.Key == nil {
		t.Fatal("expected non-nil key")
	}
	if authority.Cert.Subject.CommonName != "Egressor Local CA" {
		t.Errorf("expected CN 'Egressor Local CA', got %q", authority.Cert.Subject.CommonName)
	}
	if !authority.Cert.IsCA {
		t.Error("expected CA=true")
	}
	if authority.Cert.MaxPathLen != 1 {
		t.Errorf("expected MaxPathLen=1, got %d", authority.Cert.MaxPathLen)
	}

	// Verify files written
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Error("cert file not written")
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("key file not written")
	}

	// Verify key permissions
	info, _ := os.Stat(keyPath)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected key perm 0600, got %o", info.Mode().Perm())
	}
}

func TestLoadOrGenerate_Generate(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.pem")
	keyPath := filepath.Join(dir, "ca-key.pem")

	authority, err := LoadOrGenerate(certPath, keyPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authority.Cert == nil {
		t.Fatal("expected generated certificate")
	}
}

func TestLoadOrGenerate_Load(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.pem")
	keyPath := filepath.Join(dir, "ca-key.pem")

	// Generate first
	original, err := GenerateToPath(certPath, keyPath)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Load it back
	loaded, err := LoadOrGenerate(certPath, keyPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Cert.SerialNumber.Cmp(original.Cert.SerialNumber) != 0 {
		t.Error("loaded certificate has different serial number")
	}
}

func TestGenerateToPath_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "nested", "ca.pem")
	keyPath := filepath.Join(dir, "nested", "ca-key.pem")

	_, err := GenerateToPath(certPath, keyPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
