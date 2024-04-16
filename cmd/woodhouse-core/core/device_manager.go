package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/jimjibone/queue/v2"
	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"gopkg.in/yaml.v3"
)

type DeviceManager struct {
	log             *log.Context
	wg              sync.WaitGroup
	close           func()
	store           stores.Store
	rxDeviceUpdates *queue.Queue[deviceUpdate]
	txDeviceUpdates *queue.Pub[*clientsapi.Device]
	devices         map[string]*Device // key=device id
	changed         bool
}

type deviceUpdate struct {
	ClientID string
	Update   *clientsapi.Device
	Offline  bool
}

func NewDeviceManager(store stores.Store) (*DeviceManager, error) {
	ctx, close := context.WithCancel(context.Background())
	manager := &DeviceManager{
		log:             log.NewContext(log.DefaultLogger, "device-manager", log.DebugLevel),
		close:           close,
		store:           store,
		rxDeviceUpdates: queue.New[deviceUpdate](),
		txDeviceUpdates: queue.NewPub[*clientsapi.Device](),
		devices:         make(map[string]*Device),
	}

	// Load the previous state.
	err := manager.load()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %s", err)
	}

	// Save the state if changed.
	err = manager.saveIfChanged()
	if err != nil {
		return nil, fmt.Errorf("failed to save state: %s", err)
	}

	manager.wg.Add(1)
	go manager.run(ctx)
	return manager, nil
}

func (manager *DeviceManager) Close() {
	manager.close()
	manager.wg.Wait()

	err := manager.saveIfChanged()
	if err != nil {
		manager.log.Errorf("failed to save state: %s", err)
	}
}

func (manager *DeviceManager) PushDeviceUpdate(clientID string, update *clientsapi.Device) {
	manager.rxDeviceUpdates.Push(deviceUpdate{ClientID: clientID, Update: update})
}

func (manager *DeviceManager) SetClientOffline(clientID string) {
	manager.rxDeviceUpdates.Push(deviceUpdate{ClientID: clientID, Offline: true})
}

func (manager *DeviceManager) load() error {
	if manager.store.Has("devices") {
		manager.log.Debugf("loading...")

		// Load it.
		data, err := manager.store.Get("devices")
		if err != nil {
			return err
		}

		// Decode it.
		config := struct {
			Devices []*clientsapi.Device `json:"devices"`
		}{}
		err = json.NewDecoder(bytes.NewReader(data)).Decode(&config)
		// err = yaml.NewDecoder(bytes.NewReader(data)).Decode(&config)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			if te, ok := err.(*yaml.TypeError); ok {
				fmt.Fprintln(os.Stderr, te.Errors)
			}
			// fmt.Println(yaml.FormatError(err, true, true))
			return err
		}

		// Read the state into the manager (convert slice to map).
		manager.devices = make(map[string]*Device)
		for _, upd := range config.Devices {
			dev, err := newDevice(manager.log, "", upd)
			if err != nil {
				manager.log.Warnf("failed to load device %q: %s", upd.GetId(), err)
			} else {
				manager.devices[dev.ID] = dev
			}
		}
	}
	return nil
}

func (manager *DeviceManager) save() error {
	// Convert map to slice.
	config := struct {
		Devices []*clientsapi.Device `json:"devices"`
	}{}
	for _, dev := range manager.devices {
		config.Devices = append(config.Devices, dev.pb())
	}
	// Sorted to maintain consistent structure between saves.
	sort.Slice(config.Devices, func(i, j int) bool {
		return config.Devices[i].GetId() < config.Devices[j].GetId()
	})

	// Encode it.
	data := &bytes.Buffer{}
	err := json.NewEncoder(data).Encode(config)
	// err := yaml.NewEncoder(data).Encode(config)
	if err != nil {
		return err
	}

	// Save it.
	err = manager.store.Set("devices", data.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (manager *DeviceManager) saveIfChanged() error {
	// Save the config if changed.
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

func (manager *DeviceManager) run(ctx context.Context) {
	defer manager.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case update := <-manager.rxDeviceUpdates.Pop():
			if update.ClientID != "" {
				if update.Update != nil {
					if update.Update.GetId() != "" {
						if dev, found := manager.devices[update.Update.GetId()]; found {
							err := dev.update(manager.log, update.ClientID, update.Update)
							if err != nil {
								manager.log.Warnf("failed to update device %q: %s", update.Update.GetId(), err)
							} else {
								manager.changed = true
							}
						} else {
							dev, err := newDevice(manager.log, update.ClientID, update.Update)
							if err != nil {
								manager.log.Warnf("failed to create device %q: %s", update.Update.GetId(), err)
							} else {
								manager.devices[update.Update.GetId()] = dev
								manager.changed = true
							}
						}

						manager.txDeviceUpdates.Pub(update.Update)
					} else {
						manager.log.Warnf("device updated has empty device ID: %s", update)
					}
				} else if update.Offline {
					manager.log.Debugf("client %q has gone offline", update.ClientID)
					for _, dev := range manager.devices {
						if dev.ClientID == update.ClientID {
							offlineUpdate := dev.setOffline(manager.log)
							if offlineUpdate != nil {
								manager.changed = true
								manager.txDeviceUpdates.Pub(offlineUpdate)
							}
						}
					}
				} else {
					manager.log.Warnf("device update was empty: %s", update)
				}
			} else {
				manager.log.Warnf("device updated has empty client ID: %s", update)
			}

		case <-ticker.C:
			err := manager.saveIfChanged()
			if err != nil {
				manager.log.Fatalf("failed to save state: %s", err)
			}
		}
	}
}
