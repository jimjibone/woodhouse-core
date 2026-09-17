package cert

import (
	"bytes"
	"crypto/x509"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/jimjibone/woodhouse-core/shared/stores"
)

var fingerprintRE = regexp.MustCompile(`^([0-9A-F]{2}:){31}[0-9A-F]{2}$`)

func TestWebCertManagerFreshStore(t *testing.T) {
	store := stores.NewMemStore()

	m, err := NewWebCertManager(store)
	if err != nil {
		t.Fatalf("NewWebCertManager: %v", err)
	}

	// Decode the CA straight from the store to check its properties
	// independently of how the manager holds it in memory.
	caPEMBytes := m.CAPEM()
	caDER, err := decodeCertPEM(caPEMBytes)
	if err != nil {
		t.Fatalf("decodeCertPEM(CA): %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("ParseCertificate(CA): %v", err)
	}
	if !caCert.IsCA {
		t.Errorf("CA cert IsCA = false, want true")
	}
	if !caCert.BasicConstraintsValid {
		t.Errorf("CA cert BasicConstraintsValid = false, want true")
	}

	tlsCert, err := m.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate: %v", err)
	}
	if len(tlsCert.Certificate) != 2 {
		t.Fatalf("GetCertificate chain length = %d, want 2", len(tlsCert.Certificate))
	}
	leafCert := tlsCert.Leaf
	if leafCert == nil {
		t.Fatalf("GetCertificate Leaf is nil")
	}

	if max := 825 * 24 * time.Hour; leafCert.NotAfter.Sub(leafCert.NotBefore) > max {
		t.Errorf("leaf validity %s exceeds 825 days", leafCert.NotAfter.Sub(leafCert.NotBefore))
	}

	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	dnsNamesToCheck := []string{"woodhouse", "localhost"}
	if hostname, herr := os.Hostname(); herr == nil && hostname != "" {
		dnsNamesToCheck = append(dnsNamesToCheck, hostname+".local")
	}
	for _, name := range dnsNamesToCheck {
		if _, err := leafCert.Verify(x509.VerifyOptions{Roots: pool, DNSName: name}); err != nil {
			t.Errorf("leaf failed to verify for DNSName %q: %v", name, err)
		}
	}

	info := m.Info()
	if !fingerprintRE.MatchString(info.CAFingerprintSHA256) {
		t.Errorf("CAFingerprintSHA256 = %q, does not match %s", info.CAFingerprintSHA256, fingerprintRE.String())
	}
}

func TestWebCertManagerStable(t *testing.T) {
	store := stores.NewMemStore()

	m1, err := NewWebCertManager(store)
	if err != nil {
		t.Fatalf("NewWebCertManager (1st): %v", err)
	}
	ca1 := m1.CAPEM()
	cert1, err := m1.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate (1st): %v", err)
	}

	m2, err := NewWebCertManager(store)
	if err != nil {
		t.Fatalf("NewWebCertManager (2nd): %v", err)
	}
	ca2 := m2.CAPEM()
	cert2, err := m2.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate (2nd): %v", err)
	}

	if !bytes.Equal(ca1, ca2) {
		t.Errorf("CA bytes changed across manager instances on same store")
	}
	if !bytes.Equal(cert1.Certificate[0], cert2.Certificate[0]) {
		t.Errorf("leaf bytes changed across manager instances on same store")
	}
}

func TestWebCertManagerReissuesNearExpiry(t *testing.T) {
	store := stores.NewMemStore()

	m, err := NewWebCertManager(store)
	if err != nil {
		t.Fatalf("NewWebCertManager: %v", err)
	}

	caBefore := m.CAPEM()
	certBefore, err := m.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate: %v", err)
	}
	serialBefore := certBefore.Leaf.SerialNumber

	// Jump the clock to 10 days before the leaf's expiry (inside the 30 day
	// renewal window) and re-run ensure, as Run's ticker would.
	notAfter := certBefore.Leaf.NotAfter
	m.now = func() time.Time { return notAfter.Add(-10 * 24 * time.Hour) }
	if err := m.ensure(); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	caAfter := m.CAPEM()
	certAfter, err := m.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate: %v", err)
	}

	if serialBefore.Cmp(certAfter.Leaf.SerialNumber) == 0 {
		t.Errorf("leaf serial unchanged after near-expiry ensure()")
	}
	if !bytes.Equal(caBefore, caAfter) {
		t.Errorf("CA bytes changed after leaf reissue")
	}
}

func TestWebCertManagerReissuesOnDNSNamesChange(t *testing.T) {
	store := stores.NewMemStore()

	m, err := NewWebCertManager(store)
	if err != nil {
		t.Fatalf("NewWebCertManager: %v", err)
	}

	caBefore := m.CAPEM()

	newNames := []string{"woodhouse", "example-host", "example-host.local"}
	m.dnsNames = func() []string { return newNames }
	if err := m.ensure(); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	caAfter := m.CAPEM()
	if !bytes.Equal(caBefore, caAfter) {
		t.Errorf("CA bytes changed after DNS names change")
	}

	certAfter, err := m.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate: %v", err)
	}
	if !sameStringSet(certAfter.Leaf.DNSNames, newNames) {
		t.Errorf("leaf DNSNames = %v, want %v", certAfter.Leaf.DNSNames, newNames)
	}
}
