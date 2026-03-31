package ca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"
)

const defaultCacheSize = 1024

// CertCache generates and caches TLS leaf certificates signed by a CA.
type CertCache struct {
	mu      sync.Mutex
	ca      *Authority
	cache   map[string]*cacheEntry
	order   []string
	maxSize int
}

type cacheEntry struct {
	tlsCert   *tls.Certificate
	expiresAt time.Time
}

// NewCertCache creates a certificate cache backed by the given CA.
func NewCertCache(ca *Authority) *CertCache {
	return &CertCache{
		ca:      ca,
		cache:   make(map[string]*cacheEntry),
		maxSize: defaultCacheSize,
	}
}

// GetCertificate returns a TLS certificate for the given ClientHello.
// This matches the tls.Config.GetCertificate signature.
func (cc *CertCache) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	hostname := hello.ServerName
	if hostname == "" {
		return nil, fmt.Errorf("no SNI in ClientHello")
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	if entry, ok := cc.cache[hostname]; ok && time.Now().Before(entry.expiresAt) {
		return entry.tlsCert, nil
	}

	cert, err := cc.generateLeaf(hostname)
	if err != nil {
		return nil, err
	}

	// Evict oldest if at capacity
	if len(cc.cache) >= cc.maxSize {
		oldest := cc.order[0]
		cc.order = cc.order[1:]
		delete(cc.cache, oldest)
	}

	cc.cache[hostname] = &cacheEntry{
		tlsCert:   cert,
		expiresAt: time.Now().Add(24 * time.Hour),
	}
	cc.order = append(cc.order, hostname)

	return cert, nil
}

func (cc *CertCache) generateLeaf(hostname string) (*tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating leaf key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generating serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: hostname,
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	if ip := net.ParseIP(hostname); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{hostname}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, cc.ca.Cert, &key.PublicKey, cc.ca.Key)
	if err != nil {
		return nil, fmt.Errorf("creating leaf certificate: %w", err)
	}

	tlsCert := &tls.Certificate{
		Certificate: [][]byte{certDER, cc.ca.Cert.Raw},
		PrivateKey:  key,
	}

	return tlsCert, nil
}
