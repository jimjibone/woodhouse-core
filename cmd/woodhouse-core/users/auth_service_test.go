package users

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/limits"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// fakeClock lets tests advance the login limiter's clock deterministically,
// without sleeping (each real login call already costs a 64 MiB argon2id
// run, so keep the number of those modest per test).
type fakeClock struct {
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time          { return c.now }
func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// newTestAuthService builds a fully wired AuthService against an
// in-memory store, along with the underlying UserManager (for seeding
// users directly, bypassing the bootstrap login flow) and the fake
// clock driving cfg's limiter.
func newTestAuthService(t *testing.T, cfg limits.Config) (*AuthService, *core.UserManager, *fakeClock) {
	t.Helper()

	store := stores.NewMemStore()

	userManager, err := core.NewUserManager(store)
	if err != nil {
		t.Fatalf("failed to create user manager: %s", err)
	}
	t.Cleanup(userManager.Close)

	jwtManager, err := NewJWTManager(store)
	if err != nil {
		t.Fatalf("failed to create jwt manager: %s", err)
	}
	t.Cleanup(jwtManager.Close)

	clock := newFakeClock()
	cfg.Now = clock.Now
	lim := limits.NewLogin(cfg)

	srv := NewAuthService(userManager, jwtManager, lim)
	return srv, userManager, clock
}

// seedAdmin adds an admin user directly via the UserManager, bypassing
// the login endpoint's bootstrap path (and its extra argon2id run).
func seedAdmin(t *testing.T, userManager *core.UserManager, username, password string) {
	t.Helper()
	admin, err := core.NewUser(username, "", password, auth.AdminRole)
	if err != nil {
		t.Fatalf("failed to build admin user: %s", err)
	}
	if err := userManager.Store(admin); err != nil {
		t.Fatalf("failed to store admin user: %s", err)
	}
}

// ctxWithIP returns a context carrying a gRPC peer with the given IP, as
// peerAddr expects to find via peer.FromContext.
func ctxWithIP(ip string) context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP(ip), Port: 12345},
	})
}

func TestLogin_BootstrapThenSecondLogin(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{})
	ctx := ctxWithIP("10.0.0.1")
	req := &clientsapi.UserLoginRequest{Username: "admin", Password: "adminpass1"}

	if userManager.HasAnAdmin() {
		t.Fatalf("expected no admin before first login")
	}

	res, err := srv.Login(ctx, req)
	if err != nil {
		t.Fatalf("bootstrap login: unexpected error: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatalf("bootstrap login: expected tokens, got %+v", res)
	}
	if !userManager.HasAnAdmin() {
		t.Fatalf("expected an admin to exist after bootstrap login")
	}

	res, err = srv.Login(ctx, req)
	if err != nil {
		t.Fatalf("second login: unexpected error: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatalf("second login: expected tokens, got %+v", res)
	}
}

func TestLogin_UnknownUserAndWrongPasswordReturnSameError(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{LoginBurst: 20, LoginPerMin: 20 * 60})
	seedAdmin(t, userManager, "admin", "adminpass1")
	ctx := ctxWithIP("10.0.0.2")

	_, err := srv.Login(ctx, &clientsapi.UserLoginRequest{Username: "nosuchuser", Password: "whatever1"})
	if err == nil {
		t.Fatalf("expected error for unknown user")
	}
	stUnknown := status.Convert(err)

	_, err = srv.Login(ctx, &clientsapi.UserLoginRequest{Username: "admin", Password: "wrongpassword1"})
	if err == nil {
		t.Fatalf("expected error for wrong password")
	}
	stWrongPassword := status.Convert(err)

	if stUnknown.Code() != codes.Unauthenticated {
		t.Fatalf("unknown user: expected codes.Unauthenticated, got %s", stUnknown.Code())
	}
	if stWrongPassword.Code() != codes.Unauthenticated {
		t.Fatalf("wrong password: expected codes.Unauthenticated, got %s", stWrongPassword.Code())
	}
	if stUnknown.Message() != "incorrect username/password" {
		t.Fatalf("unknown user: unexpected message %q", stUnknown.Message())
	}
	if stWrongPassword.Message() != "incorrect username/password" {
		t.Fatalf("wrong password: unexpected message %q", stWrongPassword.Message())
	}
	if stUnknown.Message() != stWrongPassword.Message() {
		t.Fatalf("expected identical messages, got %q vs %q", stUnknown.Message(), stWrongPassword.Message())
	}
}

func TestLogin_AccountBackoff(t *testing.T) {
	srv, userManager, clock := newTestAuthService(t, limits.Config{LoginBurst: 20, LoginPerMin: 20 * 60})
	seedAdmin(t, userManager, "admin", "adminpass1")
	ctx := ctxWithIP("10.0.0.3")

	// Three wrong passwords: the first two are free (BackoffFree=2
	// default), the third pushes the account into backoff.
	for i := 0; i < 3; i++ {
		_, err := srv.Login(ctx, &clientsapi.UserLoginRequest{Username: "admin", Password: "wrongpassword1"})
		if err == nil {
			t.Fatalf("attempt %d: expected wrong-password error", i)
		}
	}

	// Even the correct password is rejected while backed off, and
	// without ever reaching the password check.
	_, err := srv.Login(ctx, &clientsapi.UserLoginRequest{Username: "admin", Password: "adminpass1"})
	if err == nil {
		t.Fatalf("expected account backoff to reject the correct password")
	}
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("expected codes.ResourceExhausted, got %s", status.Code(err))
	}

	// Advance the clock past the backoff window and retry.
	clock.Advance(2 * time.Second)
	res, err := srv.Login(ctx, &clientsapi.UserLoginRequest{Username: "admin", Password: "adminpass1"})
	if err != nil {
		t.Fatalf("expected login to succeed after backoff window, got: %v", err)
	}
	if res.AccessToken == "" {
		t.Fatalf("expected an access token after successful login")
	}

	// The failure counter should have been reset by RecordSuccess.
	if wait := srv.limits.AccountRetryIn("admin"); wait != 0 {
		t.Fatalf("expected account backoff cleared after success, got wait %s", wait)
	}
}

func TestLogin_IPLimitRejectsBeforeCredentialCheck(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{LoginBurst: 2, LoginPerMin: 2 * 60})
	seedAdmin(t, userManager, "admin", "adminpass1")

	ip1 := ctxWithIP("10.0.0.4")
	ip2 := ctxWithIP("10.0.0.5")
	req := &clientsapi.UserLoginRequest{Username: "admin", Password: "adminpass1"}

	for i := 0; i < 2; i++ {
		if _, err := srv.Login(ip1, req); err != nil {
			t.Fatalf("attempt %d from ip1: expected success within burst, got: %v", i, err)
		}
	}

	// Third attempt from the same IP, even with correct credentials,
	// must be rejected by the IP gate.
	_, err := srv.Login(ip1, req)
	if err == nil {
		t.Fatalf("expected third attempt from ip1 to be rejected")
	}
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("expected codes.ResourceExhausted, got %s", status.Code(err))
	}

	// A different IP has its own, untouched bucket.
	if _, err := srv.Login(ip2, req); err != nil {
		t.Fatalf("expected ip2 to be unaffected by ip1's limit, got: %v", err)
	}
}

func newJSONRequest(t *testing.T, method, target, body, remoteAddr string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	return req
}

func TestLoginWeb_DrainedIPBucketReturns429(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{LoginBurst: 2, LoginPerMin: 2 * 60})
	seedAdmin(t, userManager, "webuser", "webpassword1")

	body := `{"username":"webuser","password":"webpassword1"}`
	remoteAddr := "1.2.3.4:5555"

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		srv.LoginWeb(rec, newJSONRequest(t, http.MethodPost, "/api/login", body, remoteAddr))
		if rec.Code != http.StatusOK {
			t.Fatalf("attempt %d: expected 200 within burst, got %d: %s", i, rec.Code, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	srv.LoginWeb(rec, newJSONRequest(t, http.MethodPost, "/api/login", body, remoteAddr))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 once the IP bucket is drained, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRefreshWeb_DrainedIPBucketReturns429NotTeapot(t *testing.T) {
	srv, _, _ := newTestAuthService(t, limits.Config{LoginBurst: 1, LoginPerMin: 60})

	remoteAddr := "1.2.3.5:5555"
	body := `{}`

	// First request consumes the sole token; with no token cookie it
	// falls through to a plain 401, but the IP gate itself let it pass.
	rec := httptest.NewRecorder()
	srv.RefreshWeb(rec, newJSONRequest(t, http.MethodPost, "/api/refresh", body, remoteAddr))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected first request to reach the token check (401), got %d: %s", rec.Code, rec.Body.String())
	}

	// Second request from the same IP should be rejected by the IP gate
	// with 429, not fall through to the writeGRPCError default (418).
	rec = httptest.NewRecorder()
	srv.RefreshWeb(rec, newJSONRequest(t, http.MethodPost, "/api/refresh", body, remoteAddr))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 once the IP bucket is drained, got %d: %s", rec.Code, rec.Body.String())
	}
}
