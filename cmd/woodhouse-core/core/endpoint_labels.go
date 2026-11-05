package core

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"google.golang.org/protobuf/proto"
)

// endpointLabelsKey is the store key of the user's endpoint labels. They are
// kept apart from devices.json, which holds what bridges report.
const endpointLabelsKey = "endpoint-labels.json"

// endpointLabel is the persisted form of one label.
type endpointLabel struct {
	DeviceID   string `json:"device_id"`
	EndpointID string `json:"endpoint_id"`
	Label      string `json:"label"`
}

// SetEndpointLabel sets the user's label for an endpoint, which overrides the
// label the bridge reports. An empty label clears it. Returns ErrDeviceNotFound
// or ErrEndpointNotFound for an unknown endpoint.
//
// The change is published as a device update carrying only the endpoint and its
// bridge label. It carries no user label, as the update is only for users:
// user streams apply the labels themselves (see ApplyUserLabels) and favourites
// look the label up.
func (manager *DeviceManager) SetEndpointLabel(deviceID, endpointID, label string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	dev, found := manager.devices[deviceID]
	if !found {
		return ErrDeviceNotFound
	}
	ep, found := dev.Endpoints[endpointID]
	if !found {
		return ErrEndpointNotFound
	}

	if manager.labels[deviceID][endpointID] == label {
		return nil
	}
	if label == "" {
		delete(manager.labels[deviceID], endpointID)
		if len(manager.labels[deviceID]) == 0 {
			delete(manager.labels, deviceID)
		}
	} else {
		if manager.labels == nil {
			manager.labels = make(map[string]map[string]string)
		}
		if manager.labels[deviceID] == nil {
			manager.labels[deviceID] = make(map[string]string)
		}
		manager.labels[deviceID][endpointID] = label
	}
	manager.labelsChanged = true
	manager.log.Infof("device %q endpoint %q user label set to %q", deviceID, endpointID, label)

	manager.txDeviceUpdates.Pub(txDeviceUpdate{
		UserOnly: true,
		Update: &clientsapi.Device{
			Id:        dev.ID,
			ClientId:  dev.ClientID,
			Typ:       dev.Typ,
			Endpoints: []*clientsapi.Endpoint{{Id: ep.ID, Label: ep.Label}},
		},
	})
	return nil
}

// EndpointUserLabel returns the user's label for an endpoint, or "" if none.
func (manager *DeviceManager) EndpointUserLabel(deviceID, endpointID string) string {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	return manager.labels[deviceID][endpointID]
}

// ApplyUserLabels sets Endpoint.user_label on the endpoints of a device sent to
// a user. The device may be shared, so it is not modified; a copy is returned
// when the device has labels.
func (manager *DeviceManager) ApplyUserLabels(dev *clientsapi.Device) *clientsapi.Device {
	if dev == nil {
		return nil
	}

	manager.mu.RLock()
	labels := manager.labels[dev.GetId()]
	manager.mu.RUnlock()

	// With no labels the endpoints already have none, which is also how a
	// cleared label reaches the user.
	if len(labels) == 0 {
		return dev
	}
	labelled := proto.Clone(dev).(*clientsapi.Device)
	for _, ep := range labelled.GetEndpoints() {
		ep.UserLabel = labels[ep.GetId()]
	}
	return labelled
}

// removeLabels drops the labels of a removed device. Callers must hold mu.
func (manager *DeviceManager) removeLabels(deviceID string) {
	if _, found := manager.labels[deviceID]; found {
		delete(manager.labels, deviceID)
		manager.labelsChanged = true
	}
}

func (manager *DeviceManager) loadLabels() error {
	if !manager.store.Has(endpointLabelsKey) {
		return nil
	}
	manager.log.Debugf("loading endpoint labels...")

	data, err := manager.store.Get(endpointLabelsKey)
	if err != nil {
		return err
	}
	config := struct {
		Labels []endpointLabel `json:"labels"`
	}{}
	err = json.NewDecoder(bytes.NewReader(data)).Decode(&config)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	manager.labels = make(map[string]map[string]string)
	for _, l := range config.Labels {
		if l.Label == "" {
			continue
		}
		if manager.labels[l.DeviceID] == nil {
			manager.labels[l.DeviceID] = make(map[string]string)
		}
		manager.labels[l.DeviceID][l.EndpointID] = l.Label
	}
	return nil
}

// saveLabelsIfChanged writes the labels if they changed since the last save.
func (manager *DeviceManager) saveLabelsIfChanged() error {
	manager.mu.Lock()
	if !manager.labelsChanged {
		manager.mu.Unlock()
		return nil
	}
	manager.labelsChanged = false
	config := struct {
		Labels []endpointLabel `json:"labels"`
	}{Labels: []endpointLabel{}}
	for deviceID, labels := range manager.labels {
		for endpointID, label := range labels {
			config.Labels = append(config.Labels, endpointLabel{DeviceID: deviceID, EndpointID: endpointID, Label: label})
		}
	}
	manager.mu.Unlock()

	// Sorted to maintain consistent structure between saves.
	sort.Slice(config.Labels, func(i, j int) bool {
		if config.Labels[i].DeviceID != config.Labels[j].DeviceID {
			return config.Labels[i].DeviceID < config.Labels[j].DeviceID
		}
		return config.Labels[i].EndpointID < config.Labels[j].EndpointID
	})

	manager.log.Debugf("saving endpoint labels...")
	data := &bytes.Buffer{}
	encoder := json.NewEncoder(data)
	encoder.SetIndent("", "\t")
	err := encoder.Encode(config)
	if err == nil {
		err = manager.store.Set(endpointLabelsKey, data.Bytes())
	}
	if err != nil {
		manager.mu.Lock()
		manager.labelsChanged = true
		manager.mu.Unlock()
	}
	return err
}
