package core

import (
	"strings"
	"testing"
	"time"

	"github.com/jimjibone/queue/v2"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

// labelTestDevice is a device with the root endpoint and a labelled endpoint
// "1" holding a light.
func labelTestDevice(id string) *clientsapi.Device {
	dev := fullStateDevice(id, true)
	dev.Endpoints = append(dev.Endpoints, &clientsapi.Endpoint{
		Id:    "1",
		Label: "Output 1",
		Services: []*clientsapi.Service{{
			Id:  "light",
			Typ: clientsapi.Service_LIGHT,
			Attrs: []*clientsapi.Attribute{
				boolAttr("on", false, clientsapi.Permissions_PERM_READWRITE),
			},
		}},
	})
	return dev
}

// newLabelTestManager returns a device manager holding dev-a and dev-b.
func newLabelTestManager(t *testing.T, store stores.Store) *DeviceManager {
	t.Helper()

	manager, err := NewDeviceManager(store)
	if err != nil {
		t.Fatalf("NewDeviceManager: %s", err)
	}
	t.Cleanup(manager.Close)

	if len(manager.devices) == 0 {
		sub := manager.GetDeviceUpdates()
		defer sub.Close()
		manager.PushDeviceUpdate("client-1", labelTestDevice("dev-a"))
		manager.PushDeviceUpdate("client-1", labelTestDevice("dev-b"))
		for i := 0; i < 2; i++ {
			select {
			case <-sub.Sub():
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for devices")
			}
		}
	}
	return manager
}

func userLabelOf(dev *clientsapi.Device, endpointID string) string {
	for _, ep := range dev.GetEndpoints() {
		if ep.GetId() == endpointID {
			return ep.GetUserLabel()
		}
	}
	return ""
}

func TestSetEndpointLabel(t *testing.T) {
	manager := newLabelTestManager(t, stores.NewMemStore())

	if err := manager.SetEndpointLabel("ghost", "1", "x"); err != ErrDeviceNotFound {
		t.Errorf("unknown device: got %v, want ErrDeviceNotFound", err)
	}
	if err := manager.SetEndpointLabel("dev-a", "9", "x"); err != ErrEndpointNotFound {
		t.Errorf("unknown endpoint: got %v, want ErrEndpointNotFound", err)
	}
	if got := manager.EndpointUserLabel("dev-a", "1"); got != "" {
		t.Errorf("label after failed sets: got %q, want none", got)
	}

	if err := manager.SetEndpointLabel("dev-a", "1", "Left"); err != nil {
		t.Fatalf("SetEndpointLabel: %s", err)
	}
	if got := manager.EndpointUserLabel("dev-a", "1"); got != "Left" {
		t.Errorf("label: got %q, want Left", got)
	}
	if got := manager.EndpointUserLabel("dev-b", "1"); got != "" {
		t.Errorf("other device label: got %q, want none", got)
	}

	// Labels are applied to what is sent to users, and the shared original is
	// left alone.
	dev := manager.GetDevice("dev-a")
	labelled := manager.ApplyUserLabels(dev)
	if got := userLabelOf(labelled, "1"); got != "Left" {
		t.Errorf("labelled endpoint 1: got %q, want Left", got)
	}
	if got := userLabelOf(labelled, RootEndpointID); got != "" {
		t.Errorf("labelled root endpoint: got %q, want none", got)
	}
	if got := userLabelOf(dev, "1"); got != "" {
		t.Errorf("original was modified: got %q", got)
	}
	if got := manager.ApplyUserLabels(manager.GetDevice("dev-b")); userLabelOf(got, "1") != "" {
		t.Errorf("unlabelled device got a label: %v", got)
	}
	// The bridge's label is untouched.
	for _, ep := range labelled.GetEndpoints() {
		if ep.GetId() == "1" && ep.GetLabel() != "Output 1" {
			t.Errorf("bridge label: got %q, want Output 1", ep.GetLabel())
		}
	}

	// An empty label clears it.
	if err := manager.SetEndpointLabel("dev-a", "1", ""); err != nil {
		t.Fatalf("SetEndpointLabel clear: %s", err)
	}
	if got := manager.EndpointUserLabel("dev-a", "1"); got != "" {
		t.Errorf("label after clearing: got %q, want none", got)
	}
	if got := userLabelOf(manager.ApplyUserLabels(manager.GetDevice("dev-a")), "1"); got != "" {
		t.Errorf("labelled endpoint after clearing: got %q, want none", got)
	}
}

// A label change is published as a user-only partial update holding the
// endpoint and no services, and a no-op change is not published.
func TestSetEndpointLabelPublishesUpdate(t *testing.T) {
	manager := newLabelTestManager(t, stores.NewMemStore())
	sub := manager.GetDeviceUpdates()
	defer sub.Close()

	if err := manager.SetEndpointLabel("dev-a", "1", "Left"); err != nil {
		t.Fatalf("SetEndpointLabel: %s", err)
	}
	if err := manager.SetEndpointLabel("dev-a", "1", "Left"); err != nil {
		t.Fatalf("SetEndpointLabel again: %s", err)
	}
	if err := manager.SetEndpointLabel("dev-a", "1", ""); err != nil {
		t.Fatalf("SetEndpointLabel clear: %s", err)
	}

	for i := 0; i < 2; i++ {
		select {
		case update := <-sub.Sub():
			dev := update.Update
			if !update.UserOnly || dev == nil || dev.GetId() != "dev-a" || dev.GetFullState() || len(dev.GetEvents()) != 0 {
				t.Fatalf("update %d = %+v", i, update)
			}
			if eps := dev.GetEndpoints(); len(eps) != 1 || eps[0].GetId() != "1" || eps[0].GetLabel() != "Output 1" ||
				eps[0].GetUserLabel() != "" || len(eps[0].GetServices()) != 0 {
				t.Errorf("update %d endpoints = %v", i, eps)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for update %d", i)
		}
	}
	select {
	case update := <-sub.Sub():
		t.Errorf("unexpected extra update: %+v", update)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestEndpointLabelsPersist(t *testing.T) {
	store := stores.NewMemStore()
	manager := newLabelTestManager(t, store)
	if err := manager.SetEndpointLabel("dev-a", "1", "Left"); err != nil {
		t.Fatalf("SetEndpointLabel: %s", err)
	}
	manager.Close()

	data, err := store.Get(endpointLabelsKey)
	if err != nil {
		t.Fatalf("Get %s: %s", endpointLabelsKey, err)
	}
	if !strings.Contains(string(data), "Left") {
		t.Errorf("%s = %s", endpointLabelsKey, data)
	}
	// Labels never go in the devices.
	devices, err := store.Get("devices.json")
	if err != nil {
		t.Fatalf("Get devices.json: %s", err)
	}
	if strings.Contains(string(devices), "Left") || strings.Contains(string(devices), "userLabel") {
		t.Errorf("devices.json holds a user label: %s", devices)
	}

	reloaded := newLabelTestManager(t, store)
	if got := reloaded.EndpointUserLabel("dev-a", "1"); got != "Left" {
		t.Errorf("reloaded label: got %q, want Left", got)
	}
}

func TestEndpointLabelsDroppedWithDevice(t *testing.T) {
	store := stores.NewMemStore()
	manager := newLabelTestManager(t, store)
	sub := manager.GetDeviceUpdates()
	defer sub.Close()

	for _, id := range []string{"dev-a", "dev-b"} {
		if err := manager.SetEndpointLabel(id, "1", "Left"); err != nil {
			t.Fatalf("SetEndpointLabel: %s", err)
		}
	}
	manager.RemoveClientDevice("client-1", "dev-a")
	waitRemoved(t, sub, "dev-a")
	if got := manager.EndpointUserLabel("dev-a", "1"); got != "" {
		t.Errorf("label of removed device: got %q, want none", got)
	}
	if got := manager.EndpointUserLabel("dev-b", "1"); got != "Left" {
		t.Errorf("label of kept device: got %q, want Left", got)
	}

	// A device that comes back starts without the label.
	manager.PushDeviceUpdate("client-1", labelTestDevice("dev-a"))
	deadline := time.Now().Add(5 * time.Second)
	for manager.GetDevice("dev-a") == nil {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for dev-a")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := manager.EndpointUserLabel("dev-a", "1"); got != "" {
		t.Errorf("label of re-added device: got %q, want none", got)
	}

	manager.Close()
	data, err := store.Get(endpointLabelsKey)
	if err != nil {
		t.Fatalf("Get %s: %s", endpointLabelsKey, err)
	}
	if strings.Contains(string(data), "dev-a") {
		t.Errorf("%s still holds the removed device: %s", endpointLabelsKey, data)
	}
}

// waitFavouriteLabel reads favourite updates until one reports the label. A
// favourite can be sent more than once, so others are skipped.
func waitFavouriteLabel(t *testing.T, lis *queue.Sub[FavoriteUpdate], want string) {
	t.Helper()

	timeout := time.After(5 * time.Second)
	for {
		select {
		case update := <-lis.Sub():
			if update.Updated != nil && update.Updated.Pb().GetEndpointLabel() == want {
				return
			}
		case <-timeout:
			t.Fatalf("timed out waiting for a favourite with label %q", want)
		}
	}
}

// A favourite reports the user's label if set, else the bridge's, and a label
// change updates open favourites streams.
func TestFavouriteEndpointLabel(t *testing.T) {
	deviceManager := newLabelTestManager(t, stores.NewMemStore())
	favourites := NewFavoritesManager(stores.NewMemStore(), deviceManager)
	defer favourites.Close()

	deviceManager.SetFavorite("dev-a", "1", "light", true)

	lis := favourites.GetListener()
	defer lis.Close()
	waitFavouriteLabel(t, lis, "Output 1")

	if err := deviceManager.SetEndpointLabel("dev-a", "1", "Left"); err != nil {
		t.Fatalf("SetEndpointLabel: %s", err)
	}
	waitFavouriteLabel(t, lis, "Left")

	if err := deviceManager.SetEndpointLabel("dev-a", "1", ""); err != nil {
		t.Fatalf("SetEndpointLabel clear: %s", err)
	}
	waitFavouriteLabel(t, lis, "Output 1")
}
