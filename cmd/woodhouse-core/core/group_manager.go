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

	"github.com/jimjibone/queue/v2"
	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"gopkg.in/yaml.v3"
)

var (
	GroupClientID         = "_group"
	ErrGroupNotFound      = errors.New("group not found")
	ErrGroupAlreadyExists = errors.New("group already exists")
)

// GroupManager enables grouping of services of the same type. Groups are
// defined by the user and are populated with data based on the current values
// from those chosen services. The group state is then pushed to the device
// manager as if it was a real device. The group manager also listens to action
// requests as that it can route requests to all downstream device services.
type GroupManager struct {
	log           *log.Context
	wg            sync.WaitGroup
	ctx           context.Context
	close         func()
	store         stores.Store
	deviceManager *DeviceManager
	publisher     *queue.Pub[GroupUpdate]
	listenerAdd   chan *queue.Sub[GroupUpdate]

	mu      sync.RWMutex
	changed bool
	groups  map[string]*Group
}

type GroupUpdate struct {
	Updated *Group
	Removed *string
}

func NewGroupManager(store stores.Store, deviceManager *DeviceManager) (*GroupManager, error) {
	ctx, close := context.WithCancel(context.Background())
	manager := &GroupManager{
		log:           log.NewContext(log.DefaultLogger, "group-manager", log.DebugLevel),
		ctx:           ctx,
		close:         close,
		store:         store,
		deviceManager: deviceManager,
		publisher:     queue.NewPub[GroupUpdate](),
		listenerAdd:   make(chan *queue.Sub[GroupUpdate], 1),
		groups:        make(map[string]*Group),
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

func (manager *GroupManager) Close() {
	manager.close()
	manager.wg.Wait()
}

func (manager *GroupManager) GetListener() *queue.Sub[GroupUpdate] {
	sub := manager.publisher.NewSub()
	manager.listenerAdd <- sub
	return sub
}

func (manager *GroupManager) verifyGroupMembers(members []*GroupMember) error {
	// Verify that the members exist - device id and service id.
	var deviceIDs []string
	for _, member := range members {
		deviceIDs = append(deviceIDs, member.DeviceID)
	}
	serviceType := clientsapi.Service_UNDEFINED
	for dev := range manager.deviceManager.GetDevicesByIDs(deviceIDs) {
		if dev == nil {
			return fmt.Errorf("device %q not found", dev.GetId())
		}
		for _, member := range members {
			if dev.GetId() == member.DeviceID {
				for _, srv := range dev.Services {
					if srv.GetId() == member.ServiceID {
						// Verify that all services are of the same type.
						if serviceType != clientsapi.Service_UNDEFINED && srv.Typ != serviceType {
							return fmt.Errorf("service %q on device %q is of type %q, expected type %q", member.ServiceID, member.DeviceID, srv.Typ, serviceType)
						}
						// Found the service, move to the next member.
						goto nextMember
					}
				}
				return fmt.Errorf("service %q not found on device %q", member.ServiceID, member.DeviceID)
			}
		}
	nextMember:
	}
	return nil
}

func (manager *GroupManager) forceGroupUpdate(grp *Group) {
	update := grp.initUpdate(true)
	grp.updateInfo(update)

	// Get all device states that make up the group.
	var deviceIDs []string
	for _, member := range grp.Members {
		deviceIDs = append(deviceIDs, member.DeviceID)
	}
	for dev := range manager.deviceManager.GetDevicesByIDs(deviceIDs) {
		if dev != nil {
			for _, srv := range dev.Services {
				grp.updateOnline(true, update, dev.Id, srv)
				grp.updateMembers(true, update, dev.Id, srv)
			}
		}
	}

	// Publish the update to the device manager, whatever it is.
	manager.deviceManager.PushDeviceUpdate(GroupClientID, update)
}

func (manager *GroupManager) AddGroup(grp *Group) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.groups[grp.GroupID] != nil {
		return ErrGroupAlreadyExists
	}

	// Verify that the members exist - device id and service id.
	err := manager.verifyGroupMembers(grp.Members)
	if err != nil {
		return err
	}

	// Add the group.
	manager.groups[grp.GroupID] = grp
	manager.changed = true

	// Publish the new/updated group to the listeners.
	manager.publisher.Pub(GroupUpdate{Updated: grp.ShallowClone()})

	// Populate the group with the current state of the devices.
	manager.forceGroupUpdate(grp)

	manager.log.Infof("group added %s", grp.String())

	return nil
}

func (manager *GroupManager) UpdateGroup(grp *Group) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.groups[grp.GroupID] = grp
	manager.changed = true

	// Publish the new/updated group to the listeners.
	manager.publisher.Pub(GroupUpdate{Updated: grp.ShallowClone()})

	// Populate the group with the current state of the devices.
	manager.forceGroupUpdate(grp)

	manager.log.Infof("group %q updated: %s", grp.GroupID, grp.String())

	return nil
}

func (manager *GroupManager) UpdateGroupName(groupID, name string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// Verify that the group exists.
	grp := manager.groups[groupID]
	if grp == nil {
		return ErrGroupNotFound
	}

	oldName := grp.Name
	grp.Name = name
	manager.changed = true

	manager.log.Infof("group %q updated name from %q to %q", grp.GroupID, oldName, name)

	// Publish the new/updated group to the listeners.
	manager.publisher.Pub(GroupUpdate{Updated: grp.ShallowClone()})

	update := grp.initUpdate(false)
	grp.updateInfo(update)
	if update != nil {
		manager.deviceManager.PushDeviceUpdate(GroupClientID, update)
	}

	return nil
}

func (manager *GroupManager) UpdateGroupMembers(groupID string, members []*GroupMember) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// Verify that the group exists.
	grp := manager.groups[groupID]
	if grp == nil {
		return ErrGroupNotFound
	}

	// Verify that the members exist - device id and service id.
	err := manager.verifyGroupMembers(members)
	if err != nil {
		return err
	}

	// Update the members.
	grp.Members = members
	manager.changed = true

	manager.log.Infof("group %q updated members", grp.GroupID)

	// Publish the new/updated group to the listeners.
	manager.publisher.Pub(GroupUpdate{Updated: grp.ShallowClone()})

	// Populate the group with the current state of the devices.
	manager.forceGroupUpdate(grp)

	return nil
}

func (manager *GroupManager) RemoveGroup(groupID string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.log.Infof("want to remove group %q", groupID)

	if _, found := manager.groups[groupID]; found {
		delete(manager.groups, groupID)
		manager.changed = true

		manager.log.Infof("group %q removed", groupID)

		// Publish the new/updated group to the listeners.
		manager.publisher.Pub(GroupUpdate{Removed: &groupID})
	}

	// Delete the group from the device manager.
	manager.deviceManager.RemoveDevice(groupID)

	return nil
}

func (manager *GroupManager) load() error {
	if manager.store.Has("groups.json") {
		manager.log.Debugf("loading...")

		// Load it.
		data, err := manager.store.Get("groups.json")
		if err != nil {
			return err
		}

		// Decode it.
		config := struct {
			Groups []*Group `json:"groups"`
		}{}
		err = json.NewDecoder(bytes.NewReader(data)).Decode(&config)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			if te, ok := err.(*yaml.TypeError); ok {
				fmt.Fprintln(os.Stderr, te.Errors)
			}
			return err
		}

		// Read the state into the manager (convert slice to map).
		manager.groups = make(map[string]*Group)
		for _, upd := range config.Groups {
			manager.groups[upd.GroupID] = upd
		}
	}
	return nil
}

func (manager *GroupManager) save() error {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	// Convert map to slice.
	config := struct {
		Groups []*Group `json:"groups"`
	}{}
	for _, grp := range manager.groups {
		config.Groups = append(config.Groups, grp)
	}
	// Sorted to maintain consistent structure between saves.
	sort.Slice(config.Groups, func(i, j int) bool {
		return config.Groups[i].GroupID < config.Groups[j].GroupID
	})

	// Encode it.
	data := &bytes.Buffer{}
	encoder := json.NewEncoder(data)
	encoder.SetIndent("", "\t")
	err := encoder.Encode(config)
	if err != nil {
		return err
	}

	// Save it.
	err = manager.store.Set("groups.json", data.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (manager *GroupManager) saveIfChanged() error {
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

func (manager *GroupManager) run(ctx context.Context) {
	defer manager.wg.Done()

	deviceUpdates := manager.deviceManager.GetDeviceUpdates()
	defer deviceUpdates.Close()

	actionRequests := manager.deviceManager.GetActionRequests()
	defer actionRequests.Close()

	// Send initial group info.
	manager.mu.RLock()
	for _, grp := range manager.groups {
		update := grp.initUpdate(false)
		grp.updateInfo(update)
		if update != nil {
			manager.deviceManager.PushDeviceUpdate(GroupClientID, update)
		}
	}
	manager.mu.RUnlock()

	// Func to loop over groups and update them with the device update.
	updateInGroups := func(dev *clientsapi.Device) {
		manager.mu.RLock()
		defer manager.mu.RUnlock()
		for _, srv := range dev.Services {
			for _, grp := range manager.groups {
				if grp.WantsServiceUpdate(dev.Id, srv) {
					update := grp.initUpdate(false)
					grp.updateOnline(false, update, dev.Id, srv)
					grp.updateMembers(false, update, dev.Id, srv)
					if update != nil {
						manager.deviceManager.PushDeviceUpdate(GroupClientID, update)
					}
				}
			}
		}
	}

	// Func to loop over groups and remove a device.
	removeFromGroups := func(deviceID string) {
		manager.mu.RLock()
		defer manager.mu.RUnlock()
		for _, grp := range manager.groups {
			if grp.WantsDevice(deviceID) {
				update := grp.initUpdate(false)
				grp.removeDevice(update, deviceID)
				if update != nil {
					manager.deviceManager.PushDeviceUpdate(GroupClientID, update)
				}
			}
		}
	}

	// Func to handle requests and send them to the matchine group.
	handleRequest := func(request ActionRequest) {
		manager.mu.RLock()
		defer manager.mu.RUnlock()
		if request.Request != nil {
			for _, grp := range manager.groups {
				if grp.GroupID == request.Request.GetDeviceId() {
					go grp.HandleRequest(request.Request, manager.deviceManager)
				}
			}
		}
	}

	// Build up group states from initial deviceManager state.
	for dev := range manager.deviceManager.GetDevices() {
		updateInGroups(dev)
	}

	manager.mu.RLock()
	manager.log.Debugf("initial groups are: %d", len(manager.groups))
	for _, grp := range manager.groups {
		manager.log.Debugf("  %s", grp)
	}
	manager.mu.RUnlock()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case update := <-deviceUpdates.Sub():
			if update.Update != nil {
				updateInGroups(update.Update)
			}

			// If a device is removed, we need to update the groups that contain it.
			if update.RemovedID != "" {
				removeFromGroups(update.RemovedID)
			}

		case request := <-actionRequests.Sub():
			if request.ClientID == GroupClientID {
				handleRequest(request)
			}

		case lis := <-manager.listenerAdd:
			// Publish all faves to the new listener.
			manager.mu.RLock()
			for _, grp := range manager.groups {
				manager.publisher.Send(lis, GroupUpdate{Updated: grp.ShallowClone()})
			}
			manager.mu.RUnlock()

			// Send an empty update to indicate the end of the initial list.
			manager.publisher.Send(lis, GroupUpdate{})

		case <-ticker.C:
			err := manager.saveIfChanged()
			if err != nil {
				manager.log.Fatalf("failed to save state: %s", err)
			}
		}
	}
}
