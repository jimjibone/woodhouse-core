package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/jimjibone/log"
	"github.com/jimjibone/queue/v2"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

var (
	ErrZoneNotFound      = errors.New("zone not found")
	ErrZoneAlreadyExists = errors.New("zone already exists")
)

// ZoneManager owns the zones - the rooms and areas a device can be placed in.
//
// It is deliberately much smaller than GroupManager. A group aggregates service
// state and fans actions out, so it has to masquerade as a device; a zone is
// only ever metadata, so it never touches the DeviceManager beyond noticing
// that a device has gone away. Clients read zones from ZonesStream and join
// them against the devices stream themselves.
type ZoneManager struct {
	log           *log.Context
	wg            sync.WaitGroup
	ctx           context.Context
	close         func()
	store         stores.Store
	deviceManager *DeviceManager
	publisher     *queue.Pub[ZoneUpdate]
	listenerAdd   chan *queue.Sub[ZoneUpdate]

	mu      sync.RWMutex
	changed bool
	zones   map[string]*Zone
}

type ZoneUpdate struct {
	Updated *Zone
	Removed *string
}

func NewZoneManager(store stores.Store, deviceManager *DeviceManager) (*ZoneManager, error) {
	ctx, close := context.WithCancel(context.Background())
	manager := &ZoneManager{
		log:           log.NewContext(log.DefaultLogger, "zone-manager", log.DebugLevel),
		ctx:           ctx,
		close:         close,
		store:         store,
		deviceManager: deviceManager,
		publisher:     queue.NewPub[ZoneUpdate](),
		listenerAdd:   make(chan *queue.Sub[ZoneUpdate], 1),
		zones:         make(map[string]*Zone),
	}

	// Load the previous state.
	err := manager.load()
	if err != nil {
		close()
		return nil, fmt.Errorf("failed to load state: %s", err)
	}

	// Save the state if changed.
	err = manager.saveIfChanged()
	if err != nil {
		close()
		return nil, fmt.Errorf("failed to save state: %s", err)
	}

	// Subscribe before the goroutine starts. Subscribing inside run() leaves a
	// window between this constructor returning and the goroutine being
	// scheduled in which a device removal is published to nobody, and a missed
	// removal leaves a dangling device ID in a zone forever.
	deviceUpdates := deviceManager.GetDeviceUpdates()

	manager.wg.Add(1)
	go manager.run(ctx, deviceUpdates)
	return manager, nil
}

func (manager *ZoneManager) Close() {
	manager.close()
	manager.wg.Wait()

	// Flush on the way out. The 60s save ticker in run() has already stopped,
	// so without this an edit made in the last minute of uptime is lost.
	err := manager.saveIfChanged()
	if err != nil {
		manager.log.Errorf("failed to save state: %s", err)
	}
}

func (manager *ZoneManager) GetListener() *queue.Sub[ZoneUpdate] {
	sub := manager.publisher.NewSub()
	manager.listenerAdd <- sub
	return sub
}

// GetZoneForDevice reports the ID of the zone holding a device, or "" if it is
// unassigned.
func (manager *ZoneManager) GetZoneForDevice(deviceID string) string {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	for _, zone := range manager.zones {
		if zone.HasDevice(deviceID) {
			return zone.ZoneID
		}
	}
	return ""
}

// verifyDevices checks that every device ID names a device the system knows
// about, so a typo cannot leave a zone pointing at nothing.
func (manager *ZoneManager) verifyDevices(deviceIDs []string) error {
	seen := make(map[string]bool, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		if deviceID == "" {
			return fmt.Errorf("empty device id")
		}
		if seen[deviceID] {
			return fmt.Errorf("device %q listed more than once", deviceID)
		}
		seen[deviceID] = true

		if manager.deviceManager.GetDevice(deviceID) == nil {
			return fmt.Errorf("device %q not found", deviceID)
		}
	}
	return nil
}

// claimDevices enforces the one-zone-per-device invariant: it strips the given
// devices out of every zone other than claimant and publishes an update for
// each zone it changed. Callers must hold manager.mu.
//
// Publishing both sides matters - a client watching the stream needs to see the
// device leave its old zone, not just arrive in the new one.
func (manager *ZoneManager) claimDevices(claimantID string, deviceIDs []string) {
	for _, zone := range manager.zones {
		if zone.ZoneID == claimantID {
			continue
		}

		lost := false
		for _, deviceID := range deviceIDs {
			if zone.removeDevice(deviceID) {
				lost = true
			}
		}

		if lost {
			manager.changed = true
			manager.log.Infof("zone %q lost devices claimed by zone %q", zone.ZoneID, claimantID)
			manager.publisher.Pub(ZoneUpdate{Updated: zone.Clone()})
		}
	}
}

func (manager *ZoneManager) AddZone(zone *Zone) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.zones[zone.ZoneID] != nil {
		return ErrZoneAlreadyExists
	}

	if err := manager.verifyDevices(zone.DeviceIDs); err != nil {
		return err
	}

	// Take the devices off any zone that already holds them first, so no client
	// ever observes a device in two zones at once.
	manager.claimDevices(zone.ZoneID, zone.DeviceIDs)

	manager.zones[zone.ZoneID] = zone
	manager.changed = true

	manager.log.Infof("zone added %s", zone.String())

	manager.publisher.Pub(ZoneUpdate{Updated: zone.Clone()})

	return nil
}

func (manager *ZoneManager) UpdateZoneName(zoneID, name string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	zone := manager.zones[zoneID]
	if zone == nil {
		return ErrZoneNotFound
	}

	oldName := zone.Name
	zone.Name = name
	manager.changed = true

	manager.log.Infof("zone %q updated name from %q to %q", zone.ZoneID, oldName, name)

	manager.publisher.Pub(ZoneUpdate{Updated: zone.Clone()})

	return nil
}

func (manager *ZoneManager) UpdateZoneIcon(zoneID, icon string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	zone := manager.zones[zoneID]
	if zone == nil {
		return ErrZoneNotFound
	}

	oldIcon := zone.Icon
	zone.Icon = icon
	manager.changed = true

	manager.log.Infof("zone %q updated icon from %q to %q", zone.ZoneID, oldIcon, icon)

	manager.publisher.Pub(ZoneUpdate{Updated: zone.Clone()})

	return nil
}

// SetZoneDevices replaces a zone's membership in full. An empty list empties the
// zone - that is a legitimate edit, not a no-op.
func (manager *ZoneManager) SetZoneDevices(zoneID string, deviceIDs []string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	zone := manager.zones[zoneID]
	if zone == nil {
		return ErrZoneNotFound
	}

	if err := manager.verifyDevices(deviceIDs); err != nil {
		return err
	}

	manager.claimDevices(zoneID, deviceIDs)

	zone.DeviceIDs = slices.Clone(deviceIDs)
	manager.changed = true

	manager.log.Infof("zone %q updated devices: %d", zone.ZoneID, len(zone.DeviceIDs))

	manager.publisher.Pub(ZoneUpdate{Updated: zone.Clone()})

	return nil
}

// RemoveZone deletes a zone. Its devices simply become unassigned - there is no
// synthetic device to tear down, unlike a group.
func (manager *ZoneManager) RemoveZone(zoneID string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if _, found := manager.zones[zoneID]; !found {
		return ErrZoneNotFound
	}

	delete(manager.zones, zoneID)
	manager.changed = true

	manager.log.Infof("zone %q removed", zoneID)

	manager.publisher.Pub(ZoneUpdate{Removed: &zoneID})

	return nil
}

// removeDeviceFromZones drops a device that no longer exists from whichever
// zone held it, so zones never keep dangling IDs.
func (manager *ZoneManager) removeDeviceFromZones(deviceID string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	for _, zone := range manager.zones {
		if zone.removeDevice(deviceID) {
			manager.changed = true
			manager.log.Infof("zone %q dropped removed device %q", zone.ZoneID, deviceID)
			manager.publisher.Pub(ZoneUpdate{Updated: zone.Clone()})
		}
	}
}

func (manager *ZoneManager) load() error {
	if manager.store.Has("zones.json") {
		manager.log.Debugf("loading...")

		data, err := manager.store.Get("zones.json")
		if err != nil {
			return err
		}

		config := struct {
			Zones []*Zone `json:"zones"`
		}{}
		err = json.NewDecoder(bytes.NewReader(data)).Decode(&config)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		manager.zones = make(map[string]*Zone)
		for _, zone := range config.Zones {
			manager.zones[zone.ZoneID] = zone
		}
	}
	return nil
}

func (manager *ZoneManager) save() error {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	config := struct {
		Zones []*Zone `json:"zones"`
	}{}
	for _, zone := range manager.zones {
		config.Zones = append(config.Zones, zone)
	}
	// Sorted to maintain consistent structure between saves.
	sort.Slice(config.Zones, func(i, j int) bool {
		return config.Zones[i].ZoneID < config.Zones[j].ZoneID
	})

	data := &bytes.Buffer{}
	encoder := json.NewEncoder(data)
	encoder.SetIndent("", "\t")
	err := encoder.Encode(config)
	if err != nil {
		return err
	}

	return manager.store.Set("zones.json", data.Bytes())
}

func (manager *ZoneManager) saveIfChanged() error {
	if manager.changed {
		manager.log.Debugf("saving...")
		err := manager.save()
		if err != nil {
			return err
		}
		manager.changed = false
	}
	return nil
}

func (manager *ZoneManager) run(ctx context.Context, deviceUpdates *queue.Sub[txDeviceUpdate]) {
	defer manager.wg.Done()
	defer deviceUpdates.Close()

	manager.mu.RLock()
	manager.log.Debugf("initial zones are: %d", len(manager.zones))
	for _, zone := range manager.zones {
		manager.log.Debugf("  %s", zone)
	}
	manager.mu.RUnlock()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case update := <-deviceUpdates.Sub():
			// Zones hold no device state, so an ordinary update is nothing to
			// us. A removal is - it would otherwise leave a dangling ID.
			if update.RemovedID != "" {
				manager.removeDeviceFromZones(update.RemovedID)
			}

		case lis := <-manager.listenerAdd:
			// Replay the current state to the new listener.
			manager.mu.RLock()
			for _, zone := range manager.zones {
				manager.publisher.Send(lis, ZoneUpdate{Updated: zone.Clone()})
			}
			manager.mu.RUnlock()

			// An empty update indicates the end of the initial list.
			manager.publisher.Send(lis, ZoneUpdate{})

		case <-ticker.C:
			err := manager.saveIfChanged()
			if err != nil {
				manager.log.Errorf("failed to save state: %s", err)
			}
		}
	}
}
