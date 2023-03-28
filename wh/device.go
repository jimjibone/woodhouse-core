package wh

import (
	"fmt"

	"github.com/jimjibone/queue/v2"
	api "github.com/jimjibone/woodhouse-4/api/go"
	"google.golang.org/protobuf/proto"
)

type BridgeComms struct {
	deviceInfos  *queue.Queue[*api.DeviceInfo]
	deviceStates *queue.Queue[*api.DeviceState]
}

func (bd *BridgeComms) SendInfo(info *api.DeviceInfo) error {
	if bd == nil {
		return fmt.Errorf("nil receiver")
	}
	if info == nil {
		return fmt.Errorf("nil info")
	}
	if info.DeviceId == "" {
		return fmt.Errorf("device id must be set")
	}
	bd.deviceInfos.Push(proto.Clone(info).(*api.DeviceInfo))
	return nil
}

func (bd *BridgeComms) SendState(state *api.DeviceState) error {
	if bd == nil {
		return fmt.Errorf("nil receiver")
	}
	if state == nil {
		return fmt.Errorf("nil state")
	}
	if state.DeviceId == "" {
		return fmt.Errorf("device id must be set")
	}
	for i, val := range state.Values {
		if val.Name == "" {
			return fmt.Errorf("value %d must have a name", i)
		}
	}
	bd.deviceStates.Push(proto.Clone(state).(*api.DeviceState))
	return nil
}

type Device interface {
	Init(comms *BridgeComms)
	SendFullUpdate() // Send a full update of info and state. Typically done just after connecting to woodhouse or being added to the bridge.
	HandleRequest(*api.DeviceRequest) error
}
