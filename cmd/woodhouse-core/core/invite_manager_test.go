package core

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

// inviteClock is a clock the tests move by hand. It is safe for concurrent use
// because some tests redeem from several goroutines.
type inviteClock struct {
	mu  sync.Mutex
	now time.Time
}

func newInviteClock() *inviteClock {
	return &inviteClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *inviteClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *inviteClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestInviteManager(t *testing.T, store stores.Store, users *UserManager, clock *inviteClock) *InviteManager {
	t.Helper()
	manager, err := NewInviteManagerWithClock(store, users, clock.Now)
	if err != nil {
		t.Fatalf("failed to create invite manager: %s", err)
	}
	t.Cleanup(manager.Close)
	return manager
}

// newInviteFixture builds an invite manager over a fresh store and user
// manager, with a clock the test controls.
func newInviteFixture(t *testing.T) (*InviteManager, *UserManager, *inviteClock) {
	t.Helper()
	users := newTestUserManager(t)
	clock := newInviteClock()
	return newTestInviteManager(t, stores.NewMemStore(), users, clock), users, clock
}

func createInvite(t *testing.T, manager *InviteManager, username string) (*Invite, string) {
	t.Helper()
	invite, token, err := manager.Create(username, "Test Person", auth.UserRole, "carol")
	if err != nil {
		t.Fatalf("Create(%q): unexpected error: %s", username, err)
	}
	return invite, token
}

func acceptNothing(*Invite) error { return nil }

func TestInviteCreate_ReturnsInviteAndToken(t *testing.T) {
	manager, _, clock := newInviteFixture(t)

	invite, token := createInvite(t, manager, "alice")

	if invite.ID == "" {
		t.Error("invite has no id")
	}
	if invite.Username != "alice" || invite.Fullname != "Test Person" || invite.Role != auth.UserRole || invite.CreatedBy != "carol" {
		t.Errorf("invite does not carry the request: %+v", invite)
	}
	if !invite.CreatedAt.Equal(clock.Now()) {
		t.Errorf("CreatedAt = %s, want %s", invite.CreatedAt, clock.Now())
	}
	if want := invite.CreatedAt.Add(24 * time.Hour); !invite.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %s, want %s (24 hours after creation)", invite.ExpiresAt, want)
	}

	// 32 random bytes, base64url without padding.
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("token %q is not unpadded base64url: %s", token, err)
	}
	if len(raw) != 32 {
		t.Errorf("token carries %d bytes, want 32", len(raw))
	}

	if sum := sha256.Sum256([]byte(token)); string(invite.TokenHash) != string(sum[:]) {
		t.Error("TokenHash is not the SHA-256 of the token")
	}
}

func TestInviteCreate_TokensAndIDsAreUnique(t *testing.T) {
	manager, _, _ := newInviteFixture(t)

	a, tokenA := createInvite(t, manager, "alice")
	b, tokenB := createInvite(t, manager, "bob")

	if tokenA == tokenB {
		t.Error("two invites got the same token")
	}
	if a.ID == b.ID {
		t.Error("two invites got the same id")
	}
}

func TestInviteCreate_RejectsInvalidUsername(t *testing.T) {
	manager, _, _ := newInviteFixture(t)

	_, _, err := manager.Create("", "", auth.UserRole, "carol")
	if !errors.Is(err, ErrUsernameTooShort) {
		t.Fatalf("Create(\"\"): expected ErrUsernameTooShort, got %v", err)
	}

	// The same rule NewUser applies, so an invite can never be created for a
	// name the account could not be.
	_, newUserErr := NewUser("", "", "a-password", auth.UserRole)
	if !errors.Is(newUserErr, ErrUsernameTooShort) {
		t.Fatalf("NewUser(\"\"): expected ErrUsernameTooShort, got %v", newUserErr)
	}

	if got := manager.List(); len(got) != 0 {
		t.Errorf("a rejected invite was stored: %+v", got)
	}
}

func TestInviteCreate_ExistingUser(t *testing.T) {
	manager, users, _ := newInviteFixture(t)
	addTestUser(t, users, "alice", "a-password", false)

	_, _, err := manager.Create("alice", "", auth.UserRole, "carol")
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("Create for an existing user: expected ErrAlreadyExists, got %v", err)
	}
	if got := manager.List(); len(got) != 0 {
		t.Errorf("a rejected invite was stored: %+v", got)
	}
}

func TestInviteCreate_PendingInvite(t *testing.T) {
	manager, _, _ := newInviteFixture(t)
	createInvite(t, manager, "alice")

	_, _, err := manager.Create("alice", "", auth.AdminRole, "carol")
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("Create for a pending invite's username: expected ErrAlreadyExists, got %v", err)
	}
	if got := manager.List(); len(got) != 1 {
		t.Errorf("got %d pending invites, want 1", len(got))
	}
}

func TestInviteCreate_UsernameFreedByRevokeAndExpiry(t *testing.T) {
	manager, _, clock := newInviteFixture(t)

	first, _ := createInvite(t, manager, "alice")
	if err := manager.Revoke(first.ID); err != nil {
		t.Fatalf("Revoke: %s", err)
	}
	second, _ := createInvite(t, manager, "alice")

	clock.Advance(InviteLifetime)
	third, _, err := manager.Create("alice", "", auth.UserRole, "carol")
	if err != nil {
		t.Fatalf("Create after the earlier invite expired: %s", err)
	}
	if got := manager.List(); len(got) != 1 || got[0].ID != third.ID || got[0].ID == second.ID {
		t.Errorf("expected only the newest invite to be pending, got %+v", got)
	}
}

func TestInviteList_OldestFirstAndCopies(t *testing.T) {
	manager, _, clock := newInviteFixture(t)

	createInvite(t, manager, "alice")
	clock.Advance(time.Minute)
	createInvite(t, manager, "bob")
	clock.Advance(time.Minute)
	createInvite(t, manager, "dave")

	got := manager.List()
	if len(got) != 3 || got[0].Username != "alice" || got[1].Username != "bob" || got[2].Username != "dave" {
		t.Fatalf("List: got %+v, want alice, bob, dave", got)
	}

	// The caller gets copies, not the manager's own records.
	got[0].Username = "mallory"
	got[0].TokenHash[0] ^= 0xff
	if again := manager.List(); again[0].Username != "alice" {
		t.Errorf("mutating a listed invite changed the stored one: %+v", again[0])
	}
}

func TestInviteExpiry(t *testing.T) {
	manager, _, clock := newInviteFixture(t)
	_, token := createInvite(t, manager, "alice")

	// Still good right up to the deadline.
	clock.Advance(InviteLifetime - time.Millisecond)
	if got := manager.List(); len(got) != 1 {
		t.Fatalf("invite missing before its expiry: %+v", got)
	}

	// Gone at the deadline: not listed, and the token no longer redeems.
	clock.Advance(time.Millisecond)
	if got := manager.List(); len(got) != 0 {
		t.Fatalf("expired invite still listed: %+v", got)
	}
	err := manager.Redeem(token, acceptNothing)
	if !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf("Redeem of an expired invite: expected ErrInviteNotFound, got %v", err)
	}
}

func TestInviteExpiry_RedeemPrunesWithoutList(t *testing.T) {
	manager, _, clock := newInviteFixture(t)
	_, token := createInvite(t, manager, "alice")

	clock.Advance(InviteLifetime + time.Hour)
	err := manager.Redeem(token, func(*Invite) error {
		t.Error("accept ran for an expired invite")
		return nil
	})
	if !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf("Redeem of an expired invite: expected ErrInviteNotFound, got %v", err)
	}
}

func TestInviteRevoke(t *testing.T) {
	manager, _, _ := newInviteFixture(t)
	invite, token := createInvite(t, manager, "alice")
	other, otherToken := createInvite(t, manager, "bob")

	if err := manager.Revoke(invite.ID); err != nil {
		t.Fatalf("Revoke: %s", err)
	}

	got := manager.List()
	if len(got) != 1 || got[0].ID != other.ID {
		t.Fatalf("after revoking alice's invite, List = %+v", got)
	}
	if err := manager.Redeem(token, acceptNothing); !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf("Redeem of a revoked invite: expected ErrInviteNotFound, got %v", err)
	}
	if err := manager.Redeem(otherToken, acceptNothing); err != nil {
		t.Fatalf("Redeem of the other invite: %s", err)
	}
}

func TestInviteRevoke_UnknownIDSucceeds(t *testing.T) {
	manager, _, _ := newInviteFixture(t)
	createInvite(t, manager, "alice")

	if err := manager.Revoke("no-such-id"); err != nil {
		t.Fatalf("Revoke of an unknown id: %s", err)
	}
	if err := manager.Revoke(""); err != nil {
		t.Fatalf("Revoke of an empty id: %s", err)
	}
	if got := manager.List(); len(got) != 1 {
		t.Errorf("revoking an unknown id changed the pending invites: %+v", got)
	}
}

func TestInviteRedeem_SingleUse(t *testing.T) {
	manager, _, _ := newInviteFixture(t)
	invite, token := createInvite(t, manager, "alice")

	var accepted *Invite
	err := manager.Redeem(token, func(got *Invite) error {
		accepted = got
		return nil
	})
	if err != nil {
		t.Fatalf("Redeem: %s", err)
	}
	if accepted == nil || accepted.ID != invite.ID || accepted.Username != "alice" || accepted.Role != auth.UserRole {
		t.Errorf("accept got %+v, want the invite for alice", accepted)
	}

	if got := manager.List(); len(got) != 0 {
		t.Errorf("a redeemed invite is still pending: %+v", got)
	}
	err = manager.Redeem(token, func(*Invite) error {
		t.Error("accept ran for an invite that was already used")
		return nil
	})
	if !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf("second Redeem: expected ErrInviteNotFound, got %v", err)
	}
}

func TestInviteRedeem_UnknownToken(t *testing.T) {
	manager, _, _ := newInviteFixture(t)
	_, token := createInvite(t, manager, "alice")

	for _, bad := range []string{"", "nonsense", token + "x", token[1:]} {
		err := manager.Redeem(bad, func(*Invite) error {
			t.Errorf("accept ran for token %q", bad)
			return nil
		})
		if !errors.Is(err, ErrInviteNotFound) {
			t.Errorf("Redeem(%q): expected ErrInviteNotFound, got %v", bad, err)
		}
	}
	if got := manager.List(); len(got) != 1 {
		t.Errorf("a bad token disturbed the pending invites: %+v", got)
	}
}

func TestInviteRedeem_ConcurrentRedemptionsSucceedOnce(t *testing.T) {
	manager, _, _ := newInviteFixture(t)
	_, token := createInvite(t, manager, "alice")

	const attempts = 16
	var accepted, succeeded atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := manager.Redeem(token, func(*Invite) error {
				accepted.Add(1)
				return nil
			})
			if err == nil {
				succeeded.Add(1)
			} else if !errors.Is(err, ErrInviteNotFound) {
				t.Errorf("Redeem: unexpected error %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if accepted.Load() != 1 || succeeded.Load() != 1 {
		t.Fatalf("accept ran %d times and %d redemptions succeeded, want exactly 1 of each", accepted.Load(), succeeded.Load())
	}
}

func TestInviteRedeem_FailedAcceptPutsInviteBack(t *testing.T) {
	manager, _, _ := newInviteFixture(t)
	invite, token := createInvite(t, manager, "alice")

	acceptErr := errors.New("username already in use")
	err := manager.Redeem(token, func(*Invite) error { return acceptErr })
	if !errors.Is(err, acceptErr) {
		t.Fatalf("Redeem: expected accept's error, got %v", err)
	}

	got := manager.List()
	if len(got) != 1 || got[0].ID != invite.ID {
		t.Fatalf("invite was not put back: %+v", got)
	}
	if !got[0].ExpiresAt.Equal(invite.ExpiresAt) {
		t.Errorf("put-back changed ExpiresAt from %s to %s", invite.ExpiresAt, got[0].ExpiresAt)
	}

	// And it still works.
	if err := manager.Redeem(token, acceptNothing); err != nil {
		t.Fatalf("retry after a failed accept: %s", err)
	}
}

func TestInviteRedeem_FailedAcceptDoesNotResurrectRevokedInvite(t *testing.T) {
	manager, _, _ := newInviteFixture(t)
	_, token := createInvite(t, manager, "alice")

	err := manager.Redeem(token, func(in *Invite) error {
		// The admin revokes while the account is being created.
		if err := manager.Revoke(in.ID); err != nil {
			t.Errorf("Revoke during redemption: %s", err)
		}
		return errors.New("failed")
	})
	if err == nil {
		t.Fatal("Redeem: expected the accept error")
	}

	if got := manager.List(); len(got) != 0 {
		t.Fatalf("a revoked invite came back: %+v", got)
	}
	if err := manager.Redeem(token, acceptNothing); !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf("Redeem of the revoked invite: expected ErrInviteNotFound, got %v", err)
	}
}

func TestInviteRedeem_FailedAcceptDoesNotResurrectExpiredInvite(t *testing.T) {
	manager, _, clock := newInviteFixture(t)
	_, token := createInvite(t, manager, "alice")

	_ = manager.Redeem(token, func(*Invite) error {
		clock.Advance(InviteLifetime)
		return errors.New("failed")
	})

	if got := manager.List(); len(got) != 0 {
		t.Fatalf("an invite that expired mid-redemption came back: %+v", got)
	}
}

func TestInviteRedeem_UsernameStaysReservedWhileRedeeming(t *testing.T) {
	manager, _, _ := newInviteFixture(t)
	_, token := createInvite(t, manager, "alice")

	_ = manager.Redeem(token, func(*Invite) error {
		_, _, err := manager.Create("alice", "", auth.UserRole, "carol")
		if !errors.Is(err, ErrAlreadyExists) {
			t.Errorf("Create during redemption of the same username: expected ErrAlreadyExists, got %v", err)
		}
		return errors.New("failed")
	})

	// Back as one pending invite, not two.
	if got := manager.List(); len(got) != 1 {
		t.Fatalf("got %d pending invites, want 1: %+v", len(got), got)
	}
}

func TestInvitePersistence_RoundTrip(t *testing.T) {
	store := stores.NewMemStore()
	users := newTestUserManager(t)
	clock := newInviteClock()

	manager := newTestInviteManager(t, store, users, clock)
	alice, aliceToken := createInvite(t, manager, "alice")
	clock.Advance(time.Minute)
	bob, bobToken, err := manager.Create("bob", "Bob", auth.AdminRole, "carol")
	if err != nil {
		t.Fatalf("Create: %s", err)
	}

	// The store holds the hash, never the token.
	data, err := store.Get("invites")
	if err != nil {
		t.Fatalf("invites were not written to the store: %s", err)
	}
	for _, token := range []string{aliceToken, bobToken} {
		if strings.Contains(string(data), token) {
			t.Fatal("the stored invites contain a plaintext token")
		}
	}
	manager.Close()

	reloaded := newTestInviteManager(t, store, users, clock)
	got := reloaded.List()
	if len(got) != 2 {
		t.Fatalf("reloaded %d invites, want 2: %+v", len(got), got)
	}
	if got[0].ID != alice.ID || got[0].Username != "alice" || got[0].Role != auth.UserRole || !got[0].CreatedAt.Equal(alice.CreatedAt) || !got[0].ExpiresAt.Equal(alice.ExpiresAt) {
		t.Errorf("reloaded first invite %+v, want %+v", got[0], alice)
	}
	if got[1].ID != bob.ID || got[1].Username != "bob" || got[1].Fullname != "Bob" || got[1].Role != auth.AdminRole || got[1].CreatedBy != "carol" {
		t.Errorf("reloaded second invite %+v, want %+v", got[1], bob)
	}

	// The tokens still redeem after the restart.
	if err := reloaded.Redeem(bobToken, acceptNothing); err != nil {
		t.Fatalf("Redeem of bob's token after restart: %s", err)
	}
	if err := reloaded.Redeem(aliceToken, acceptNothing); err != nil {
		t.Fatalf("Redeem of alice's token after restart: %s", err)
	}
}

func TestInvitePersistence_ChangesAreSavedImmediately(t *testing.T) {
	store := stores.NewMemStore()
	users := newTestUserManager(t)
	clock := newInviteClock()

	manager := newTestInviteManager(t, store, users, clock)
	alice, _ := createInvite(t, manager, "alice")
	_, bobToken := createInvite(t, manager, "bob")
	_, daveToken := createInvite(t, manager, "dave")

	if err := manager.Revoke(alice.ID); err != nil {
		t.Fatalf("Revoke: %s", err)
	}
	if err := manager.Redeem(bobToken, acceptNothing); err != nil {
		t.Fatalf("Redeem: %s", err)
	}

	// No Close: a crash now must not bring the revoked or redeemed invite
	// back. A second manager reading the same store is that restart.
	crashed := newTestInviteManager(t, store, users, clock)
	got := crashed.List()
	if len(got) != 1 || got[0].Username != "dave" {
		t.Fatalf("after a crash the pending invites are %+v, want only dave", got)
	}
	if err := crashed.Redeem(daveToken, acceptNothing); err != nil {
		t.Fatalf("Redeem of dave's token: %s", err)
	}
}

func TestInvitePersistence_FailedAcceptPutBackIsSaved(t *testing.T) {
	store := stores.NewMemStore()
	users := newTestUserManager(t)
	clock := newInviteClock()

	manager := newTestInviteManager(t, store, users, clock)
	_, token := createInvite(t, manager, "alice")
	_ = manager.Redeem(token, func(*Invite) error { return errors.New("failed") })

	restarted := newTestInviteManager(t, store, users, clock)
	if err := restarted.Redeem(token, acceptNothing); err != nil {
		t.Fatalf("Redeem after a failed attempt and a restart: %s", err)
	}
}

func TestInvitePersistence_ExpiredInvitesPrunedOnLoad(t *testing.T) {
	store := stores.NewMemStore()
	users := newTestUserManager(t)
	clock := newInviteClock()

	manager := newTestInviteManager(t, store, users, clock)
	createInvite(t, manager, "alice")
	clock.Advance(InviteLifetime - time.Hour)
	createInvite(t, manager, "bob")
	manager.Close()

	// The server was down while alice's invite expired.
	clock.Advance(2 * time.Hour)
	reloaded := newTestInviteManager(t, store, users, clock)

	// And the prune was written back, not just applied in memory.
	data, err := store.Get("invites")
	if err != nil {
		t.Fatalf("Get invites: %s", err)
	}
	if strings.Contains(string(data), `"alice"`) {
		t.Error("the expired invite is still in the store")
	}

	got := reloaded.List()
	if len(got) != 1 || got[0].Username != "bob" {
		t.Fatalf("after reload the pending invites are %+v, want only bob", got)
	}
}

func TestInviteManager_EmptyStore(t *testing.T) {
	manager, _, _ := newInviteFixture(t)
	if got := manager.List(); len(got) != 0 {
		t.Fatalf("a fresh manager lists %+v", got)
	}
}
