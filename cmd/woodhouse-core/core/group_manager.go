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

	"github.com/jimjibone/queue/v2"
	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"gopkg.in/yaml.v3"
)

var GroupClientID = "_group"

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

func (manager *GroupManager) AddGroup(grp *Group) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.groups[grp.GroupID] != nil {
		return ErrAlreadyExists
	}

	manager.groups[grp.GroupID] = grp.Clone()
	manager.changed = true

	manager.log.Infof("group %q added", grp.GroupID)

	// Publish the new/updated group to the listeners.
	manager.publisher.Pub(GroupUpdate{Updated: grp.Clone()})

	return nil
}

func (manager *GroupManager) UpdateGroup(grp *Group) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.groups[grp.GroupID] = grp.Clone()
	manager.changed = true

	manager.log.Infof("group %q updated", grp.GroupID)

	// Publish the new/updated group to the listeners.
	manager.publisher.Pub(GroupUpdate{Updated: grp.Clone()})

	return nil
}

func (manager *GroupManager) RemoveGroup(groupID string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if _, found := manager.groups[groupID]; !found {
		delete(manager.groups, groupID)
		manager.changed = true

		manager.log.Infof("group %q removed", groupID)

		// Publish the new/updated group to the listeners.
		manager.publisher.Pub(GroupUpdate{Removed: &groupID})
	}

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
		update := grp.UpdateInfo()
		if update != nil {
			manager.deviceManager.PushDeviceUpdate(GroupClientID, update)
		}
	}
	manager.mu.RUnlock()

	// Func to loop over groups and update them with the device update.
	updateGroups := func(dev *clientsapi.Device) {
		manager.mu.RLock()
		defer manager.mu.RUnlock()
		for _, srv := range dev.Services {
			for _, grp := range manager.groups {
				for _, member := range grp.Members {
					if dev.GetId() == member.DeviceID {
						if srv.Typ == clientsapi.Service_ONLINE {
							update := grp.UpdateOnline(manager.log, dev.GetId(), srv)
							if update != nil {
								manager.deviceManager.PushDeviceUpdate(GroupClientID, update)
							}
						}
						if srv.GetId() == member.ServiceID {
							update := grp.Update(manager.log, dev.GetId(), srv)
							if update != nil {
								manager.deviceManager.PushDeviceUpdate(GroupClientID, update)
							}
						}
					}
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
		updateGroups(dev)
	}

	manager.mu.RLock()
	manager.log.Debugf("initial groups are: %d", len(manager.groups))
	for _, grp := range manager.groups {
		manager.log.Debugf("  %s", grp)
	}
	manager.mu.RUnlock()

	for {
		select {
		case <-ctx.Done():
			return

		case update := <-deviceUpdates.Sub():
			updateGroups(update)

		case request := <-actionRequests.Sub():
			if request.ClientID == GroupClientID {
				handleRequest(request)
			}

		case lis := <-manager.listenerAdd:
			// Publish all faves to the new listener.
			manager.mu.RLock()
			for _, grp := range manager.groups {
				manager.publisher.Send(lis, GroupUpdate{Updated: grp.Clone()})
			}
			manager.mu.RUnlock()

			// Send an empty update to indicate the end of the initial list.
			manager.publisher.Send(lis, GroupUpdate{})
		}
	}
}
