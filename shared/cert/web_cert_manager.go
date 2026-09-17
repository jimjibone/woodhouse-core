package cert

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jimjibone/log"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

var (
	storeWebCAPath    = "web-ca.crt"
	storeWebCAKeyPath = "web-ca.key"
	storeWebCertPath  = "web.crt"
	storeWebKeyPath   = "web.key"
)

const (
	// webCertRenewBefore is how far ahead of expiry the leaf is reissued.
	webCertRenewBefore = 30 * 24 * time.Hour
	// webCertRecheckEvery is how often Run polls for a leaf that needs
	// reissuing (expiry, or a DNS name set that changed under it).
	webCertRecheckEvery = 24 * time.Hour
)

// WebCertInfo describes the current web certificate chain, for surfacing to
// an operator (e.g. via an unauthenticated /api/trust/info endpoint) so they
// know what they're about to install.
type WebCertInfo struct {
	CASubject           string    `json:"caSubject"`
	CAFingerprintSHA256 string    `json:"caFingerprintSha256"`
	CANotAfter          time.Time `json:"caNotAfter"`
	DNSNames            []string  `json:"dnsNames"`
	LeafNotAfter        time.Time `json:"leafNotAfter"`
}

// WebCertManager owns the certificate chain served by the web listener: a
// long-lived local CA plus a short-lived leaf signed by it. Unlike
// CertManager's pinned leaf, both are stored so that the leaf can be renewed
// automatically without invalidating a CA a user has already installed as a
// trusted root.
type WebCertManager struct {
	store stores.Store

	// Injectable for tests; default to time.Now and WebDNSNames.
	now      func() time.Time
	dnsNames func() []string

	mu    sync.RWMutex
	cert  *tls.Certificate
	caPEM []byte
	info  WebCertInfo
}

// NewWebCertManager loads (or creates) the web CA and leaf certificate in
// store, and returns a manager serving them.
func NewWebCertManager(store stores.Store) (*WebCertManager, error) {
	m := &WebCertManager{
		store:    store,
		now:      time.Now,
		dnsNames: WebDNSNames,
	}

	if err := m.ensure(); err != nil {
		return nil, err
	}

	return m, nil
}

// GetCertificate implements the signature required by tls.Config's
// GetCertificate field. It always returns the currently active chain
// (leaf + CA), refreshed in the background by Run.
func (m *WebCertManager) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cert, nil
}

// CAPEM returns the PEM-encoded local CA certificate, for offering to
// devices to install as a trusted root.
func (m *WebCertManager) CAPEM() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.caPEM
}

// Info returns a snapshot of the current chain's metadata.
func (m *WebCertManager) Info() WebCertInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.info
}

// Run periodically re-checks the leaf certificate, reissuing it once it
// nears expiry or the wanted DNS names change. It blocks until ctx is done.
func (m *WebCertManager) Run(ctx context.Context) {
	ticker := time.NewTicker(webCertRecheckEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.ensure(); err != nil {
				log.Errorf("failed to refresh web certificate: %s", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// ensure loads the CA and leaf from store, generating or reissuing whichever
// part is missing or no longer good, then publishes the resulting chain.
func (m *WebCertManager) ensure() error {
	caKey, caCert, caDER, caPEM, err := m.loadOrCreateCA()
	if err != nil {
		return err
	}

	wantDNSNames := m.dnsNames()

	leafKey, leafCert, leafDER, err := m.loadOrIssueLeaf(caCert, caKey, wantDNSNames)
	if err != nil {
		return err
	}

	tlsCert := &tls.Certificate{
		// Send the full chain so clients that haven't cached the CA
		// separately can still build a path to it.
		Certificate: [][]byte{leafDER, caDER},
		PrivateKey:  leafKey,
		Leaf:        leafCert,
	}

	m.mu.Lock()
	m.cert = tlsCert
	m.caPEM = caPEM
	m.info = WebCertInfo{
		CASubject:           caCert.Subject.CommonName,
		CAFingerprintSHA256: fingerprintSHA256(caDER),
		CANotAfter:          caCert.NotAfter,
		DNSNames:            append([]string(nil), leafCert.DNSNames...),
		LeafNotAfter:        leafCert.NotAfter,
	}
	m.mu.Unlock()

	return nil
}

// loadOrCreateCA loads the existing web CA from store, or generates and
// saves a new one if either the key or cert is missing. An existing CA is
// never regenerated: doing so would invalidate it on every device a user has
// already installed it on.
func (m *WebCertManager) loadOrCreateCA() (*ecdsa.PrivateKey, *x509.Certificate, []byte, []byte, error) {
	if m.store.Has(storeWebCAPath) && m.store.Has(storeWebCAKeyPath) {
		keyData, err := m.store.Get(storeWebCAKeyPath)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to load web CA key: %w", err)
		}
		caKey, err := DecodeECDSAPrivKey(keyData)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to decode web CA key: %w", err)
		}

		caPEM, err := m.store.Get(storeWebCAPath)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to load web CA cert: %w", err)
		}
		caDER, err := decodeCertPEM(caPEM)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to decode web CA cert: %w", err)
		}
		caCert, err := x509.ParseCertificate(caDER)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to parse web CA cert: %w", err)
		}

		return caKey, caCert, caDER, caPEM, nil
	}

	log.Infof("generating new web local CA")

	caKey, err := GenerateECDSAPrivKey()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to generate web CA key: %w", err)
	}
	keyData, err := EncodeECDSAPrivKey(caKey)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to encode web CA key: %w", err)
	}
	if err := m.store.Set(storeWebCAKeyPath, keyData); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to save web CA key: %w", err)
	}

	caDER, err := GenerateWebCA(caKey)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to generate web CA cert: %w", err)
	}
	caPEM, err := EncodeCert(caDER)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to encode web CA cert: %w", err)
	}
	if err := m.store.Set(storeWebCAPath, caPEM); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to save web CA cert: %w", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to parse generated web CA cert: %w", err)
	}

	return caKey, caCert, caDER, caPEM, nil
}

// loadOrIssueLeaf loads the existing web leaf from store, reissuing it (with
// a new key) if it is missing, unparseable, not signed by caCert, nearing
// expiry, or covers a different set of DNS names than wanted.
func (m *WebCertManager) loadOrIssueLeaf(caCert *x509.Certificate, caKey *ecdsa.PrivateKey, wantDNSNames []string) (*ecdsa.PrivateKey, *x509.Certificate, []byte, error) {
	leafKey, leafCert, leafDER, err := m.loadLeaf()
	var reason string
	switch {
	case err != nil:
		reason = err.Error()
	case leafCert.CheckSignatureFrom(caCert) != nil:
		reason = "not signed by current CA"
	case leafCert.NotAfter.Before(m.now().Add(webCertRenewBefore)):
		reason = "nearing expiry"
	case !sameStringSet(leafCert.DNSNames, wantDNSNames):
		reason = "DNS names changed"
	default:
		return leafKey, leafCert, leafDER, nil
	}

	log.Infof("issuing new web certificate: %s", reason)

	leafKey, err = GenerateECDSAPrivKey()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate web leaf key: %w", err)
	}
	keyData, err := EncodeECDSAPrivKey(leafKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to encode web leaf key: %w", err)
	}
	if err := m.store.Set(storeWebKeyPath, keyData); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to save web leaf key: %w", err)
	}

	leafDER, err = GenerateWebCert(caCert, caKey, leafKey, wantDNSNames)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate web leaf cert: %w", err)
	}
	certPEM, err := EncodeCert(leafDER)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to encode web leaf cert: %w", err)
	}
	if err := m.store.Set(storeWebCertPath, certPEM); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to save web leaf cert: %w", err)
	}
	leafCert, err = x509.ParseCertificate(leafDER)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse generated web leaf cert: %w", err)
	}

	return leafKey, leafCert, leafDER, nil
}

// loadLeaf loads the stored web leaf key and certificate. Any failure is a
// reason to reissue rather than a fatal error, so the message is kept short.
func (m *WebCertManager) loadLeaf() (*ecdsa.PrivateKey, *x509.Certificate, []byte, error) {
	if !m.store.Has(storeWebCertPath) || !m.store.Has(storeWebKeyPath) {
		return nil, nil, nil, fmt.Errorf("missing")
	}

	keyData, err := m.store.Get(storeWebKeyPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load leaf key: %w", err)
	}
	leafKey, err := DecodeECDSAPrivKey(keyData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode leaf key: %w", err)
	}

	certPEM, err := m.store.Get(storeWebCertPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load leaf cert: %w", err)
	}
	leafDER, err := decodeCertPEM(certPEM)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode leaf cert: %w", err)
	}
	leafCert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse leaf cert: %w", err)
	}

	return leafKey, leafCert, leafDER, nil
}

// decodeCertPEM extracts the DER bytes from a single PEM-encoded certificate.
func decodeCertPEM(data []byte) ([]byte, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("not a PEM certificate")
	}
	return block.Bytes, nil
}

// sameStringSet reports whether a and b contain the same strings, ignoring
// order and duplicates.
func sameStringSet(a, b []string) bool {
	return subset(a, b) && subset(b, a)
}

func subset(a, b []string) bool {
	set := make(map[string]bool, len(b))
	for _, s := range b {
		set[s] = true
	}
	for _, s := range a {
		if !set[s] {
			return false
		}
	}
	return true
}

// fingerprintSHA256 returns the uppercase, colon-separated hex SHA-256
// fingerprint of der, as commonly displayed by browsers and OSes for
// certificate trust decisions.
func fingerprintSHA256(der []byte) string {
	sum := sha256.Sum256(der)
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(parts, ":")
}
