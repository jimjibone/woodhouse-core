package wh

import (
	"log"

	"github.com/jimjibone/queue/v2"
	api "github.com/jimjibone/woodhouse-4/api/go"
	"google.golang.org/protobuf/proto"
)

type BridgeComms struct {
	deviceInfos  *queue.Queue[*api.DeviceInfo]
	deviceStates *queue.Queue[*api.DeviceState]
}

func (bd *BridgeComms) SendInfo(info *api.DeviceInfo) {
	if bd != nil {
		bd.deviceInfos.Push(proto.Clone(info).(*api.DeviceInfo))
	} else {
		log.Printf("ERROR: SendInfo called on nil BridgeComms")
	}
}

func (bd *BridgeComms) SendState(state *api.DeviceState) {
	if bd != nil {
		bd.deviceStates.Push(proto.Clone(state).(*api.DeviceState))
	} else {
		log.Printf("ERROR: SendState called on nil BridgeComms")
	}
}

type Device interface {
	Init(bridge *BridgeComms)
	SendFullUpdate() // Send a full update of info and state. Typically done just after connecting to woodhouse or being added to the bridge.
	HandleRequest(*api.DeviceRequest) error
}
