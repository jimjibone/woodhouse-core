package cert

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// The web listener (web-addr, serving the admin UI and grpc-web) does not use
// the pinned self-signed leaf from cert.go: that leaf is byte-exact matched
// by paired bridge clients and the iOS app, so it can never be rotated. The
// web chain instead uses its own long-lived local CA plus a short-lived leaf
// signed by that CA, so a browser (or OS) can be told to trust the CA once
// and stop warning on every visit, while the leaf itself still rotates.

// DecodeECDSAPrivKey decodes a PKCS#8 PEM-encoded ECDSA private key.
func DecodeECDSAPrivKey(priv []byte) (*ecdsa.PrivateKey, error) {
	privPEM, _ := pem.Decode(priv)
	if privPEM == nil || privPEM.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("not a PKCS#8 private key")
	}

	key, err := x509.ParsePKCS8PrivateKey(privPEM.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKCS#8 private key: %w", err)
	}

	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not ECDSA")
	}

	return ecKey, nil
}

// GenerateECDSAPrivKey generates a new P-256 ECDSA private key.
func GenerateECDSAPrivKey() (*ecdsa.PrivateKey, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

// EncodeECDSAPrivKey PEM-encodes an ECDSA private key as PKCS#8.
func EncodeECDSAPrivKey(privKey *ecdsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, err
	}

	f := bytes.NewBuffer(nil)
	err = pem.Encode(f, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	})
	if err != nil {
		return nil, err
	}
	return f.Bytes(), nil
}

// WebDNSNames returns the DNS names the web certificate should cover: the
// fixed names always reachable on this machine, plus the local hostname (and
// its ".local" mDNS form) when available. Mirrors the SAN logic in
// GenerateSelfSignedCert.
func WebDNSNames() []string {
	names := []string{"woodhouse", "localhost"}
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		names = append(names, hostname, hostname+".local")
	}

	// Dedupe while preserving order.
	seen := make(map[string]bool, len(names))
	deduped := make([]string, 0, len(names))
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		deduped = append(deduped, name)
	}
	return deduped
}

// GenerateWebCA generates a new self-signed local CA certificate for the web
// listener, returned as DER bytes.
//
// The CA is long-lived (100 years): it is only useful once a user has
// installed it as a trusted root on their own devices, and re-issuing it
// would mean redoing that installation everywhere. iOS (and most other OSes)
// will only offer to trust an installed certificate as a root if it carries
// basicConstraints CA:TRUE, which is why this is a separate CA certificate
// rather than just relaxing the leaf.
func GenerateWebCA(key *ecdsa.PrivateKey) ([]byte, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"woodhouse"},
			CommonName:   "Woodhouse Local CA",
		},
		NotBefore:             time.Now().Add(-5 * time.Minute),
		NotAfter:              time.Now().AddDate(100, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	// Go computes and sets SubjectKeyId automatically for certificates with
	// IsCA set, so it is not set explicitly here.
	cert, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	return cert, nil
}

// GenerateWebCert generates a new leaf certificate for the web listener,
// signed by ca/caKey, returned as DER bytes.
//
// Validity is capped at 365 days, well under the 825-day limit that Apple
// enforces for any TLS server certificate (see
// https://support.apple.com/en-us/103769): Safari and other Apple TLS
// clients hard-reject longer-lived leaves outright, regardless of who signed
// them. The short lifetime is fine here because, unlike the pinned leaf in
// cert.go, this one is renewed automatically by WebCertManager well before
// it expires.
//
// IP SANs are limited to the loopback addresses. LAN IPs are deliberately
// left out: they change with DHCP, and the supported way to reach the server
// from another device is by DNS name (the hostname's ".local" mDNS form).
func GenerateWebCert(ca *x509.Certificate, caKey *ecdsa.PrivateKey, leafKey *ecdsa.PrivateKey, dnsNames []string) ([]byte, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"woodhouse"},
			CommonName:   "woodhouse",
		},
		DNSNames:              dnsNames,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:             time.Now().Add(-5 * time.Minute),
		NotAfter:              time.Now().AddDate(0, 0, 365),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	cert, err := x509.CreateCertificate(rand.Reader, template, ca, &leafKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}

	return cert, nil
}
