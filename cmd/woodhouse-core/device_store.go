package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/jimjibone/queue/v2"
	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/internal/jsonfile"
	"google.golang.org/protobuf/proto"
)

var (
	ErrDeviceNotFound = errors.New("device not found")
)

type DeviceStore struct {
	wg         sync.WaitGroup
	close      func()
	mu         sync.RWMutex
	changed    bool
	filename   string
	bridges    map[string]*api.BridgeInfo
	infos      map[string]*api.DeviceExtendedInfo
	states     map[string]*api.DeviceState
	bridgesPub *queue.Pub[*api.BridgeInfo]
	infosPub   *queue.Pub[*api.DeviceExtendedInfo]
	statesPub  *queue.Pub[*api.DeviceState]
}

func NewDeviceStore(storeFilename string) (*DeviceStore, error) {
	ctx, cancel := context.WithCancel(context.Background())
	ds := &DeviceStore{
		close:      cancel,
		filename:   storeFilename,
		bridges:    make(map[string]*api.BridgeInfo),
		infos:      make(map[string]*api.DeviceExtendedInfo),
		states:     make(map[string]*api.DeviceState),
		bridgesPub: queue.NewPub[*api.BridgeInfo](),
		infosPub:   queue.NewPub[*api.DeviceExtendedInfo](),
		statesPub:  queue.NewPub[*api.DeviceState](),
	}

	// Load the previous store from file.
	err := ds.loadStore(storeFilename)
	if err != nil {
		return nil, err
	}

	// Save an empty version of the store if the file doesn't exist.
	if _, err := os.Stat(storeFilename); errors.Is(err, fs.ErrNotExist) {
		err = ds.saveStore(storeFilename)
		if err != nil {
			return nil, err
		}
	}

	// Periodically save the store if it has changed.
	ds.wg.Add(1)
	go func() {
		defer ds.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ds.mu.RLock()
				if ds.changed {
					ds.changed = false
					err := ds.saveStore(storeFilename)
					if err != nil {
						log.Fatalf("failed to save device store: %s", err)
					}
				}
				ds.mu.RUnlock()
			}
		}
	}()

	return ds, nil
}

func (ds *DeviceStore) Close() error {
	ds.close()
	ds.wg.Wait()
	return ds.saveStore(ds.filename)
}

func (ds *DeviceStore) loadStore(filename string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	store := struct {
		Info  []*api.DeviceExtendedInfo `json:"info"`
		State []*api.DeviceState        `json:"state"`
	}{}

	err := jsonfile.LoadFile(&store, filename)
	if err != nil {
		return err
	}

	for _, info := range store.Info {
		fullID := info.BridgeId + "." + info.DeviceId
		ds.infos[fullID] = info
	}

	for _, state := range store.State {
		fullID := state.BridgeId + "." + state.DeviceId
		ds.states[fullID] = state
	}

	return nil
}

func (ds *DeviceStore) saveStore(filename string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	store := struct {
		Info  []*api.DeviceExtendedInfo `json:"info"`
		State []*api.DeviceState        `json:"state"`
	}{}

	for _, info := range ds.infos {
		store.Info = append(store.Info, info)
	}
	sort.Slice(store.Info, func(i, j int) bool {
		iid := store.Info[i].BridgeId + "." + store.Info[i].DeviceId
		jid := store.Info[j].BridgeId + "." + store.Info[j].DeviceId
		return iid < jid
	})

	for _, state := range ds.states {
		store.State = append(store.State, state)
	}
	sort.Slice(store.State, func(i, j int) bool {
		iid := store.State[i].BridgeId + "." + store.State[i].DeviceId
		jid := store.State[j].BridgeId + "." + store.State[j].DeviceId
		return iid < jid
	})

	return jsonfile.SaveFile(store, filename)
}

func (ds *DeviceStore) SetBridgeInfo(in *api.BridgeInfo) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.changed = true

	if prev, found := ds.bridges[in.BridgeId]; found {
		log.Printf("SetBridgeInfo updated %s", in)
		prev.Name = in.Name
		prev.Description = in.Description
		prev.BootTime = proto.Clone(in.BootTime).(*api.Timestamp)
	} else {
		log.Printf("SetBridgeInfo added %s", in)
		ds.bridges[in.BridgeId] = proto.Clone(in).(*api.BridgeInfo)
	}

	// Publish the new version.
	ds.bridgesPub.Pub(proto.Clone(ds.bridges[in.BridgeId]).(*api.BridgeInfo))
	return nil
}

func (ds *DeviceStore) SetDeviceInfo(in *api.DeviceInfo) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.changed = true

	fullID := in.BridgeId + "." + in.DeviceId
	if prev, found := ds.infos[fullID]; found {
		log.Printf("SetDeviceInfo updated %s", in)
		prev.Name = in.Name
		prev.Description = in.Description
		prev.Url = in.Url
	} else {
		log.Printf("SetDeviceInfo added %s", in)
		ds.infos[fullID] = &api.DeviceExtendedInfo{
			BridgeId:    in.BridgeId,
			DeviceId:    in.DeviceId,
			Name:        in.Name,
			Description: in.Description,
			Url:         in.Url,
			Hidden:      false,
		}
	}

	// Publish the new version.
	ds.infosPub.Pub(proto.Clone(ds.infos[fullID]).(*api.DeviceExtendedInfo))
	return nil
}

func (ds *DeviceStore) SetDeviceHidden(bridgeID, deviceID string, hidden bool) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.changed = true

	fullID := bridgeID + "." + deviceID
	if prev, found := ds.infos[fullID]; found {
		log.Printf("SetDeviceHidden %q hidden set to %t", fullID, hidden)
		prev.Hidden = hidden
	} else {
		return ErrDeviceNotFound
	}

	// Publish the new version.
	ds.infosPub.Pub(proto.Clone(ds.infos[fullID]).(*api.DeviceExtendedInfo))
	return nil
}

func (ds *DeviceStore) SetDeviceFavourite(bridgeID, deviceID string, favourite bool) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.changed = true

	fullID := bridgeID + "." + deviceID
	if prev, found := ds.infos[fullID]; found {
		log.Printf("SetDeviceFavourite %q favourite set to %t", fullID, favourite)
		prev.Favourite = favourite
	} else {
		return ErrDeviceNotFound
	}

	// Publish the new version.
	ds.infosPub.Pub(proto.Clone(ds.infos[fullID]).(*api.DeviceExtendedInfo))
	return nil
}

func (ds *DeviceStore) SetDeviceState(in *api.DeviceState) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.changed = true

	fullID := in.BridgeId + "." + in.DeviceId
	if prev, found := ds.states[fullID]; found {
		log.Printf("SetDeviceState updated %s", in)
		if in.FullUpdate {
			ds.states[fullID] = proto.Clone(in).(*api.DeviceState)
		} else {
			prev.Online = in.Online
			prev.LastSeen = proto.Clone(in.LastSeen).(*api.Timestamp)
			for _, next := range in.Values {
				found := false
				for i, val := range prev.Values {
					if val.Name == next.Name {
						found = true
						prev.Values[i] = proto.Clone(next).(*api.DeviceValue)
						break
					}
				}
				if !found {
					prev.Values = append(prev.Values, proto.Clone(next).(*api.DeviceValue))
				}
			}
		}
	} else {
		log.Printf("SetDeviceState added %s", in)
		ds.states[fullID] = proto.Clone(in).(*api.DeviceState)
	}

	// Publish the new version.
	ds.statesPub.Pub(proto.Clone(ds.states[fullID]).(*api.DeviceState))
	return nil
}

func (ds *DeviceStore) GetBridgeInfos() []*api.BridgeInfo {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	var out []*api.BridgeInfo
	for _, bridge := range ds.bridges {
		out = append(out, proto.Clone(bridge).(*api.BridgeInfo))
	}
	return out
}

func (ds *DeviceStore) GetDeviceExtendedInfos() []*api.DeviceExtendedInfo {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	var out []*api.DeviceExtendedInfo
	for _, info := range ds.infos {
		out = append(out, proto.Clone(info).(*api.DeviceExtendedInfo))
	}
	return out
}

func (ds *DeviceStore) GetDeviceStates() []*api.DeviceState {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	var out []*api.DeviceState
	for _, state := range ds.states {
		out = append(out, proto.Clone(state).(*api.DeviceState))
	}
	return out
}
