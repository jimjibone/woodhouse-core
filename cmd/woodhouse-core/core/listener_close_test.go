package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

// Every manager that replays current state to a new listener does so from its
// run goroutine, with publisher.Send, while the listener's owner - a gRPC
// stream handler - may close it at any moment because its client disconnected.
//
// That combination used to wedge the manager permanently: Send bottoms out in
// Queue.Push, an unbuffered send served only by the queue's runloop, and Close
// stops that runloop. The push then had no reader and never returned, stranding
// the run goroutine that also serves every other subscriber, the save ticker
// and the manager's own shutdown. Closing the manager afterwards hung too,
// because Close waits on that goroutine.
//
// These tests take each manager, register a listener against a snapshot large
// enough that the close lands mid-replay, and assert the manager is still alive
// afterwards. "Still alive" is checked by closing it under a deadline, since a
// stranded run goroutine makes Close block forever.
//
// The fix is in the queue dependency (Push drops on a closed Queue rather than
// blocking), so these guard against a regression there as much as here.

// closeWithin fails unless the manager shuts down inside the deadline. A hang
// here means a run goroutine is stranded.
func closeWithin(t *testing.T, name string, closeManager func()) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		closeManager()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not shut down: its run goroutine is stranded", name)
	}
}

func TestZoneManagerSurvivesListenerClosedMidReplay(t *testing.T) {
	deviceIDs := make([]string, 0, 200)
	for i := range 200 {
		deviceIDs = append(deviceIDs, fmt.Sprintf("dev-%03d", i))
	}

	manager, err := NewZoneManager(stores.NewMemStore(), testZoneDeviceManager(deviceIDs...))
	if err != nil {
		t.Fatalf("NewZoneManager: %s", err)
	}

	for i, deviceID := range deviceIDs {
		zone := NewZone(fmt.Sprintf("zone-%03d", i), "Room", "sofa", []string{deviceID})
		if err := manager.AddZone(zone); err != nil {
			t.Fatalf("AddZone: %s", err)
		}
	}

	// Register and immediately abandon, the way a client that drops during the
	// initial batch does.
	lis := manager.GetListener()
	lis.Close()

	// The manager must still serve everybody else.
	survivor := manager.GetListener()
	defer survivor.Close()
	if err := manager.UpdateZoneName("zone-000", "Scullery"); err != nil {
		t.Fatalf("UpdateZoneName after a listener was abandoned: %s", err)
	}

	closeWithin(t, "ZoneManager", manager.Close)
}

func TestUserManagerSurvivesListenerClosedMidReplay(t *testing.T) {
	manager, err := NewUserManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("NewUserManager: %s", err)
	}

	// Enough users that a close reliably lands inside the replay. Passwords are
	// argon2-hashed, so keep the count modest and the work off the hot path.
	for i := range 40 {
		user, err := NewUser(fmt.Sprintf("user-%03d", i), "Test User", "a-password", auth.UserRole)
		if err != nil {
			t.Fatalf("NewUser: %s", err)
		}
		if err := manager.Store(user); err != nil {
			t.Fatalf("Store: %s", err)
		}
	}

	lis := manager.GetListener()
	lis.Close()

	if err := manager.SetFullname("user-000", "Renamed"); err != nil {
		t.Fatalf("SetFullname after a listener was abandoned: %s", err)
	}

	closeWithin(t, "UserManager", manager.Close)
}

func TestGroupManagerSurvivesListenerClosedMidReplay(t *testing.T) {
	deviceManager, err := NewDeviceManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("NewDeviceManager: %s", err)
	}
	defer deviceManager.Close()

	manager, err := NewGroupManager(stores.NewMemStore(), deviceManager)
	if err != nil {
		t.Fatalf("NewGroupManager: %s", err)
	}

	lis := manager.GetListener()
	lis.Close()

	closeWithin(t, "GroupManager", manager.Close)
}

func TestFavoritesManagerSurvivesListenerClosedMidReplay(t *testing.T) {
	deviceManager, err := NewDeviceManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("NewDeviceManager: %s", err)
	}
	defer deviceManager.Close()

	manager := NewFavoritesManager(stores.NewMemStore(), deviceManager)

	lis := manager.GetListener()
	lis.Close()

	closeWithin(t, "FavoritesManager", manager.Close)
}

func TestClientManagerSurvivesListenerClosedMidReplay(t *testing.T) {
	manager, err := NewClientManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("NewClientManager: %s", err)
	}

	// ClientManager runs two publishers off the one goroutine, so a strand on
	// either takes both streams down with it.
	clients := manager.GetClientListener()
	clients.Close()
	pairings := manager.GetPairingListener()
	pairings.Close()

	closeWithin(t, "ClientManager", manager.Close)
}
