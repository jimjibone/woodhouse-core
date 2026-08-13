package main

import (
	"context"
	"testing"

	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/users"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	testUserMethod      = "/woodhouse.api.v1.clients.UserService/GetDevices"
	testUserAdminMethod = "/woodhouse.api.v1.clients.UserService/AddUser"
	testClientMethod    = "/woodhouse.api.v1.clients.ClientService/StatusStream"
)

// newTestInterceptor builds a fully wired AuthInterceptor backed by an
// in-memory store, along with the underlying managers needed to seed users
// and clients and to mint/revoke tokens directly in tests.
func newTestInterceptor(t *testing.T) (*AuthInterceptor, *core.UserManager, *core.ClientManager, *users.JWTManager, *clients.JWTManager) {
	t.Helper()

	store := stores.NewMemStore()

	userManager, err := core.NewUserManager(store)
	if err != nil {
		t.Fatalf("failed to create user manager: %s", err)
	}
	t.Cleanup(userManager.Close)

	clientManager, err := core.NewClientManager(store)
	if err != nil {
		t.Fatalf("failed to create client manager: %s", err)
	}
	t.Cleanup(clientManager.Close)

	userJwtManager, err := users.NewJWTManager(store)
	if err != nil {
		t.Fatalf("failed to create user jwt manager: %s", err)
	}
	t.Cleanup(userJwtManager.Close)

	clientJwtManager, err := clients.NewJWTManager(store)
	if err != nil {
		t.Fatalf("failed to create client jwt manager: %s", err)
	}
	t.Cleanup(clientJwtManager.Close)

	interceptor := NewAuthInterceptor(clientJwtManager, userJwtManager, clientManager, userManager)

	return interceptor, userManager, clientManager, userJwtManager, clientJwtManager
}

// seedUser adds a user directly via the UserManager, bypassing the login
// endpoint's bootstrap path.
func seedUser(t *testing.T, userManager *core.UserManager, username string, role auth.Role) {
	t.Helper()
	user, err := core.NewUser(username, "Test User", "password123", role)
	if err != nil {
		t.Fatalf("failed to build user: %s", err)
	}
	if err := userManager.Store(user); err != nil {
		t.Fatalf("failed to store user: %s", err)
	}
}

// ctxWithToken wraps ctx with incoming gRPC metadata carrying the given
// bearer token, mimicking what a real client request would present.
func ctxWithToken(token string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.MD{
		"authorization": []string{token},
	})
}

func TestAuthorize_ValidUserToken(t *testing.T) {
	interceptor, userManager, _, userJwtManager, _ := newTestInterceptor(t)

	seedUser(t, userManager, "alice", auth.UserRole)

	td, err := userJwtManager.GenerateTokens("alice", "Test User", auth.UserRole, false)
	if err != nil {
		t.Fatalf("failed to generate tokens: %s", err)
	}

	id, ctx, err := interceptor.authorize(ctxWithToken(td.AccessToken), testUserMethod)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
	if id != "alice" {
		t.Errorf("expected id %q, got %q", "alice", id)
	}

	claims, ok := ctx.Value("claims").(*users.AccessTokenClaims)
	if !ok {
		t.Fatalf("expected claims in context")
	}
	if claims.Username != "alice" {
		t.Errorf("expected claims username %q, got %q", "alice", claims.Username)
	}
}

func TestAuthorize_DeletedUser(t *testing.T) {
	interceptor, userManager, _, userJwtManager, _ := newTestInterceptor(t)

	seedUser(t, userManager, "alice", auth.UserRole)

	td, err := userJwtManager.GenerateTokens("alice", "Test User", auth.UserRole, false)
	if err != nil {
		t.Fatalf("failed to generate tokens: %s", err)
	}

	if err := userManager.Delete("alice"); err != nil {
		t.Fatalf("failed to delete user: %s", err)
	}

	_, _, err = interceptor.authorize(ctxWithToken(td.AccessToken), testUserMethod)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got: %v", err)
	}
}

func TestAuthorize_RevokedRefreshUUID(t *testing.T) {
	interceptor, userManager, _, userJwtManager, _ := newTestInterceptor(t)

	seedUser(t, userManager, "alice", auth.UserRole)

	td, err := userJwtManager.GenerateTokens("alice", "Test User", auth.UserRole, false)
	if err != nil {
		t.Fatalf("failed to generate tokens: %s", err)
	}

	userJwtManager.RevokeRefreshUUID(td.RefreshUUID)

	_, _, err = interceptor.authorize(ctxWithToken(td.AccessToken), testUserMethod)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got: %v", err)
	}
}

func TestAuthorize_ValidPairedClientToken(t *testing.T) {
	interceptor, _, clientManager, _, clientJwtManager := newTestInterceptor(t)

	const clientID = "client-1"
	if err := clientManager.UpdateClient(&core.Client{ID: clientID}); err != nil {
		t.Fatalf("failed to create client: %s", err)
	}
	if err := clientManager.SetClientPaired(clientID, true); err != nil {
		t.Fatalf("failed to pair client: %s", err)
	}

	td, err := clientJwtManager.GenerateTokens(clientID)
	if err != nil {
		t.Fatalf("failed to generate tokens: %s", err)
	}

	id, ctx, err := interceptor.authorize(ctxWithToken(td.AccessToken), testClientMethod)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
	if id != clientID {
		t.Errorf("expected id %q, got %q", clientID, id)
	}
	if ctx.Value("claims") == nil {
		t.Errorf("expected claims in context")
	}
}

func TestAuthorize_UnpairedClient(t *testing.T) {
	interceptor, _, clientManager, _, clientJwtManager := newTestInterceptor(t)

	const clientID = "client-1"
	if err := clientManager.UpdateClient(&core.Client{ID: clientID}); err != nil {
		t.Fatalf("failed to create client: %s", err)
	}
	if err := clientManager.SetClientPaired(clientID, true); err != nil {
		t.Fatalf("failed to pair client: %s", err)
	}

	td, err := clientJwtManager.GenerateTokens(clientID)
	if err != nil {
		t.Fatalf("failed to generate tokens: %s", err)
	}

	if err := clientManager.SetClientPaired(clientID, false); err != nil {
		t.Fatalf("failed to unpair client: %s", err)
	}

	_, _, err = interceptor.authorize(ctxWithToken(td.AccessToken), testClientMethod)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got: %v", err)
	}
}

func TestAuthorize_RoleEnforcement(t *testing.T) {
	interceptor, userManager, _, userJwtManager, _ := newTestInterceptor(t)

	seedUser(t, userManager, "bob", auth.UserRole)

	td, err := userJwtManager.GenerateTokens("bob", "Test User", auth.UserRole, false)
	if err != nil {
		t.Fatalf("failed to generate tokens: %s", err)
	}

	_, _, err = interceptor.authorize(ctxWithToken(td.AccessToken), testUserAdminMethod)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got: %v", err)
	}
}
