package wh

import (
	"github.com/jimjibone/queue/v2"
	api "github.com/jimjibone/woodhouse-4/api/go"
	"google.golang.org/protobuf/proto"
)

type BridgeDevice struct {
	deviceInfos  *queue.Queue[*api.DeviceInfo]
	deviceStates *queue.Queue[*api.DeviceState]
}

func (bd BridgeDevice) SendInfo(info *api.DeviceInfo) {
	bd.deviceInfos.Push(proto.Clone(info).(*api.DeviceInfo))
}

func (bd BridgeDevice) SendState(state *api.DeviceState) {
	bd.deviceStates.Push(proto.Clone(state).(*api.DeviceState))
}

type Device interface {
	Init(bd *BridgeDevice)
	SendFullUpdate() // Send a full update of info and state. Typically done just after connecting to woodhouse or being added to the bridge.
	HandleRequest(*api.DeviceRequest) error
}
