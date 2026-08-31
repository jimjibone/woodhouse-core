package users

import (
	"strings"
	"testing"

	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

func TestValidNotificationText(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		body    string
		wantErr bool
	}{
		{name: "simple", title: "Test notification", body: "Delivery is working."},
		{name: "empty body", title: "Test notification"},
		{name: "title at the limit", title: strings.Repeat("a", maxNotificationTitleLength)},
		{name: "body at the limit", title: "Test", body: strings.Repeat("a", maxNotificationBodyLength)},
		// Notifications are stored and replayed to every client on connect, so
		// an unbounded title bloats every reconnect, not just one record.
		{name: "title too long", title: strings.Repeat("a", maxNotificationTitleLength+1), wantErr: true},
		{name: "body too long", title: "Test", body: strings.Repeat("a", maxNotificationBodyLength+1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validNotificationText(tt.title, tt.body)
			if tt.wantErr && err == nil {
				t.Errorf("validNotificationText: got nil, want an error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validNotificationText: got %v, want nil", err)
			}
		})
	}
}

func TestValidNotificationAudience(t *testing.T) {
	userManager, err := core.NewUserManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("NewUserManager: %s", err)
	}
	t.Cleanup(userManager.Close)

	alice, err := core.NewUser("alice", "Alice", "a-password", auth.UserRole)
	if err != nil {
		t.Fatalf("NewUser: %s", err)
	}
	if err := userManager.Store(alice); err != nil {
		t.Fatalf("Store: %s", err)
	}

	service := &UserService{userManager: userManager}

	tests := []struct {
		name     string
		audience core.Audience
		wantErr  bool
	}{
		{name: "everyone", audience: core.AudienceAll()},
		{name: "admins", audience: core.AudienceRole(auth.AdminRole)},
		{name: "known user", audience: core.AudienceUser("alice")},
		// A typo should surface as an error rather than as silence.
		{name: "unknown user", audience: core.AudienceUser("bob"), wantErr: true},
		{name: "empty username", audience: core.AudienceUser(""), wantErr: true},
		{name: "unspecified", audience: core.Audience{}, wantErr: true},
		{name: "unaddressable role", audience: core.AudienceRole(auth.NoAuthRole), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validNotificationAudience(tt.audience)
			if tt.wantErr && err == nil {
				t.Errorf("validNotificationAudience(%s): got nil, want an error", tt.audience)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validNotificationAudience(%s): got %v, want nil", tt.audience, err)
			}
		})
	}
}
