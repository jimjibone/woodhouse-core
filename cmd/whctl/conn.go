package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/schema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	defaultPort = "4000"
	// renewThresholdDays asks the server to rotate the refresh token once it
	// has fewer than this many days left (it lasts 30).
	renewThresholdDays = 7
	// accessTokenSlack refreshes the access token this long before it expires
	// so a call does not race the expiry.
	accessTokenSlack = 60 * time.Second
)

var schemaVersion = strconv.Itoa(schema.Version())

// normaliseServer adds the default API port when the address has none.
func normaliseServer(addr string) string {
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return net.JoinHostPort(strings.Trim(addr, "[]"), defaultPort)
}

// certFingerprint is the lowercase hex SHA-256 of a DER certificate, the same
// form core advertises as "fp" in its discovery TXT record.
func certFingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:])
}

// normaliseFingerprint accepts hex with or without colons, in either case.
func normaliseFingerprint(fp string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(fp), ":", ""))
}

// probeFingerprint connects to addr and returns the fingerprint of the leaf
// certificate it presents, without trusting it.
func probeFingerprint(ctx context.Context, addr string) (string, error) {
	dialer := &tls.Dialer{Config: &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"h2"}}}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", fmt.Errorf("connect to %s: %w", addr, err)
	}
	defer conn.Close()
	certs := conn.(*tls.Conn).ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return "", fmt.Errorf("%s presented no certificate", addr)
	}
	return certFingerprint(certs[0].Raw), nil
}

// pinnedTLS trusts exactly the leaf certificate with the given fingerprint.
// Core's certificate is self-signed, so chain verification is replaced by the
// pin. With insecure set no check is made at all.
func pinnedTLS(fingerprint string, insecure bool) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
			if insecure {
				return nil
			}
			if len(raw) == 0 {
				return errors.New("server presented no certificate")
			}
			got := certFingerprint(raw[0])
			if fingerprint == "" {
				return fmt.Errorf("no certificate pinned for this server (it presented %s); run 'whctl login' to pin it", got)
			}
			if got != fingerprint {
				return fmt.Errorf("server certificate changed: presented %s, pinned %s; if this is expected run 'whctl login --fingerprint %s'", got, fingerprint, got)
			}
			return nil
		},
	}
}

// Session is a connection to one profile's server.
type Session struct {
	cfg     *Config
	name    string
	prof    *Profile
	conn    *grpc.ClientConn
	timeout time.Duration
}

// Dial connects to the profile's server. Nothing is sent until the first
// call.
func Dial(cfg *Config, name string, prof *Profile, insecure bool, timeout time.Duration) (*Session, error) {
	s := &Session{cfg: cfg, name: name, prof: prof, timeout: timeout}
	conn, err := grpc.NewClient(prof.Server,
		grpc.WithTransportCredentials(credentials.NewTLS(pinnedTLS(prof.Fingerprint, insecure))),
		grpc.WithChainUnaryInterceptor(s.unaryMetadata),
		grpc.WithChainStreamInterceptor(s.streamMetadata),
	)
	if err != nil {
		return nil, err
	}
	s.conn = conn
	return s, nil
}

func (s *Session) Close() error { return s.conn.Close() }

// withMetadata adds the schema version every call needs and, when logged in,
// the access token. Core ignores the token on methods that need none.
func (s *Session) withMetadata(ctx context.Context) context.Context {
	kv := []string{schema.HeaderName, schemaVersion}
	if s.prof.AccessToken != "" {
		kv = append(kv, "authorization", s.prof.AccessToken)
	}
	return metadata.AppendToOutgoingContext(ctx, kv...)
}

func (s *Session) unaryMetadata(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return invoker(s.withMetadata(ctx), method, req, reply, cc, opts...)
}

func (s *Session) streamMetadata(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return streamer(s.withMetadata(ctx), desc, cc, method, opts...)
}

// tokenExpiry returns the exp claim of a JWT without verifying it; the server
// does the verifying, this only decides when to refresh.
func tokenExpiry(token string) (time.Time, bool) {
	claims := jwt.RegisteredClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(token, &claims); err != nil || claims.ExpiresAt == nil {
		return time.Time{}, false
	}
	return claims.ExpiresAt.Time, true
}

// EnsureAccess makes sure the session holds an access token that is not about
// to expire, refreshing it (and saving the result) when needed or forced.
func (s *Session) EnsureAccess(ctx context.Context, force bool) error {
	if !force && s.prof.AccessToken != "" {
		if exp, ok := tokenExpiry(s.prof.AccessToken); ok && time.Until(exp) > accessTokenSlack {
			return nil
		}
	}
	if s.prof.RefreshToken == "" {
		return fmt.Errorf("profile %q is not logged in; run 'whctl login'", s.name)
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	s.prof.AccessToken = ""
	res, err := clientsapi.NewUserAuthServiceClient(s.conn).Refresh(ctx, &clientsapi.UserRefreshRequest{
		RefreshToken:   s.prof.RefreshToken,
		RenewThreshold: renewThresholdDays,
	})
	if err != nil {
		if status.Code(err) == codes.Unauthenticated {
			s.prof.RefreshToken = ""
			_ = s.cfg.Save()
			return fmt.Errorf("session for profile %q has expired or was revoked; run 'whctl login': %w", s.name, err)
		}
		return fmt.Errorf("refresh session: %w", err)
	}
	s.prof.AccessToken = res.AccessToken
	if res.RefreshToken != "" {
		s.prof.RefreshToken = res.RefreshToken
	}
	return s.cfg.Save()
}
