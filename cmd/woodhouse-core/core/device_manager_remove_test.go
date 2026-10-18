package core

import (
	"testing"
	"time"

	"github.com/jimjibone/queue/v2"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

func fullStateDevice(id string, online bool) *clientsapi.Device {
	return &clientsapi.Device{
		Id:        id,
		FullState: true,
		Typ:       clientsapi.Device_DEVICE,
		Services: []*clientsapi.Service{
			{
				Id:  "info",
				Typ: clientsapi.Service_INFO,
				Attrs: []*clientsapi.Attribute{
					{Id: "name", Text: &clientsapi.TextAttribute{Value: id, Perms: clientsapi.Permissions_PERM_READONLY}},
				},
			},
			{
				Id:  "online",
				Typ: clientsapi.Service_ONLINE,
				Attrs: []*clientsapi.Attribute{
					{Id: "online", Bool: &clientsapi.BoolAttribute{Value: online, Perms: clientsapi.Permissions_PERM_READONLY}},
				},
			},
		},
	}
}

// waitRemoved waits for the removal of id, failing if any other device is
// removed first.
func waitRemoved(t *testing.T, sub *queue.Sub[txDeviceUpdate], id string) {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case update := <-sub.Sub():
			if update.RemovedID == "" {
				continue
			}
			if update.RemovedID != id {
				t.Fatalf("device %q removed while waiting for %q", update.RemovedID, id)
			}
			return
		case <-timeout:
			t.Fatalf("timed out waiting for removal of %q", id)
		}
	}
}

func TestRemoveClientDevice(t *testing.T) {
	manager, err := NewDeviceManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("NewDeviceManager: %s", err)
	}
	defer manager.Close()

	sub := manager.GetDeviceUpdates()
	defer sub.Close()

	manager.PushDeviceUpdate("client-1", fullStateDevice("dev-a", true))
	manager.PushDeviceUpdate("client-1", fullStateDevice("dev-c", true))
	manager.PushDeviceUpdate("client-2", fullStateDevice("dev-b", true))

	// Online device owned by the client is removed. Queued straight after its
	// creation, so this also checks the removal is not reordered.
	manager.RemoveClientDevice("client-1", "dev-a")
	waitRemoved(t, sub, "dev-a")
	if manager.GetDevice("dev-a") != nil {
		t.Errorf("dev-a still present after removal")
	}

	// Devices owned by another client and unknown devices are ignored. The
	// final removal is a marker: once it is seen the others were processed.
	manager.RemoveClientDevice("client-1", "dev-b")
	manager.RemoveClientDevice("client-1", "dev-unknown")
	manager.RemoveClientDevice("client-1", "dev-c")
	waitRemoved(t, sub, "dev-c")
	if manager.GetDevice("dev-b") == nil {
		t.Errorf("dev-b owned by client-2 was removed by client-1")
	}
	if manager.GetDevice("dev-c") != nil {
		t.Errorf("dev-c still present after removal")
	}
}

// A device saved before a restart keeps its owner, so its client can remove it
// without it reporting again first.
func TestRemoveClientDeviceAfterRestart(t *testing.T) {
	store := stores.NewMemStore()

	manager, err := NewDeviceManager(store)
	if err != nil {
		t.Fatalf("NewDeviceManager: %s", err)
	}
	sub := manager.GetDeviceUpdates()
	manager.PushDeviceUpdate("client-1", fullStateDevice("dev-a", true))
	manager.PushDeviceUpdate("client-1", fullStateDevice("dev-b", true))
	manager.PushDeviceUpdate("client-1", fullStateDevice("dev-marker", true))
	manager.RemoveClientDevice("client-1", "dev-marker")
	waitRemoved(t, sub, "dev-marker")
	sub.Close()
	manager.Close()

	manager, err = NewDeviceManager(store)
	if err != nil {
		t.Fatalf("NewDeviceManager after restart: %s", err)
	}
	defer manager.Close()
	if got := manager.GetDevice("dev-a").GetClientId(); got != "client-1" {
		t.Fatalf("dev-a client after restart = %q, want client-1", got)
	}

	sub = manager.GetDeviceUpdates()
	defer sub.Close()
	manager.RemoveClientDevice("client-2", "dev-a")
	manager.RemoveClientDevice("client-1", "dev-b")
	waitRemoved(t, sub, "dev-b")
	if manager.GetDevice("dev-a") == nil {
		t.Errorf("dev-a removed by client-2 after restart")
	}
}

func TestRemoveClientDeviceThenReAdd(t *testing.T) {
	manager, err := NewDeviceManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("NewDeviceManager: %s", err)
	}
	defer manager.Close()

	sub := manager.GetDeviceUpdates()
	defer sub.Close()

	manager.PushDeviceUpdate("client-1", fullStateDevice("dev-a", true))
	manager.RemoveClientDevice("client-1", "dev-a")
	waitRemoved(t, sub, "dev-a")

	// The same device paired again is created afresh.
	manager.PushDeviceUpdate("client-1", fullStateDevice("dev-a", true))
	manager.PushDeviceUpdate("client-1", fullStateDevice("dev-marker", true))
	manager.RemoveClientDevice("client-1", "dev-marker")
	waitRemoved(t, sub, "dev-marker")
	if manager.GetDevice("dev-a") == nil {
		t.Errorf("dev-a not re-created after removal")
	}
}
