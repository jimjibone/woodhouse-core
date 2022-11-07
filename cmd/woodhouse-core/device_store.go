package main

import (
	"log"
	"sync"

	"github.com/jimjibone/queue/v2"
	api "github.com/jimjibone/woodhouse-4/api/go"
	"google.golang.org/protobuf/proto"
)

type DeviceStore struct {
	mu         sync.RWMutex
	bridges    map[string]*api.BridgeInfo
	infos      map[string]*api.DeviceInfo
	states     map[string]*api.DeviceState
	bridgesPub *queue.Pub[*api.BridgeInfo]
	infosPub   *queue.Pub[*api.DeviceInfo]
	statesPub  *queue.Pub[*api.DeviceState]
}

func NewDeviceStore() *DeviceStore {
	return &DeviceStore{
		bridges:    make(map[string]*api.BridgeInfo),
		infos:      make(map[string]*api.DeviceInfo),
		states:     make(map[string]*api.DeviceState),
		bridgesPub: queue.NewPub[*api.BridgeInfo](),
		infosPub:   queue.NewPub[*api.DeviceInfo](),
		statesPub:  queue.NewPub[*api.DeviceState](),
	}
}

func (ds *DeviceStore) SetBridgeInfo(in *api.BridgeInfo) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
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
	fullID := in.BridgeId + "." + in.DeviceId
	if prev, found := ds.infos[fullID]; found {
		log.Printf("SetDeviceInfo updated %s", in)
		prev.Name = in.Name
		prev.Description = in.Description
		prev.Url = in.Url
	} else {
		log.Printf("SetDeviceInfo added %s", in)
		ds.infos[fullID] = proto.Clone(in).(*api.DeviceInfo)
	}

	// Publish the new version.
	ds.infosPub.Pub(proto.Clone(ds.infos[fullID]).(*api.DeviceInfo))
	return nil
}

func (ds *DeviceStore) SetDeviceState(in *api.DeviceState) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	fullID := in.BridgeId + "." + in.DeviceId
	if prev, found := ds.states[fullID]; found {
		log.Printf("SetDeviceState updated %s", in)
		if in.FullUpdate {
			ds.states[fullID] = proto.Clone(in).(*api.DeviceState)
		} else {
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

func (ds *DeviceStore) GetDeviceInfos() []*api.DeviceInfo {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	var out []*api.DeviceInfo
	for _, info := range ds.infos {
		out = append(out, proto.Clone(info).(*api.DeviceInfo))
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
