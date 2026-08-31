package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jimjibone/queue/v2"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

func newTestNotificationManager(t *testing.T, store stores.Store, userManager *UserManager) *NotificationManager {
	t.Helper()

	manager, err := NewNotificationManager(store, userManager)
	if err != nil {
		t.Fatalf("NewNotificationManager: %s", err)
	}
	t.Cleanup(manager.Close)
	return manager
}

// expectNoUpdate asserts a listener is quiet, which is how the tests check that
// a per-user change did not leak to everybody.
func expectNoUpdate(t *testing.T, lis *queue.Sub[NotificationUpdate]) {
	t.Helper()

	select {
	case update := <-lis.Sub():
		t.Fatalf("unexpected update: %+v", update)
	case <-time.After(100 * time.Millisecond):
	}
}

// collectNotificationUpdates reads exactly count updates off a listener.
func collectNotificationUpdates(t *testing.T, lis *queue.Sub[NotificationUpdate], count int) []NotificationUpdate {
	t.Helper()

	updates := make([]NotificationUpdate, 0, count)
	for len(updates) < count {
		select {
		case update := <-lis.Sub():
			updates = append(updates, update)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out after %d of %d updates", len(updates), count)
		}
	}
	return updates
}

func testNotification(id, title string, audience Audience) *Notification {
	return NewNotification(id, title, "", InfoLevel, audience)
}

func TestNotificationAudienceMatches(t *testing.T) {
	tests := []struct {
		name     string
		audience Audience
		username string
		role     auth.Role
		want     bool
	}{
		{"all matches an admin", AudienceAll(), "alice", auth.AdminRole, true},
		{"all matches a user", AudienceAll(), "bob", auth.UserRole, true},
		{"role matches the same role", AudienceRole(auth.AdminRole), "alice", auth.AdminRole, true},
		{"role rejects a different role", AudienceRole(auth.AdminRole), "bob", auth.UserRole, false},
		{"user matches the named user", AudienceUser("alice"), "alice", auth.UserRole, true},
		{"user rejects anybody else", AudienceUser("alice"), "bob", auth.UserRole, false},
		// The zero value has to fail closed: a record this build cannot decode
		// must reach nobody rather than everybody.
		{"unspecified matches nobody", Audience{}, "alice", auth.AdminRole, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.audience.Matches(test.username, test.role); got != test.want {
				t.Errorf("Matches(%q, %s): got %v, want %v", test.username, test.role, got, test.want)
			}
		})
	}
}

func TestNotificationAudienceValid(t *testing.T) {
	tests := []struct {
		name     string
		audience Audience
		want     bool
	}{
		{"all", AudienceAll(), true},
		{"admin role", AudienceRole(auth.AdminRole), true},
		{"user role", AudienceRole(auth.UserRole), true},
		{"noauth role is not addressable", AudienceRole(auth.NoAuthRole), false},
		{"named user", AudienceUser("alice"), true},
		{"user with no name", AudienceUser(""), false},
		{"unspecified", Audience{}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.audience.Valid(); got != test.want {
				t.Errorf("Valid(): got %v, want %v", got, test.want)
			}
		})
	}
}

func TestNotificationManagerAddPublishesAndPersists(t *testing.T) {
	store := stores.NewMemStore()
	manager := newTestNotificationManager(t, store, newTestUserManager(t))

	lis := manager.GetListener()
	defer lis.Close()

	notification := testNotification("notif-1", "Test notification", AudienceAll())
	if err := manager.Add(notification); err != nil {
		t.Fatalf("Add: %s", err)
	}

	if err := manager.Add(testNotification("notif-1", "Again", AudienceAll())); err != ErrNotificationAlreadyExists {
		t.Fatalf("Add duplicate: got %v, want ErrNotificationAlreadyExists", err)
	}

	updates := collectNotificationUpdates(t, lis, 1)
	if updates[0].Updated == nil || updates[0].Updated.ID != "notif-1" {
		t.Fatalf("published update: got %+v", updates[0])
	}
	if updates[0].ForUser != "" {
		t.Errorf("ForUser: got %q, want empty (audience-wide)", updates[0].ForUser)
	}

	if err := manager.save(); err != nil {
		t.Fatalf("save: %s", err)
	}
	if !store.Has("notifications.json") {
		t.Fatal("notifications.json was not written")
	}

	data, err := store.Get("notifications.json")
	if err != nil {
		t.Fatalf("Get: %s", err)
	}
	stored := struct {
		Notifications []*Notification `json:"notifications"`
	}{}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("unmarshal: %s", err)
	}
	if len(stored.Notifications) != 1 || stored.Notifications[0].Title != "Test notification" {
		t.Fatalf("stored: got %+v", stored.Notifications)
	}
}

func TestNotificationManagerAddRejectsInvalid(t *testing.T) {
	manager := newTestNotificationManager(t, stores.NewMemStore(), newTestUserManager(t))

	tests := []struct {
		name         string
		notification *Notification
	}{
		{"empty id", testNotification("", "Title", AudienceAll())},
		{"empty title", testNotification("notif-1", "", AudienceAll())},
		{"unspecified audience", testNotification("notif-1", "Title", Audience{})},
		{"user audience with no name", testNotification("notif-1", "Title", AudienceUser(""))},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := manager.Add(test.notification)
			if !errors.Is(err, ErrNotificationInvalid) {
				t.Fatalf("Add: got %v, want ErrNotificationInvalid", err)
			}
		})
	}
}

func TestNotificationManagerReadStateIsPerUser(t *testing.T) {
	manager := newTestNotificationManager(t, stores.NewMemStore(), newTestUserManager(t))

	if err := manager.Add(testNotification("notif-1", "Test", AudienceAll())); err != nil {
		t.Fatalf("Add: %s", err)
	}

	lis := manager.GetListener()
	defer lis.Close()

	if err := manager.MarkRead("notif-1", "alice"); err != nil {
		t.Fatalf("MarkRead: %s", err)
	}

	update := collectNotificationUpdates(t, lis, 1)[0]
	if update.ForUser != "alice" {
		t.Errorf("ForUser: got %q, want %q", update.ForUser, "alice")
	}
	if !update.Updated.IsReadBy("alice") {
		t.Error("alice should have read the notification")
	}
	if update.Updated.IsReadBy("bob") {
		t.Error("bob must not inherit alice's read state")
	}

	// The projection is what the wire actually carries.
	if !update.Updated.Pb("alice").GetRead() {
		t.Error("Pb(alice).Read: got false, want true")
	}
	if update.Updated.Pb("bob").GetRead() {
		t.Error("Pb(bob).Read: got true, want false")
	}

	// Marking again is a no-op and must publish nothing, so a repeated click
	// costs no stream traffic.
	if err := manager.MarkRead("notif-1", "alice"); err != nil {
		t.Fatalf("MarkRead again: %s", err)
	}
	expectNoUpdate(t, lis)

	if err := manager.MarkRead("missing", "alice"); err != ErrNotificationNotFound {
		t.Fatalf("MarkRead missing: got %v, want ErrNotificationNotFound", err)
	}
}

func TestNotificationManagerMarkAllReadRespectsAudience(t *testing.T) {
	manager := newTestNotificationManager(t, stores.NewMemStore(), newTestUserManager(t))

	if err := manager.Add(testNotification("notif-all", "Everyone", AudienceAll())); err != nil {
		t.Fatalf("Add: %s", err)
	}
	if err := manager.Add(testNotification("notif-admin", "Admins", AudienceRole(auth.AdminRole))); err != nil {
		t.Fatalf("Add: %s", err)
	}
	if err := manager.Add(testNotification("notif-bob", "Bob only", AudienceUser("bob"))); err != nil {
		t.Fatalf("Add: %s", err)
	}

	// alice is a plain user, so only the audience-all notification is hers.
	marked, err := manager.MarkAllRead("alice", auth.UserRole)
	if err != nil {
		t.Fatalf("MarkAllRead: %s", err)
	}
	if marked != 1 {
		t.Fatalf("marked: got %d, want 1", marked)
	}

	if got := manager.Unread("alice", auth.UserRole); got != 0 {
		t.Errorf("alice unread: got %d, want 0", got)
	}
	// An admin still has both of theirs unread.
	if got := manager.Unread("carol", auth.AdminRole); got != 2 {
		t.Errorf("carol unread: got %d, want 2", got)
	}
}

func TestNotificationManagerDismissIsPerUser(t *testing.T) {
	manager := newTestNotificationManager(t, stores.NewMemStore(), newTestUserManager(t))

	if err := manager.Add(testNotification("notif-1", "Test", AudienceAll())); err != nil {
		t.Fatalf("Add: %s", err)
	}

	lis := manager.GetListener()
	defer lis.Close()

	if err := manager.Dismiss("notif-1", "alice"); err != nil {
		t.Fatalf("Dismiss: %s", err)
	}

	update := collectNotificationUpdates(t, lis, 1)[0]
	if update.Removed == nil || *update.Removed != "notif-1" {
		t.Fatalf("dismiss update: got %+v, want a removal", update)
	}
	if update.ForUser != "alice" {
		t.Errorf("ForUser: got %q, want %q - a dismissal must not reach anybody else", update.ForUser, "alice")
	}

	if got := manager.Snapshot("alice", auth.UserRole); len(got) != 0 {
		t.Errorf("alice snapshot: got %d, want 0", len(got))
	}
	if got := manager.Snapshot("bob", auth.UserRole); len(got) != 1 {
		t.Errorf("bob snapshot: got %d, want 1 - a dismissal is per-user", len(got))
	}

	// Dismissing implies reading, so the badge does not count a row the user
	// can no longer see.
	if got := manager.Unread("alice", auth.UserRole); got != 0 {
		t.Errorf("alice unread after dismiss: got %d, want 0", got)
	}
}

func TestNotificationManagerCollapseKeyReplaces(t *testing.T) {
	manager := newTestNotificationManager(t, stores.NewMemStore(), newTestUserManager(t))

	first := testNotification("notif-1", "Sensor offline", AudienceAll())
	first.CollapseKey = "device-offline:dev-1"
	if err := manager.Add(first); err != nil {
		t.Fatalf("Add first: %s", err)
	}

	lis := manager.GetListener()
	defer lis.Close()

	second := testNotification("notif-2", "Sensor offline again", AudienceAll())
	second.CollapseKey = "device-offline:dev-1"
	if err := manager.Add(second); err != nil {
		t.Fatalf("Add second: %s", err)
	}

	// The removal of the collapsed row is published before the replacement, so
	// a client never briefly shows both.
	updates := collectNotificationUpdates(t, lis, 2)
	if updates[0].Removed == nil || *updates[0].Removed != "notif-1" {
		t.Fatalf("first update: got %+v, want removal of notif-1", updates[0])
	}
	if updates[1].Updated == nil || updates[1].Updated.ID != "notif-2" {
		t.Fatalf("second update: got %+v, want notif-2", updates[1])
	}

	if got := manager.Snapshot("alice", auth.UserRole); len(got) != 1 || got[0].ID != "notif-2" {
		t.Fatalf("snapshot: got %+v, want only notif-2", got)
	}
}

func TestNotificationManagerTrimsByCount(t *testing.T) {
	manager := newTestNotificationManager(t, stores.NewMemStore(), newTestUserManager(t))

	lis := manager.GetListener()
	defer lis.Close()

	created := time.Now().UTC().Add(-time.Hour)
	for i := range maxNotifications + 5 {
		notification := testNotification(fmt.Sprintf("notif-%03d", i), "Test", AudienceAll())
		// Ascending timestamps, so the low-numbered ones are the oldest.
		notification.Created = created.Add(time.Duration(i) * time.Second)
		if err := manager.Add(notification); err != nil {
			t.Fatalf("Add %d: %s", i, err)
		}
	}

	snapshot := manager.Snapshot("alice", auth.UserRole)
	if len(snapshot) != maxNotifications {
		t.Fatalf("snapshot: got %d, want %d", len(snapshot), maxNotifications)
	}
	// Newest first.
	if snapshot[0].ID != fmt.Sprintf("notif-%03d", maxNotifications+4) {
		t.Errorf("newest: got %q", snapshot[0].ID)
	}
	// The five oldest are gone.
	for _, notification := range snapshot {
		if notification.ID <= "notif-004" {
			t.Errorf("notification %q should have been trimmed", notification.ID)
		}
	}

	// Every trim must be published - a long-lived client would otherwise keep
	// showing rows the server has forgotten.
	removed := map[string]bool{}
	deadline := time.After(2 * time.Second)
	for len(removed) < 5 {
		select {
		case update := <-lis.Sub():
			if update.Removed != nil {
				removed[*update.Removed] = true
			}
		case <-deadline:
			t.Fatalf("only saw %d of 5 trim removals: %v", len(removed), removed)
		}
	}
	for i := range 5 {
		id := fmt.Sprintf("notif-%03d", i)
		if !removed[id] {
			t.Errorf("no removal published for trimmed notification %q", id)
		}
	}
}

func TestNotificationManagerTrimsByAgeOnLoad(t *testing.T) {
	store := stores.NewMemStore()

	fresh := testNotification("notif-fresh", "Fresh", AudienceAll())
	stale := testNotification("notif-stale", "Stale", AudienceAll())
	stale.Created = time.Now().UTC().Add(-maxNotificationAge - time.Hour)

	seed := struct {
		Notifications []*Notification `json:"notifications"`
	}{Notifications: []*Notification{fresh, stale}}
	data, err := json.Marshal(seed)
	if err != nil {
		t.Fatalf("marshal: %s", err)
	}
	if err := store.Set("notifications.json", data); err != nil {
		t.Fatalf("Set: %s", err)
	}

	manager := newTestNotificationManager(t, store, newTestUserManager(t))

	snapshot := manager.Snapshot("alice", auth.UserRole)
	if len(snapshot) != 1 || snapshot[0].ID != "notif-fresh" {
		t.Fatalf("snapshot: got %+v, want only notif-fresh", snapshot)
	}
}

func TestNotificationManagerPrunesStateForRemovedUser(t *testing.T) {
	userManager := newTestUserManager(t)
	addTestUser(t, userManager, "alice", "a-password", false)

	manager := newTestNotificationManager(t, stores.NewMemStore(), userManager)

	if err := manager.Add(testNotification("notif-1", "Test", AudienceAll())); err != nil {
		t.Fatalf("Add: %s", err)
	}
	if err := manager.MarkRead("notif-1", "alice"); err != nil {
		t.Fatalf("MarkRead: %s", err)
	}

	if err := userManager.Delete("alice"); err != nil {
		t.Fatalf("Delete: %s", err)
	}

	// A recreated account of the same name must not inherit the old holder's
	// read state.
	deadline := time.Now().Add(2 * time.Second)
	for {
		snapshot := manager.Snapshot("alice", auth.UserRole)
		if len(snapshot) == 1 && !snapshot[0].IsReadBy("alice") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("read state for the removed user was never pruned")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestNotificationManagerPersistsAcrossRestart(t *testing.T) {
	store := stores.NewMemStore()

	manager := newTestNotificationManager(t, store, newTestUserManager(t))
	if err := manager.Add(testNotification("notif-1", "Test", AudienceAll())); err != nil {
		t.Fatalf("Add: %s", err)
	}
	if err := manager.MarkRead("notif-1", "alice"); err != nil {
		t.Fatalf("MarkRead: %s", err)
	}
	manager.Close()

	reloaded := newTestNotificationManager(t, store, newTestUserManager(t))

	snapshot := reloaded.Snapshot("alice", auth.UserRole)
	if len(snapshot) != 1 {
		t.Fatalf("snapshot: got %d, want 1", len(snapshot))
	}
	if !snapshot[0].IsReadBy("alice") {
		t.Error("alice's read state did not survive the restart")
	}
	if snapshot[0].IsReadBy("bob") {
		t.Error("bob's read state appeared from nowhere")
	}
	if snapshot[0].Title != "Test" {
		t.Errorf("title: got %q, want %q", snapshot[0].Title, "Test")
	}
}

func TestNotificationManagerSnapshotIsNewestFirst(t *testing.T) {
	manager := newTestNotificationManager(t, stores.NewMemStore(), newTestUserManager(t))

	for i := range 3 {
		notification := testNotification(fmt.Sprintf("notif-%d", i), "Test", AudienceAll())
		notification.Created = time.Now().UTC().Add(time.Duration(i) * time.Second)
		if err := manager.Add(notification); err != nil {
			t.Fatalf("Add %d: %s", i, err)
		}
	}

	snapshot := manager.Snapshot("alice", auth.UserRole)
	if len(snapshot) != 3 {
		t.Fatalf("snapshot: got %d, want 3", len(snapshot))
	}
	for i, notification := range snapshot {
		want := fmt.Sprintf("notif-%d", 2-i)
		if notification.ID != want {
			t.Errorf("snapshot[%d]: got %q, want %q", i, notification.ID, want)
		}
	}
}

// GetListener must never replay, and closing a listener must never wedge the
// manager. Both hold because the manager only ever publishes through Pub, which
// takes the publisher's lock and skips unregistered subscribers - unlike Send,
// which does neither and blocks forever on a closed subscriber's queue.
func TestNotificationManagerListenerDoesNotReplay(t *testing.T) {
	manager := newTestNotificationManager(t, stores.NewMemStore(), newTestUserManager(t))

	if err := manager.Add(testNotification("notif-1", "Before", AudienceAll())); err != nil {
		t.Fatalf("Add: %s", err)
	}

	lis := manager.GetListener()
	expectNoUpdate(t, lis)

	// Closing immediately, then continuing to publish, must not block.
	lis.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 5 {
			if err := manager.Add(testNotification(fmt.Sprintf("notif-after-%d", i), "After", AudienceAll())); err != nil {
				t.Errorf("Add after close: %s", err)
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publishing after a listener closed blocked")
	}
}

// stubDeliverer records what it was handed and can be told to fail.
type stubDeliverer struct {
	mu       sync.Mutex
	err      error
	received []string
}

func (d *stubDeliverer) Name() string { return "stub" }

func (d *stubDeliverer) Deliver(_ context.Context, notification *Notification) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.received = append(d.received, notification.ID)
	return d.err
}

func (d *stubDeliverer) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.received)
}

func TestNotificationManagerDelivererCalled(t *testing.T) {
	manager := newTestNotificationManager(t, stores.NewMemStore(), newTestUserManager(t))

	deliverer := &stubDeliverer{err: errors.New("transport unavailable")}
	manager.AddDeliverer(deliverer)

	// A failing transport must not fail the Add: the in-app stream is the
	// delivery path that matters and it has already succeeded.
	if err := manager.Add(testNotification("notif-1", "Test", AudienceAll())); err != nil {
		t.Fatalf("Add: %s", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for deliverer.count() < 1 {
		if time.Now().After(deadline) {
			t.Fatal("deliverer was never called")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := manager.Snapshot("alice", auth.UserRole); len(got) != 1 {
		t.Errorf("snapshot: got %d, want 1 - a delivery failure must not drop the notification", len(got))
	}
}
