package local

import (
	"context"
	"net"
	"testing"

	localapi "github.com/jimjibone/woodhouse-api/go/v1/local"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/limits"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeSettings struct{ name string }

func (s *fakeSettings) InstanceName() string { return s.name }
func (s *fakeSettings) SetInstanceName(name string) (string, error) {
	s.name = name
	return name, nil
}

func newTestService(t *testing.T) (*BootstrapService, *core.UserManager, *fakeSettings) {
	t.Helper()
	users, err := core.NewUserManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("user manager: %s", err)
	}
	t.Cleanup(users.Close)
	settings := &fakeSettings{name: "banana"}
	srv := NewBootstrapService(users, settings, limits.NewLogin(limits.Config{}),
		&net.TCPAddr{Port: 4000}, &net.TCPAddr{Port: 4080}, []byte{1, 2, 3})
	return srv, users, settings
}

func TestCreateAdminOnce(t *testing.T) {
	srv, users, settings := newTestService(t)
	ctx := context.Background()

	st, err := srv.GetStatus(ctx, &localapi.GetStatusRequest{})
	if err != nil || st.HasAdmin || st.ApiPort != 4000 || st.WebPort != 4080 {
		t.Fatalf("unexpected status %+v, err %v", st, err)
	}

	_, err = srv.CreateAdmin(ctx, &localapi.CreateAdminRequest{Username: "alice", Password: "alicepass1", InstanceName: "kitchen"})
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	u := users.Find("alice")
	if u == nil || u.Role != auth.AdminRole || u.ResetPassword {
		t.Fatalf("unexpected user %+v", u)
	}
	if settings.name != "kitchen" {
		t.Fatalf("instance name not set: %q", settings.name)
	}

	_, err = srv.CreateAdmin(ctx, &localapi.CreateAdminRequest{Username: "bob", Password: "bobpass12"})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("second admin: expected FailedPrecondition, got %v", err)
	}
}

func TestVerifyAdmin(t *testing.T) {
	srv, users, _ := newTestService(t)
	ctx := context.Background()

	if _, err := srv.CreateAdmin(ctx, &localapi.CreateAdminRequest{Username: "alice", Password: "alicepass1"}); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	viewer, err := core.NewUser("bob", "", "bobpass12", auth.UserRole)
	if err != nil {
		t.Fatalf("new user: %v", err)
	}
	if err := users.Store(viewer); err != nil {
		t.Fatalf("store user: %v", err)
	}

	if _, err := srv.VerifyAdmin(ctx, &localapi.VerifyAdminRequest{Username: "alice", Password: "alicepass1"}); err != nil {
		t.Fatalf("correct admin: %v", err)
	}
	for _, tc := range []struct{ user, pass string }{
		{"alice", "wrong-pass"},
		{"bob", "bobpass12"}, // not an admin
		{"nobody", "whatever1"},
	} {
		_, err := srv.VerifyAdmin(ctx, &localapi.VerifyAdminRequest{Username: tc.user, Password: tc.pass})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("%s: expected Unauthenticated, got %v", tc.user, err)
		}
	}
}
