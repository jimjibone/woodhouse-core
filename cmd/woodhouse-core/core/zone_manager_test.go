package core

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jimjibone/queue/v2"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

// testZoneDeviceManager builds a DeviceManager holding the named devices
// without starting its goroutines, so zone tests exercise only the zone code.
func testZoneDeviceManager(deviceIDs ...string) *DeviceManager {
	devices := make(map[string]*Device, len(deviceIDs))
	for _, id := range deviceIDs {
		devices[id] = &Device{
			ClientID:  "client-1",
			ID:        id,
			Typ:       clientsapi.Device_DEVICE,
			Endpoints: map[string]*DeviceEndpoint{},
		}
	}
	return &DeviceManager{
		devices:         devices,
		txDeviceUpdates: queue.NewPub[txDeviceUpdate](),
	}
}

func newTestZoneManager(t *testing.T, store stores.Store, deviceIDs ...string) *ZoneManager {
	t.Helper()

	manager, err := NewZoneManager(store, testZoneDeviceManager(deviceIDs...))
	if err != nil {
		t.Fatalf("NewZoneManager: %s", err)
	}
	t.Cleanup(manager.Close)
	return manager
}

// zoneByID reads a zone back out of the manager the way save() sees it.
func zoneByID(t *testing.T, manager *ZoneManager, zoneID string) *Zone {
	t.Helper()

	manager.mu.RLock()
	defer manager.mu.RUnlock()

	zone, found := manager.zones[zoneID]
	if !found {
		t.Fatalf("zone %q not found", zoneID)
	}
	return zone.Clone()
}

func TestZoneManagerAddUpdateRemove(t *testing.T) {
	manager := newTestZoneManager(t, stores.NewMemStore(), "dev-1", "dev-2")

	if err := manager.AddZone(NewZone("zone-1", "Kitchen", "utensils-crossed", []string{"dev-1"})); err != nil {
		t.Fatalf("AddZone: %s", err)
	}

	if err := manager.AddZone(NewZone("zone-1", "Kitchen again", "sofa", nil)); err != ErrZoneAlreadyExists {
		t.Fatalf("AddZone duplicate: got %v, want ErrZoneAlreadyExists", err)
	}

	if err := manager.UpdateZoneName("zone-1", "Scullery"); err != nil {
		t.Fatalf("UpdateZoneName: %s", err)
	}
	if err := manager.UpdateZoneIcon("zone-1", "cooking-pot"); err != nil {
		t.Fatalf("UpdateZoneIcon: %s", err)
	}

	zone := zoneByID(t, manager, "zone-1")
	if zone.Name != "Scullery" {
		t.Errorf("name: got %q, want %q", zone.Name, "Scullery")
	}
	if zone.Icon != "cooking-pot" {
		t.Errorf("icon: got %q, want %q", zone.Icon, "cooking-pot")
	}

	if err := manager.UpdateZoneName("nope", "Nowhere"); err != ErrZoneNotFound {
		t.Errorf("UpdateZoneName unknown: got %v, want ErrZoneNotFound", err)
	}

	if err := manager.RemoveZone("zone-1"); err != nil {
		t.Fatalf("RemoveZone: %s", err)
	}
	if err := manager.RemoveZone("zone-1"); err != ErrZoneNotFound {
		t.Errorf("RemoveZone twice: got %v, want ErrZoneNotFound", err)
	}
}

func TestZoneManagerRejectsUnknownDevice(t *testing.T) {
	manager := newTestZoneManager(t, stores.NewMemStore(), "dev-1")

	if err := manager.AddZone(NewZone("zone-1", "Kitchen", "sofa", []string{"ghost"})); err == nil {
		t.Fatal("AddZone with an unknown device: got nil, want an error")
	}

	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if len(manager.zones) != 0 {
		t.Errorf("a rejected zone was stored anyway: %d zones", len(manager.zones))
	}
}

// A device lives in exactly one zone. Claiming it for a second zone has to take
// it off the first, and both sides have to reach listeners - a client that only
// heard about the arrival would show the device twice.
func TestZoneManagerDeviceBelongsToOneZone(t *testing.T) {
	manager := newTestZoneManager(t, stores.NewMemStore(), "dev-1", "dev-2")

	if err := manager.AddZone(NewZone("zone-1", "Kitchen", "utensils", []string{"dev-1", "dev-2"})); err != nil {
		t.Fatalf("AddZone zone-1: %s", err)
	}

	lis := manager.GetListener()
	defer lis.Close()
	drainInitialZones(t, lis)

	if err := manager.AddZone(NewZone("zone-2", "Living Room", "sofa", []string{"dev-1"})); err != nil {
		t.Fatalf("AddZone zone-2: %s", err)
	}

	if got := zoneByID(t, manager, "zone-1").DeviceIDs; !slices.Equal(got, []string{"dev-2"}) {
		t.Errorf("zone-1 devices: got %v, want [dev-2]", got)
	}
	if got := zoneByID(t, manager, "zone-2").DeviceIDs; !slices.Equal(got, []string{"dev-1"}) {
		t.Errorf("zone-2 devices: got %v, want [dev-1]", got)
	}

	// Both the losing and the gaining zone must be published, in that order.
	updated := collectZoneUpdates(t, lis, 2)
	if len(updated) != 2 {
		t.Fatalf("got %d updates, want 2", len(updated))
	}
	if updated[0].ZoneID != "zone-1" || !slices.Equal(updated[0].DeviceIDs, []string{"dev-2"}) {
		t.Errorf("first update: got %s", updated[0])
	}
	if updated[1].ZoneID != "zone-2" {
		t.Errorf("second update: got %s", updated[1])
	}

	if got := manager.GetZoneForDevice("dev-1"); got != "zone-2" {
		t.Errorf("GetZoneForDevice(dev-1): got %q, want %q", got, "zone-2")
	}
	if got := manager.GetZoneForDevice("ghost"); got != "" {
		t.Errorf("GetZoneForDevice(ghost): got %q, want %q", got, "")
	}
}

// An empty device list is a real edit - it empties the zone. This is the case
// UpdateGroupRequest cannot express, and the reason UpdateZoneRequest carries
// an explicit set_devices flag.
func TestZoneManagerSetZoneDevicesEmpties(t *testing.T) {
	manager := newTestZoneManager(t, stores.NewMemStore(), "dev-1", "dev-2")

	if err := manager.AddZone(NewZone("zone-1", "Kitchen", "utensils", []string{"dev-1", "dev-2"})); err != nil {
		t.Fatalf("AddZone: %s", err)
	}

	if err := manager.SetZoneDevices("zone-1", nil); err != nil {
		t.Fatalf("SetZoneDevices: %s", err)
	}

	if got := zoneByID(t, manager, "zone-1").DeviceIDs; len(got) != 0 {
		t.Errorf("devices after clearing: got %v, want empty", got)
	}
	if got := manager.GetZoneForDevice("dev-1"); got != "" {
		t.Errorf("GetZoneForDevice(dev-1): got %q, want unassigned", got)
	}
}

func TestZoneManagerRejectsDuplicateDeviceInOneZone(t *testing.T) {
	manager := newTestZoneManager(t, stores.NewMemStore(), "dev-1")

	err := manager.AddZone(NewZone("zone-1", "Kitchen", "utensils", []string{"dev-1", "dev-1"}))
	if err == nil {
		t.Fatal("AddZone with a repeated device: got nil, want an error")
	}
}

// A removed device must not linger as a dangling ID in whichever zone held it.
func TestZoneManagerDropsRemovedDevice(t *testing.T) {
	deviceManager := testZoneDeviceManager("dev-1", "dev-2")
	manager, err := NewZoneManager(stores.NewMemStore(), deviceManager)
	if err != nil {
		t.Fatalf("NewZoneManager: %s", err)
	}
	defer manager.Close()

	if err := manager.AddZone(NewZone("zone-1", "Kitchen", "utensils", []string{"dev-1", "dev-2"})); err != nil {
		t.Fatalf("AddZone: %s", err)
	}

	deviceManager.txDeviceUpdates.Pub(txDeviceUpdate{RemovedID: "dev-1"})

	deadline := time.Now().Add(2 * time.Second)
	for {
		if got := zoneByID(t, manager, "zone-1").DeviceIDs; slices.Equal(got, []string{"dev-2"}) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("zone still holds the removed device: %v", zoneByID(t, manager, "zone-1").DeviceIDs)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestZoneManagerPersistsAcrossRestart(t *testing.T) {
	store := stores.NewMemStore()

	manager := newTestZoneManager(t, store, "dev-1", "dev-2")
	if err := manager.AddZone(NewZone("zone-1", "Kitchen", "utensils-crossed", []string{"dev-1"})); err != nil {
		t.Fatalf("AddZone: %s", err)
	}
	if err := manager.AddZone(NewZone("zone-2", "Garden", "trees", []string{"dev-2"})); err != nil {
		t.Fatalf("AddZone: %s", err)
	}

	// Close() has to flush: the 60s save ticker will not have fired.
	manager.Close()

	if !store.Has("zones.json") {
		t.Fatal("zones.json was not written")
	}

	data, err := store.Get("zones.json")
	if err != nil {
		t.Fatalf("Get zones.json: %s", err)
	}
	saved := struct {
		Zones []*Zone `json:"zones"`
	}{}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal zones.json: %s", err)
	}
	// Sorted by ID so the file does not churn between saves.
	if len(saved.Zones) != 2 || saved.Zones[0].ZoneID != "zone-1" || saved.Zones[1].ZoneID != "zone-2" {
		t.Fatalf("zones.json: got %+v, want zone-1 then zone-2", saved.Zones)
	}

	reloaded := newTestZoneManager(t, store, "dev-1", "dev-2")
	zone := zoneByID(t, reloaded, "zone-1")
	if zone.Name != "Kitchen" || zone.Icon != "utensils-crossed" || !slices.Equal(zone.DeviceIDs, []string{"dev-1"}) {
		t.Errorf("reloaded zone-1: got %s", zone)
	}
}

// drainInitialZones consumes the replay of current state plus the empty
// end-of-batch sentinel a new listener receives.
func drainInitialZones(t *testing.T, lis *queue.Sub[ZoneUpdate]) {
	t.Helper()

	for {
		select {
		case update := <-lis.Sub():
			if update.Updated == nil && update.Removed == nil {
				return
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out draining the initial zone batch")
		}
	}
}

func collectZoneUpdates(t *testing.T, lis *queue.Sub[ZoneUpdate], want int) []*Zone {
	t.Helper()

	var zones []*Zone
	for len(zones) < want {
		select {
		case update := <-lis.Sub():
			if update.Updated != nil {
				zones = append(zones, update.Updated)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for zone updates: got %d, want %d", len(zones), want)
		}
	}
	return zones
}

// testZoneDeviceManagerEndpoints is testZoneDeviceManager for devices that
// have endpoints, given as device ID to endpoint IDs.
func testZoneDeviceManagerEndpoints(endpoints map[string][]string) *DeviceManager {
	manager := testZoneDeviceManager()
	for id, epIDs := range endpoints {
		dev := &Device{
			ClientID:  "client-1",
			ID:        id,
			Typ:       clientsapi.Device_DEVICE,
			Endpoints: map[string]*DeviceEndpoint{},
		}
		for _, epID := range epIDs {
			dev.Endpoints[epID] = &DeviceEndpoint{ID: epID, Services: map[string]*clientsapi.Service{}}
		}
		manager.devices[id] = dev
	}
	return manager
}

func TestZoneManagerEndpointOverride(t *testing.T) {
	deviceManager := testZoneDeviceManagerEndpoints(map[string][]string{"dev-1": {"0", "1", "2"}, "dev-2": {"0", "1"}})
	manager, err := NewZoneManager(stores.NewMemStore(), deviceManager)
	if err != nil {
		t.Fatalf("NewZoneManager: %s", err)
	}
	defer manager.Close()

	if err := manager.AddZone(NewZone("zone-1", "Kitchen", "utensils", []string{"dev-1"})); err != nil {
		t.Fatalf("AddZone zone-1: %s", err)
	}
	ref := EndpointRef{DeviceID: "dev-1", EndpointID: "2"}
	if err := manager.AddZone(NewZone("zone-2", "Garden", "trees", nil, ref)); err != nil {
		t.Fatalf("AddZone zone-2: %s", err)
	}

	for epID, want := range map[string]string{"0": "zone-1", "1": "zone-1", "2": "zone-2", "9": "zone-1"} {
		if got := manager.GetZoneForEndpoint("dev-1", epID); got != want {
			t.Errorf("GetZoneForEndpoint(dev-1, %q): got %q, want %q", epID, got, want)
		}
	}
	if got := manager.GetZoneForEndpoint("dev-2", "1"); got != "" {
		t.Errorf("GetZoneForEndpoint(dev-2, 1): got %q, want unassigned", got)
	}
	if got := manager.GetZoneForDevice("dev-1"); got != "zone-1" {
		t.Errorf("GetZoneForDevice(dev-1): got %q, want zone-1", got)
	}
	if got := zoneByID(t, manager, "zone-2").Pb().GetEndpoints(); len(got) != 1 || got[0].GetDeviceId() != "dev-1" || got[0].GetEndpointId() != "2" {
		t.Errorf("Pb endpoints: got %v", got)
	}
}

func TestZoneManagerRejectsUnknownEndpoint(t *testing.T) {
	deviceManager := testZoneDeviceManagerEndpoints(map[string][]string{"dev-1": {"0", "1"}})
	manager, err := NewZoneManager(stores.NewMemStore(), deviceManager)
	if err != nil {
		t.Fatalf("NewZoneManager: %s", err)
	}
	defer manager.Close()

	for name, ref := range map[string]EndpointRef{
		"unknown device":   {DeviceID: "ghost", EndpointID: "0"},
		"unknown endpoint": {DeviceID: "dev-1", EndpointID: "7"},
		"empty device":     {DeviceID: "", EndpointID: "0"},
		"empty endpoint":   {DeviceID: "dev-1", EndpointID: ""},
	} {
		if err := manager.AddZone(NewZone("zone-1", "Kitchen", "sofa", nil, ref)); err == nil {
			t.Errorf("AddZone with %s: got nil, want an error", name)
		}
	}

	dup := EndpointRef{DeviceID: "dev-1", EndpointID: "1"}
	if err := manager.AddZone(NewZone("zone-1", "Kitchen", "sofa", nil, dup, dup)); err == nil {
		t.Error("AddZone with a repeated endpoint: got nil, want an error")
	}

	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if len(manager.zones) != 0 {
		t.Errorf("a rejected zone was stored anyway: %d zones", len(manager.zones))
	}
}

// An endpoint is overridden into one zone only. Claiming it for a second zone
// takes it off the first, and the loser reaches listeners.
func TestZoneManagerEndpointBelongsToOneZone(t *testing.T) {
	deviceManager := testZoneDeviceManagerEndpoints(map[string][]string{"dev-1": {"0", "1", "2"}})
	manager, err := NewZoneManager(stores.NewMemStore(), deviceManager)
	if err != nil {
		t.Fatalf("NewZoneManager: %s", err)
	}
	defer manager.Close()

	ep1 := EndpointRef{DeviceID: "dev-1", EndpointID: "1"}
	ep2 := EndpointRef{DeviceID: "dev-1", EndpointID: "2"}
	if err := manager.AddZone(NewZone("zone-1", "Kitchen", "utensils", nil, ep1, ep2)); err != nil {
		t.Fatalf("AddZone zone-1: %s", err)
	}

	lis := manager.GetListener()
	defer lis.Close()
	drainInitialZones(t, lis)

	if err := manager.AddZone(NewZone("zone-2", "Garden", "trees", nil, ep2)); err != nil {
		t.Fatalf("AddZone zone-2: %s", err)
	}

	updates := collectZoneUpdates(t, lis, 2)
	if updates[0].ZoneID != "zone-1" || !slices.Equal(updates[0].Endpoints, []EndpointRef{ep1}) {
		t.Errorf("first update: got %s, want zone-1 holding only endpoint 1", updates[0])
	}
	if updates[1].ZoneID != "zone-2" {
		t.Errorf("second update: got %s, want zone-2", updates[1])
	}
	if got := manager.GetZoneForEndpoint("dev-1", "2"); got != "zone-2" {
		t.Errorf("GetZoneForEndpoint(dev-1, 2): got %q, want zone-2", got)
	}

	// SetZoneEndpoints claims the same way.
	if err := manager.SetZoneEndpoints("zone-1", []EndpointRef{ep2}); err != nil {
		t.Fatalf("SetZoneEndpoints: %s", err)
	}
	if got := zoneByID(t, manager, "zone-2").Endpoints; len(got) != 0 {
		t.Errorf("zone-2 endpoints after zone-1 claimed: got %v, want none", got)
	}
}

// An empty list is a real edit - it clears the overrides.
func TestZoneManagerSetZoneEndpointsClears(t *testing.T) {
	deviceManager := testZoneDeviceManagerEndpoints(map[string][]string{"dev-1": {"0", "1"}})
	manager, err := NewZoneManager(stores.NewMemStore(), deviceManager)
	if err != nil {
		t.Fatalf("NewZoneManager: %s", err)
	}
	defer manager.Close()

	ref := EndpointRef{DeviceID: "dev-1", EndpointID: "1"}
	if err := manager.AddZone(NewZone("zone-1", "Kitchen", "utensils", nil, ref)); err != nil {
		t.Fatalf("AddZone: %s", err)
	}
	if err := manager.SetZoneEndpoints("zone-1", nil); err != nil {
		t.Fatalf("SetZoneEndpoints: %s", err)
	}
	if got := zoneByID(t, manager, "zone-1").Endpoints; len(got) != 0 {
		t.Errorf("endpoints after clearing: got %v, want none", got)
	}
	if err := manager.SetZoneEndpoints("nope", nil); err != ErrZoneNotFound {
		t.Errorf("SetZoneEndpoints unknown zone: got %v, want ErrZoneNotFound", err)
	}
	if err := manager.SetZoneEndpoints("zone-1", []EndpointRef{{DeviceID: "dev-1", EndpointID: "7"}}); err == nil {
		t.Error("SetZoneEndpoints with an unknown endpoint: got nil, want an error")
	}
}

func waitForZoneEndpoints(t *testing.T, manager *ZoneManager, zoneID string, want []EndpointRef) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		got := zoneByID(t, manager, zoneID).Endpoints
		if slices.Equal(got, want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("zone %q endpoints: got %v, want %v", zoneID, got, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Removing a device drops its endpoint overrides, and a full state that no
// longer reports an endpoint drops that endpoint's override. Partial updates
// leave overrides alone.
func TestZoneManagerDropsGoneEndpoints(t *testing.T) {
	deviceManager := testZoneDeviceManagerEndpoints(map[string][]string{"dev-1": {"0", "1", "2"}, "dev-2": {"0", "1"}})
	manager, err := NewZoneManager(stores.NewMemStore(), deviceManager)
	if err != nil {
		t.Fatalf("NewZoneManager: %s", err)
	}
	defer manager.Close()

	d1e1 := EndpointRef{DeviceID: "dev-1", EndpointID: "1"}
	d1e2 := EndpointRef{DeviceID: "dev-1", EndpointID: "2"}
	d2e1 := EndpointRef{DeviceID: "dev-2", EndpointID: "1"}
	if err := manager.AddZone(NewZone("zone-1", "Kitchen", "utensils", nil, d1e1, d1e2, d2e1)); err != nil {
		t.Fatalf("AddZone: %s", err)
	}

	// A partial update without endpoint 2 changes nothing.
	deviceManager.txDeviceUpdates.Pub(txDeviceUpdate{Update: &clientsapi.Device{
		Id:        "dev-1",
		Endpoints: []*clientsapi.Endpoint{{Id: "1"}},
	}})
	// A full state without endpoint 2 drops it, for that device only.
	deviceManager.txDeviceUpdates.Pub(txDeviceUpdate{Update: &clientsapi.Device{
		Id:        "dev-1",
		FullState: true,
		Endpoints: []*clientsapi.Endpoint{{Id: "0"}, {Id: "1"}},
	}})
	waitForZoneEndpoints(t, manager, "zone-1", []EndpointRef{d1e1, d2e1})

	deviceManager.txDeviceUpdates.Pub(txDeviceUpdate{RemovedID: "dev-1"})
	waitForZoneEndpoints(t, manager, "zone-1", []EndpointRef{d2e1})
}

func TestZoneManagerPersistsEndpoints(t *testing.T) {
	store := stores.NewMemStore()
	deviceManager := testZoneDeviceManagerEndpoints(map[string][]string{"dev-1": {"0", "1"}})

	manager, err := NewZoneManager(store, deviceManager)
	if err != nil {
		t.Fatalf("NewZoneManager: %s", err)
	}
	ref := EndpointRef{DeviceID: "dev-1", EndpointID: "1"}
	if err := manager.AddZone(NewZone("zone-1", "Kitchen", "utensils", nil, ref)); err != nil {
		t.Fatalf("AddZone: %s", err)
	}
	manager.Close()

	data, err := store.Get("zones.json")
	if err != nil {
		t.Fatalf("Get zones.json: %s", err)
	}
	if !strings.Contains(string(data), `"endpoints"`) {
		t.Errorf("zones.json has no endpoints field: %s", data)
	}

	reloaded, err := NewZoneManager(store, deviceManager)
	if err != nil {
		t.Fatalf("NewZoneManager reload: %s", err)
	}
	defer reloaded.Close()
	if got := zoneByID(t, reloaded, "zone-1").Endpoints; !slices.Equal(got, []EndpointRef{ref}) {
		t.Errorf("reloaded endpoints: got %v, want %v", got, []EndpointRef{ref})
	}
}
