package wh

import (
	"fmt"
	"strings"
	"sync"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/apitools"
	"google.golang.org/protobuf/proto"
)

type ReactorDevice struct {
	reactor       *Reactor
	bridgeID      string
	deviceID      string
	name          string
	valuesMu      sync.RWMutex
	values        map[string]*api.DeviceValue
	handlersMu    sync.RWMutex
	stateHandlers []func(value []*api.DeviceValue)
	eventHandlers map[string]*eventHandler
}

type eventHandler struct {
	Handlers []func(value *api.DeviceValue)
}

func newReactorDevice(reactor *Reactor, deviceID string) *ReactorDevice {
	return &ReactorDevice{
		reactor:       reactor,
		deviceID:      deviceID,
		values:        make(map[string]*api.DeviceValue),
		eventHandlers: make(map[string]*eventHandler),
	}
}

func (rd *ReactorDevice) handleInfo(info *api.DeviceExtendedInfo) {
	rd.valuesMu.Lock()
	defer rd.valuesMu.Unlock()
	rd.bridgeID = info.BridgeId
	rd.name = info.Name
}

func (rd *ReactorDevice) handleState(state *api.DeviceState) {
	rd.valuesMu.Lock()
	defer rd.valuesMu.Unlock()

	// Update the device first.
	for _, value := range state.Values {
		name := strings.ToLower(value.Name)
		rd.values[name] = proto.Clone(value).(*api.DeviceValue)
	}

	// Call any state handlers.
	for _, handler := range rd.stateHandlers {
		handler(state.Values)
	}

	// Call any matching event handlers.
	for _, value := range state.Values {
		name := strings.ToLower(value.Name)
		if handlers, found := rd.eventHandlers[name]; found {
			for _, handler := range handlers.Handlers {
				handler(value)
			}
		}
	}
}

func (rd *ReactorDevice) originalValueName(name string) string {
	if value, found := rd.values[name]; found {
		return value.Name
	}
	return name
}

func (rd *ReactorDevice) DeviceID() string {
	return rd.deviceID
}

func (rd *ReactorDevice) Name() string {
	return rd.name
}

func (rd *ReactorDevice) OnState(handler func(values []*api.DeviceValue)) {
	rd.handlersMu.Lock()
	defer rd.handlersMu.Unlock()
	rd.stateHandlers = append(rd.stateHandlers, handler)
}

func (rd *ReactorDevice) OnEvent(name string, handler func(value *api.DeviceValue)) {
	rd.handlersMu.Lock()
	defer rd.handlersMu.Unlock()
	name = strings.ToLower(name)
	if handlers, found := rd.eventHandlers[name]; found {
		handlers.Handlers = append(handlers.Handlers, handler)
	} else {
		rd.eventHandlers[name] = &eventHandler{
			Handlers: []func(value *api.DeviceValue){handler},
		}
	}
}

// func (rd *ReactorDevice) ValueAs(name string, out interface{}) bool {
// 	rd.valuesMu.RLock()
// 	defer rd.valuesMu.RUnlock()
// 	name = strings.ToLower(name)
// 	if value, found := rd.values[name]; found {
// 		return apitools.ValueAs(value, out)
// 	}
// 	return false
// }

func (rd *ReactorDevice) ValueAs(name string, out interface{}) error {
	rd.valuesMu.RLock()
	defer rd.valuesMu.RUnlock()
	name = strings.ToLower(name)
	if value, found := rd.values[name]; found {
		if apitools.ValueAs(value, out) {
			return nil
		}
		return fmt.Errorf("failed conversion")
	}
	return fmt.Errorf("no value with name")
}

func (rd *ReactorDevice) RequestOne(name string, value interface{}) error {
	rd.valuesMu.RLock()
	defer rd.valuesMu.RUnlock()
	request := &api.DeviceRequest{
		BridgeId: rd.bridgeID,
		DeviceId: rd.deviceID,
		Values: []*api.DeviceValue{
			apitools.ValueFrom(rd.originalValueName(name), value),
		},
	}
	return rd.reactor.Request(request)
}

func (rd *ReactorDevice) Request(values ...*api.DeviceValue) error {
	rd.valuesMu.RLock()
	defer rd.valuesMu.RUnlock()
	request := &api.DeviceRequest{
		BridgeId: rd.bridgeID,
		DeviceId: rd.deviceID,
		Values:   values,
	}
	for _, value := range request.Values {
		value.Name = rd.originalValueName(value.Name)
	}
	return rd.reactor.Request(request)
}
