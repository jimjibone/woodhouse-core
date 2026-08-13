package core

import (
	"errors"
	"testing"

	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

func newTestUserManager(t *testing.T) *UserManager {
	t.Helper()
	store := stores.NewMemStore()
	manager, err := NewUserManager(store)
	if err != nil {
		t.Fatalf("failed to create user manager: %s", err)
	}
	t.Cleanup(manager.Close)
	return manager
}

// addTestUser stores a user with a known password. resetPassword mirrors
// what AddUser does to a freshly created account.
func addTestUser(t *testing.T, manager *UserManager, username, password string, resetPassword bool) *User {
	t.Helper()
	user, err := NewUser(username, "Test User", password, auth.UserRole)
	if err != nil {
		t.Fatalf("NewUser: unexpected error: %s", err)
	}
	user.ResetPassword = resetPassword
	if err := manager.Store(user); err != nil {
		t.Fatalf("Store: unexpected error: %s", err)
	}
	return user
}

func TestChangePassword_CorrectCurrentPassword(t *testing.T) {
	manager := newTestUserManager(t)
	addTestUser(t, manager, "alice", "old-password", false)

	if err := manager.ChangePassword("alice", "old-password", "new-password"); err != nil {
		t.Fatalf("ChangePassword: unexpected error: %s", err)
	}

	user := manager.Find("alice")
	if user == nil {
		t.Fatal("Find: user went missing after password change")
	}
	if !user.IsCorrectPassword("new-password") {
		t.Error("new password was not accepted after ChangePassword")
	}
	if user.IsCorrectPassword("old-password") {
		t.Error("old password still accepted after ChangePassword")
	}
}

func TestChangePassword_WrongCurrentPasswordLeavesPasswordAlone(t *testing.T) {
	manager := newTestUserManager(t)
	addTestUser(t, manager, "alice", "old-password", false)

	err := manager.ChangePassword("alice", "not-the-password", "new-password")
	if !errors.Is(err, ErrIncorrectPassword) {
		t.Fatalf("ChangePassword: expected ErrIncorrectPassword, got %v", err)
	}

	// The whole point of the check: a failed attempt must not have moved
	// the password, or a stolen session could still lock the owner out.
	user := manager.Find("alice")
	if !user.IsCorrectPassword("old-password") {
		t.Error("original password no longer works after a rejected change")
	}
	if user.IsCorrectPassword("new-password") {
		t.Error("rejected password was applied anyway")
	}
}

func TestChangePassword_ClearsResetPasswordFlag(t *testing.T) {
	manager := newTestUserManager(t)
	addTestUser(t, manager, "alice", "temp-password", true)

	if err := manager.ChangePassword("alice", "temp-password", "chosen-password"); err != nil {
		t.Fatalf("ChangePassword: unexpected error: %s", err)
	}

	if manager.Find("alice").ResetPassword {
		t.Error("ResetPassword still set after the user chose their own password")
	}
}

func TestChangePassword_RejectsShortPassword(t *testing.T) {
	manager := newTestUserManager(t)
	addTestUser(t, manager, "alice", "old-password", false)

	if err := manager.ChangePassword("alice", "old-password", "short"); err == nil {
		t.Fatal("ChangePassword: expected an error for a too-short password, got nil")
	}

	if !manager.Find("alice").IsCorrectPassword("old-password") {
		t.Error("original password stopped working after a rejected short password")
	}
}

func TestChangePassword_UnknownUser(t *testing.T) {
	manager := newTestUserManager(t)

	err := manager.ChangePassword("nobody", "old-password", "new-password")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ChangePassword: expected ErrUserNotFound, got %v", err)
	}
}

func TestResetPassword_SetsPasswordAndForcesReset(t *testing.T) {
	manager := newTestUserManager(t)
	addTestUser(t, manager, "alice", "old-password", false)

	if err := manager.ResetPassword("alice", "admin-chosen-password"); err != nil {
		t.Fatalf("ResetPassword: unexpected error: %s", err)
	}

	user := manager.Find("alice")
	if !user.IsCorrectPassword("admin-chosen-password") {
		t.Error("admin-set password was not accepted after ResetPassword")
	}
	// User.SetPassword clears the flag, so this guards the re-raise that
	// makes the temporary password temporary.
	if !user.ResetPassword {
		t.Error("ResetPassword flag not set after an admin reset")
	}
}

func TestResetPassword_UnknownUser(t *testing.T) {
	manager := newTestUserManager(t)

	err := manager.ResetPassword("nobody", "admin-chosen-password")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ResetPassword: expected ErrUserNotFound, got %v", err)
	}
}
