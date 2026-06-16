package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/jimjibone/log"
	"github.com/jimjibone/queue/v2"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/shared/random"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"
)

var (
	ErrDeviceNotFound = errors.New("device not found")
	ErrDeviceIsOnline = errors.New("device is online")
	ErrDeviceIsGroup  = errors.New("device is a group")
)

type DeviceManager struct {
	log               *log.Context
	wg                sync.WaitGroup
	mu                sync.RWMutex
	ctx               context.Context
	close             func()
	store             stores.Store
	rxDeviceUpdates   *queue.Queue[rxDeviceUpdate]
	txDeviceUpdates   *queue.Pub[txDeviceUpdate]
	actionRequests    *queue.Pub[ActionRequest]
	actionResponses   *queue.Pub[ActionResponse]
	imageRequests     *queue.Pub[ImageRequest]
	imageResponses    *queue.Pub[ImageResponse]
	imageCacheMu      sync.RWMutex
	imageCache        map[string]*CachedImage // key = deviceID+":"+serviceID+":"+attributeID
	imageCacheUpdates *queue.Pub[*CachedImage]
	rxSetFavourites   *queue.Queue[favoriteUpdate]
	rxRemoveDevices   *queue.Queue[removalUpdate]
	devices           map[string]*Device // key=device id
	changed           bool
}

type rxDeviceUpdate struct {
	ClientID string
	Update   *clientsapi.Device
	Offline  bool
}

type txDeviceUpdate struct {
	ClientID  string             // The client that caused the update.
	Update    *clientsapi.Device // The updated device.
	RemovedID string             // If the device was removed, the ID of the removed device.
}

type favoriteUpdate struct {
	DeviceID  string
	ServiceID string
	Favorite  bool
}

type removalUpdate struct {
	DeviceID string
	Force    bool
	Callback func(error)
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

// CachedImage holds the most recently fetched image for a camera attribute.
type CachedImage struct {
	DeviceID    string
	ServiceID   string
	AttributeID string
	Data        []byte
	MimeType    string
	FetchedAt   time.Time
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
		log:               log.NewContext(log.DefaultLogger, "device-manager", log.DebugLevel),
		ctx:               ctx,
		close:             close,
		store:             store,
		rxDeviceUpdates:   queue.New[rxDeviceUpdate](),
		txDeviceUpdates:   queue.NewPub[txDeviceUpdate](),
		actionRequests:    queue.NewPub[ActionRequest](),
		actionResponses:   queue.NewPub[ActionResponse](),
		imageRequests:     queue.NewPub[ImageRequest](),
		imageResponses:    queue.NewPub[ImageResponse](),
		imageCache:        make(map[string]*CachedImage),
		imageCacheUpdates: queue.NewPub[*CachedImage](),
		rxSetFavourites:   queue.New[favoriteUpdate](),
		rxRemoveDevices:   queue.New[removalUpdate](),
		devices:           make(map[string]*Device),
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

	manager.wg.Add(2)
	go manager.run(ctx)
	go manager.runImageScheduler(ctx)
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

func (manager *DeviceManager) SetFavorite(deviceID, serviceID string, favorite bool) {
	manager.rxSetFavourites.Push(favoriteUpdate{DeviceID: deviceID, ServiceID: serviceID, Favorite: favorite})
}

// Remove a device from the manager. Devices are not normally removable if they
// are online to avoid state inconsistencies if the client is still active. If
// force is true then the device will be removed. This is normally only true
// when removing groups.
func (manager *DeviceManager) RemoveDevice(deviceID string, force bool) error {
	err := make(chan error, 1)
	manager.rxRemoveDevices.Push(removalUpdate{
		DeviceID: deviceID,
		Force:    force,
		Callback: func(e error) {
			err <- e
		},
	})
	return <-err
}

func (manager *DeviceManager) PushDeviceUpdate(clientID string, update *clientsapi.Device) {
	manager.rxDeviceUpdates.Push(rxDeviceUpdate{ClientID: clientID, Update: update})
}

func (manager *DeviceManager) SetClientOffline(clientID string) {
	manager.rxDeviceUpdates.Push(rxDeviceUpdate{ClientID: clientID, Offline: true})
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

// UpdateImageCache stores a newly fetched image in the cache and publishes it
// to all ImagesStream subscribers.
func (manager *DeviceManager) UpdateImageCache(deviceID, serviceID, attributeID string, data []byte, mimeType string) {
	key := deviceID + ":" + serviceID + ":" + attributeID
	cached := &CachedImage{
		DeviceID:    deviceID,
		ServiceID:   serviceID,
		AttributeID: attributeID,
		Data:        data,
		MimeType:    mimeType,
		FetchedAt:   time.Now(),
	}
	manager.imageCacheMu.Lock()
	manager.imageCache[key] = cached
	manager.imageCacheMu.Unlock()
	manager.imageCacheUpdates.Pub(cached)
}

// GetCachedImages returns a channel that yields all currently cached images
// then closes.
func (manager *DeviceManager) GetCachedImages() <-chan *CachedImage {
	manager.imageCacheMu.RLock()
	defer manager.imageCacheMu.RUnlock()

	ch := make(chan *CachedImage, len(manager.imageCache))
	for _, img := range manager.imageCache {
		ch <- img
	}
	close(ch)
	return ch
}

// GetImageCacheUpdates returns a subscription that receives new CachedImage
// entries whenever the cache is updated.
func (manager *DeviceManager) GetImageCacheUpdates() *queue.Sub[*CachedImage] {
	return manager.imageCacheUpdates.NewSub()
}

type CameraDevice struct {
	DeviceID    string
	ServiceID   string
	AttributeID string
	ClientID    string
}

// GetCameraDevices returns a snapshot of all online devices that have at least
// one Camera service, along with the attribute ID and client ID needed to
// request an image.
func (manager *DeviceManager) GetCameraDevices() []CameraDevice {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	var devices []CameraDevice
	for _, dev := range manager.devices {
		if !dev.isOnline() {
			continue
		}
		for _, svc := range dev.Services {
			if svc.GetTyp() != clientsapi.Service_CAMERA {
				continue
			}
			for _, attr := range svc.GetAttrs() {
				if attr.GetImage() != nil {
					devices = append(devices, CameraDevice{
						DeviceID:    dev.ID,
						ServiceID:   svc.GetId(),
						AttributeID: attr.GetId(),
						ClientID:    dev.ClientID,
					})
				}
			}
		}
	}
	return devices
}

// Get the device by ID. Returns nil if the device was not found.
func (manager *DeviceManager) GetDevice(id string) *clientsapi.Device {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	if dev, found := manager.devices[id]; found {
		return dev.pb()
	}

	return nil
}

// Get a channel on which all devices will be sent. The channel will close when
// all devices have been sent.
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

// Get a channel on which all devices will be sent. The channel will close when
// all devices have been sent.
func (manager *DeviceManager) GetDevicesByIDs(ids []string) <-chan *clientsapi.Device {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	ch := make(chan *clientsapi.Device, len(ids))
	for _, id := range ids {
		if dev, found := manager.devices[id]; found {
			ch <- dev.pb()
		}
	}
	close(ch)

	return ch
}

func (manager *DeviceManager) GetDeviceUpdates() *queue.Sub[txDeviceUpdate] {
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

	// if !dev.isOnline() {
	// 	return "", "", status.Error(codes.Unavailable, "device is offline")
	// }

	actionID, err = random.GenerateRandomPin(10)
	if err != nil {
		manager.log.Errorf("failed to generate action id: %s", err)
		return "", "", status.Error(codes.Internal, "failed to generate action id")
	}

	return actionID, dev.ClientID, nil
}

func (manager *DeviceManager) load() error {
	// Migrate 'devices' to 'devices.json'.
	if manager.store.Has("devices") {
		manager.log.Infof(`migrating "devices" store to "devices.json"`)
		data, err := manager.store.Get("devices")
		if err != nil {
			return err
		}
		err = manager.store.Set("devices.json", data)
		if err != nil {
			return err
		}
		manager.store.Del("devices")
		if err != nil {
			return err
		}
	}

	// Read devices store.
	if manager.store.Has("devices.json") {
		manager.log.Debugf("loading...")

		// Load it.
		data, err := manager.store.Get("devices.json")
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
	encoder := json.NewEncoder(data)
	encoder.SetIndent("", "\t")
	err := encoder.Encode(config)
	// err := yaml.NewEncoder(data).Encode(config)
	if err != nil {
		return err
	}

	// Save it.
	err = manager.store.Set("devices.json", data.Bytes())
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

		case update := <-manager.rxSetFavourites.Pop():
			manager.handleFavoriteUpdate(update)

		case update := <-manager.rxRemoveDevices.Pop():
			manager.handleDeviceRemoval(update)

		case <-ticker.C:
			err := manager.saveIfChanged()
			if err != nil {
				manager.log.Fatalf("failed to save state: %s", err)
			}
		}
	}
}

func (manager *DeviceManager) handleDeviceUpdate(update rxDeviceUpdate) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if update.ClientID != "" {
		if update.Update != nil {
			if update.Update.GetId() != "" {
				var changes *clientsapi.Device
				if dev, found := manager.devices[update.Update.GetId()]; found {
					var err error
					changes, err = dev.update(manager.log, update.ClientID, update.Update)
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
						changes = dev.pb()
					}
				}

				manager.txDeviceUpdates.Pub(txDeviceUpdate{
					Update: changes,
				})
			} else {
				manager.log.Warnf("device updated has empty device ID: %s", update)
			}
		} else if update.Offline {
			manager.log.Infof("client %q has gone offline", update.ClientID)
			for _, dev := range manager.devices {
				if dev.ClientID == update.ClientID {
					changes := dev.setOffline(manager.log)
					if changes != nil {
						manager.changed = true
						manager.txDeviceUpdates.Pub(txDeviceUpdate{
							Update: changes,
						})
					}
				}
			}
		} else {
			manager.log.Warnf("device update was empty: %s", update)
		}
	} else {
		manager.log.Warnf("device update has empty client ID: %s", update)
	}
}

func (manager *DeviceManager) handleFavoriteUpdate(update favoriteUpdate) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if dev, found := manager.devices[update.DeviceID]; found {
		err := dev.updateFavoriteService(manager.log, update.ServiceID, update.Favorite)
		if err != nil {
			manager.log.Warnf("failed to update favorite device %q, service %q: %s", update.DeviceID, update.ServiceID, err)
		} else {
			manager.changed = true
			manager.txDeviceUpdates.Pub(txDeviceUpdate{
				Update: dev.pb(),
			})
		}
	} else {
		manager.log.Warnf("favorite update device ID not found: %s", update)
	}
}

func (manager *DeviceManager) handleDeviceRemoval(update removalUpdate) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// TODO: inform clients that the device was removed.
	// TODO: inform favorites manager that the device was removed.

	if dev, found := manager.devices[update.DeviceID]; found {
		if dev.ClientID == GroupClientID && !update.Force {
			update.Callback(ErrDeviceIsGroup)
			return
		}
		if dev.isOnline() && !update.Force {
			update.Callback(ErrDeviceIsOnline)
			return
		}
		manager.changed = true
		manager.txDeviceUpdates.Pub(txDeviceUpdate{
			RemovedID: dev.ID,
		})
		delete(manager.devices, dev.ID)
		manager.log.Infof("removed device %q", dev.ID)
		update.Callback(nil)
	} else {
		manager.log.Warnf("removal device ID not found: %s", update)
		update.Callback(ErrDeviceNotFound)
	}
}

func (manager *DeviceManager) runImageScheduler(ctx context.Context) {
	defer manager.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			manager.pollAllCameras(ctx)
		}
	}
}

// pollAllCameras fires an ImageRequest for every online camera device and
// stores the result in the image cache.
func (manager *DeviceManager) pollAllCameras(ctx context.Context) {
	devices := manager.GetCameraDevices()
	for _, dev := range devices {
		go manager.pollCamera(ctx, dev)
	}
}

func (manager *DeviceManager) pollCamera(ctx context.Context, dev CameraDevice) {
	requestID, err := random.GenerateRandomPin(10)
	if err != nil {
		manager.log.Errorf("image scheduler: failed to generate request id: %s", err)
		return
	}

	req := &clientsapi.ImageRequest{
		RequestId:   requestID,
		DeviceId:    dev.DeviceID,
		ServiceId:   dev.ServiceID,
		AttributeId: dev.AttributeID,
	}

	sub := manager.GetImageResponses()
	defer sub.Close()

	manager.PushImageRequest(dev.ClientID, req)

	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout.C:
			manager.log.Warnf("image scheduler: request %q for device %q timed out", requestID, dev.DeviceID)
			return
		case update := <-sub.Sub():
			if update.ClientID != dev.ClientID {
				continue
			}
			if update.Offline {
				manager.log.Debugf("image scheduler: client for device %q went offline", dev.DeviceID)
				return
			}
			if update.Response == nil || update.Response.GetRequestId() != requestID {
				continue
			}
			if update.Response.Status == clientsapi.ImageResponse_COMPLETE {
				if len(update.Response.Data) > 0 {
					manager.log.Debugf("image scheduler: cached image for device %q (%d bytes)", dev.DeviceID, len(update.Response.Data))
					manager.UpdateImageCache(dev.DeviceID, dev.ServiceID, dev.AttributeID, update.Response.Data, "image/jpeg")
				}
				return
			}
			if update.Response.Status >= clientsapi.ImageResponse_TIMEOUT {
				manager.log.Warnf("image scheduler: request %q for device %q failed: %s", requestID, dev.DeviceID, update.Response.Details)
				return
			}
		}
	}
}
