package main

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/discovery"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

// fakeTXTSetter records the TXT records it is given, and can be made to fail.
type fakeTXTSetter struct {
	mu    sync.Mutex
	calls [][]string
	fails int // number of upcoming calls to fail
}

func (f *fakeTXTSetter) SetTXT(txt []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, slices.Clone(txt))
	if f.fails > 0 {
		f.fails--
		return errors.New("set failed")
	}
	return nil
}

func (f *fakeTXTSetter) last() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return nil
	}
	return f.calls[len(f.calls)-1]
}

func (f *fakeTXTSetter) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeTXTSetter) all() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

func waitForTXT(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

type txtWatchFixture struct {
	users *core.UserManager
	set   *fakeTXTSetter
	stop  func()
}

// newTXTWatchFixture starts a watcher over a real UserManager for an appliance,
// building the record the way main does.
func newTXTWatchFixture(t *testing.T, retryDelay time.Duration) *txtWatchFixture {
	t.Helper()

	users, err := core.NewUserManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("failed to create user manager: %s", err)
	}
	t.Cleanup(users.Close)

	der := []byte("example certificate bytes")
	build := func() []string { return discovery.TXT(der, true, users.HasAnAdmin()) }

	f := &txtWatchFixture{users: users, set: &fakeTXTSetter{}}
	f.stop = watchDiscoveryTXT(users.GetUserUpdates(), f.set, build, retryDelay)
	t.Cleanup(f.stop)
	return f
}

func (f *txtWatchFixture) txt(hasAdmin bool) []string {
	return discovery.TXT([]byte("example certificate bytes"), true, hasAdmin)
}

func storeTestUser(t *testing.T, users *core.UserManager, username string, role auth.Role) {
	t.Helper()
	user, err := core.NewUser(username, "", "a-password", role)
	if err != nil {
		t.Fatalf("NewUser: %s", err)
	}
	if err := users.Store(user); err != nil {
		t.Fatalf("Store: %s", err)
	}
}

func TestWatchDiscoveryTXT_FollowsAdminState(t *testing.T) {
	f := newTXTWatchFixture(t, time.Hour)

	// The first admin (the agent's CreateAdmin) ends the setup state.
	storeTestUser(t, f.users, "alice", auth.AdminRole)
	waitForTXT(t, "setup flag to clear", func() bool { return slices.Equal(f.set.last(), f.txt(true)) })
	if slices.Contains(f.set.last(), "setup=1") {
		t.Fatalf("setup flag still advertised with an admin: %q", f.set.last())
	}

	// Removing the last admin brings it back.
	if err := f.users.Delete("alice"); err != nil {
		t.Fatalf("Delete: %s", err)
	}
	waitForTXT(t, "setup flag to return", func() bool { return slices.Equal(f.set.last(), f.txt(false)) })
	if !slices.Contains(f.set.last(), "setup=1") {
		t.Fatalf("setup flag not advertised without an admin: %q", f.set.last())
	}
}

func TestWatchDiscoveryTXT_NonAdminUserKeepsSetupState(t *testing.T) {
	f := newTXTWatchFixture(t, time.Hour)

	storeTestUser(t, f.users, "bob", auth.UserRole)
	waitForTXT(t, "a rebuild", func() bool { return f.set.count() >= 1 })

	if !slices.Equal(f.set.last(), f.txt(false)) {
		t.Fatalf("TXT after adding a non-admin = %q, want setup to remain", f.set.last())
	}
}

func TestWatchDiscoveryTXT_RetriesAfterFailure(t *testing.T) {
	f := newTXTWatchFixture(t, 5*time.Millisecond)
	f.set.mu.Lock()
	f.set.fails = 2
	f.set.mu.Unlock()

	storeTestUser(t, f.users, "alice", auth.AdminRole)

	// Two failures, then the retry that goes through, all with the same record.
	waitForTXT(t, "retries", func() bool { return f.set.count() >= 3 })
	for _, call := range f.set.all()[:3] {
		if !slices.Equal(call, f.txt(true)) {
			t.Fatalf("attempt carried %q, want %q", call, f.txt(true))
		}
	}

	// Once it succeeds the retrying stops.
	n := f.set.count()
	time.Sleep(40 * time.Millisecond)
	if got := f.set.count(); got != n {
		t.Fatalf("still setting the TXT after success: %d calls, was %d", got, n)
	}
}

func TestWatchDiscoveryTXT_StopEndsTheWatcher(t *testing.T) {
	f := newTXTWatchFixture(t, time.Hour)

	storeTestUser(t, f.users, "alice", auth.AdminRole)
	waitForTXT(t, "a rebuild", func() bool { return f.set.count() >= 1 })

	f.stop()
	f.stop() // may be called more than once
	n := f.set.count()

	storeTestUser(t, f.users, "bob", auth.AdminRole)
	time.Sleep(40 * time.Millisecond)
	if got := f.set.count(); got != n {
		t.Fatalf("the watcher kept running after stop: %d calls, was %d", got, n)
	}
}

func TestWatchDiscoveryTXT_StopCancelsPendingRetry(t *testing.T) {
	f := newTXTWatchFixture(t, 10*time.Millisecond)
	f.set.mu.Lock()
	f.set.fails = 1000
	f.set.mu.Unlock()

	storeTestUser(t, f.users, "alice", auth.AdminRole)
	waitForTXT(t, "a failed attempt", func() bool { return f.set.count() >= 1 })

	f.stop()
	n := f.set.count()
	time.Sleep(50 * time.Millisecond)
	if got := f.set.count(); got != n {
		t.Fatalf("retries continued after stop: %d calls, was %d", got, n)
	}
}
