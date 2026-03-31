package ca

import (
	"crypto/tls"
	"fmt"
	"path/filepath"
	"testing"
)

func newTestAuthority(t *testing.T) *Authority {
	t.Helper()
	dir := t.TempDir()
	auth, err := GenerateToPath(
		filepath.Join(dir, "ca.pem"),
		filepath.Join(dir, "ca-key.pem"),
	)
	if err != nil {
		t.Fatalf("failed to generate test CA: %v", err)
	}
	return auth
}

func TestCertCache_GetCertificate(t *testing.T) {
	auth := newTestAuthority(t)
	cache := NewCertCache(auth)

	hello := &tls.ClientHelloInfo{ServerName: "api.openai.com"}
	cert, err := cache.GetCertificate(hello)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert == nil {
		t.Fatal("expected non-nil certificate")
	}
	if len(cert.Certificate) != 2 {
		t.Errorf("expected 2 certs in chain (leaf + CA), got %d", len(cert.Certificate))
	}
}

func TestCertCache_CacheHit(t *testing.T) {
	auth := newTestAuthority(t)
	cache := NewCertCache(auth)

	hello := &tls.ClientHelloInfo{ServerName: "example.com"}
	cert1, _ := cache.GetCertificate(hello)
	cert2, _ := cache.GetCertificate(hello)

	// Same pointer = cache hit
	if cert1 != cert2 {
		t.Error("expected cache hit to return same certificate")
	}
}

func TestCertCache_DifferentHosts(t *testing.T) {
	auth := newTestAuthority(t)
	cache := NewCertCache(auth)

	cert1, _ := cache.GetCertificate(&tls.ClientHelloInfo{ServerName: "a.com"})
	cert2, _ := cache.GetCertificate(&tls.ClientHelloInfo{ServerName: "b.com"})

	if cert1 == cert2 {
		t.Error("expected different certs for different hosts")
	}
}

func TestCertCache_EmptySNI(t *testing.T) {
	auth := newTestAuthority(t)
	cache := NewCertCache(auth)

	_, err := cache.GetCertificate(&tls.ClientHelloInfo{ServerName: ""})
	if err == nil {
		t.Error("expected error for empty SNI")
	}
}

func TestCertCache_Eviction(t *testing.T) {
	auth := newTestAuthority(t)
	cache := NewCertCache(auth)
	cache.maxSize = 3

	// Fill cache
	for i := 0; i < 5; i++ {
		hello := &tls.ClientHelloInfo{ServerName: fmt.Sprintf("host%d.com", i)}
		_, err := cache.GetCertificate(hello)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if len(cache.cache) > 3 {
		t.Errorf("expected cache size <= 3, got %d", len(cache.cache))
	}
}

func TestCertCache_IPAddress(t *testing.T) {
	auth := newTestAuthority(t)
	cache := NewCertCache(auth)

	cert, err := cache.GetCertificate(&tls.ClientHelloInfo{ServerName: "192.168.1.1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert == nil {
		t.Fatal("expected certificate for IP address")
	}
}
