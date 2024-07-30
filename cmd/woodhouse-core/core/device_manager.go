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
	"github.com/jimjibone/woodhouse-4/shared/random"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"
)

type DeviceManager struct {
	log             *log.Context
	wg              sync.WaitGroup
	mu              sync.RWMutex
	ctx             context.Context
	close           func()
	store           stores.Store
	rxDeviceUpdates *queue.Queue[deviceUpdate]
	txDeviceUpdates *queue.Pub[*clientsapi.Device]
	actionRequests  *queue.Pub[ActionRequest]
	actionResponses *queue.Pub[ActionResponse]
	imageRequests   *queue.Pub[ImageRequest]
	imageResponses  *queue.Pub[ImageResponse]
	devices         map[string]*Device // key=device id
	changed         bool
}

type deviceUpdate struct {
	ClientID string
	Update   *clientsapi.Device
	Offline  bool
}

type ActionRequest struct {
	ClientID string
	Request  *clientsapi.ActionRequest
}

type ActionResponse struct {
	ClientID string
	Response *clientsapi.ActionResponse
	Offline  bool
}

type ImageRequest struct {
	ClientID string
	Request  *clientsapi.ImageRequest
}

type ImageResponse struct {
	ClientID string
	Response *clientsapi.ImageResponse
	Offline  bool
}

func (ar ActionResponse) String() string {
	if ar.Response != nil {
		return fmt.Sprintf("client_id:%q response:{%s}", ar.ClientID, ar.Response)
	}
	return fmt.Sprintf("client_id:%q offline:%t", ar.ClientID, ar.Offline)
}

func NewDeviceManager(store stores.Store) (*DeviceManager, error) {
	ctx, close := context.WithCancel(context.Background())
	manager := &DeviceManager{
		log:             log.NewContext(log.DefaultLogger, "device-manager", log.DebugLevel),
		ctx:             ctx,
		close:           close,
		store:           store,
		rxDeviceUpdates: queue.New[deviceUpdate](),
		txDeviceUpdates: queue.NewPub[*clientsapi.Device](),
		actionRequests:  queue.NewPub[ActionRequest](),
		actionResponses: queue.NewPub[ActionResponse](),
		imageRequests:   queue.NewPub[ImageRequest](),
		imageResponses:  queue.NewPub[ImageResponse](),
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

	// Set all devices as offline.
	for _, dev := range manager.devices {
		dev.setOffline(manager.log)
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

func (manager *DeviceManager) PushActionRequest(clientID string, req *clientsapi.ActionRequest) {
	manager.actionRequests.Pub(ActionRequest{
		ClientID: clientID,
		Request:  req,
	})
}

func (manager *DeviceManager) GetActionRequests() *queue.Sub[ActionRequest] {
	return manager.actionRequests.NewSub()
}

func (manager *DeviceManager) PushActionResponse(clientID string, res *clientsapi.ActionResponse, offline bool) {
	manager.actionResponses.Pub(ActionResponse{
		ClientID: clientID,
		Response: res,
		Offline:  offline,
	})
}

func (manager *DeviceManager) GetActionResponses() *queue.Sub[ActionResponse] {
	return manager.actionResponses.NewSub()
}

func (manager *DeviceManager) PushImageRequest(clientID string, req *clientsapi.ImageRequest) {
	manager.imageRequests.Pub(ImageRequest{
		ClientID: clientID,
		Request:  req,
	})
}

func (manager *DeviceManager) GetImageRequests() *queue.Sub[ImageRequest] {
	return manager.imageRequests.NewSub()
}

func (manager *DeviceManager) PushImageResponse(clientID string, res *clientsapi.ImageResponse, offline bool) {
	manager.imageResponses.Pub(ImageResponse{
		ClientID: clientID,
		Response: res,
		Offline:  offline,
	})
}

func (manager *DeviceManager) GetImageResponses() *queue.Sub[ImageResponse] {
	return manager.imageResponses.NewSub()
}

func (manager *DeviceManager) GetDevices() <-chan *clientsapi.Device {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	ch := make(chan *clientsapi.Device, len(manager.devices))
	for _, dev := range manager.devices {
		ch <- dev.pb()
	}
	close(ch)

	return ch
}

func (manager *DeviceManager) GetDeviceUpdates() *queue.Sub[*clientsapi.Device] {
	return manager.txDeviceUpdates.NewSub()
}

// Checks that the device exists, it's online and the client is known. Generates
// a new action ID and the client ID for the requested device.
func (manager *DeviceManager) PrepAction(deviceID string) (actionID, clientID string, err error) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	// Check that device exists.
	// Check that device is online and the client is known.
	// Generate an action ID.
	// Push the request onto the actionRequest pub queue.
	// Listen on actionResponse sub queue.
	// Add option to cancel the request from here.

	dev, found := manager.devices[deviceID]
	if !found {
		return "", "", status.Error(codes.NotFound, "device not found")
	}

	if dev.ClientID == "" {
		return "", "", status.Error(codes.Unavailable, "device has no client")
	}

	if !dev.isOnline() {
		return "", "", status.Error(codes.Unavailable, "device is offline")
	}

	actionID, err = random.GenerateRandomPin(10)
	if err != nil {
		manager.log.Errorf("failed to generate action id: %s", err)
		return "", "", status.Error(codes.Internal, "failed to generate action id")
	}

	return actionID, dev.ClientID, nil
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
	manager.mu.RLock()
	defer manager.mu.RUnlock()

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
			manager.handleDeviceUpdate(update)

		case <-ticker.C:
			err := manager.saveIfChanged()
			if err != nil {
				manager.log.Fatalf("failed to save state: %s", err)
			}
		}
	}
}

func (manager *DeviceManager) handleDeviceUpdate(update deviceUpdate) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

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
			manager.log.Infof("client %q has gone offline", update.ClientID)
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
}
