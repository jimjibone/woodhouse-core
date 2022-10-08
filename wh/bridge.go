package wh

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/jimjibone/queue/v2"
	api "github.com/jimjibone/woodhouse-4/api/go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type Bridge struct {
	info           *api.BridgeInfo
	devicesMu      sync.RWMutex
	devices        map[string]Device // devices map with device ID as the key
	deviceInfos    *queue.Queue[*api.DeviceInfo]
	deviceStates   *queue.Queue[*api.DeviceState]
	OnConnected    func()
	OnDisconnected func()
}

func NewBridge(info *api.BridgeInfo) *Bridge {
	return &Bridge{
		info:         info,
		devices:      make(map[string]Device),
		deviceInfos:  queue.New[*api.DeviceInfo](),
		deviceStates: queue.New[*api.DeviceState](),
	}
}

func (b *Bridge) AddDevice(deviceID string, device Device) {
	b.devicesMu.Lock()
	defer b.devicesMu.Unlock()
	b.devices[deviceID] = device
	device.Init(&BridgeDevice{
		deviceInfos:  b.deviceInfos,
		deviceStates: b.deviceStates,
	})
	device.SendFullUpdate()
}

func (b *Bridge) Run(appctx context.Context, conn *grpc.ClientConn) error {
	ctx, cancel := context.WithCancel(appctx)
	defer cancel()

	// Send the bridge info.
	client := api.NewBridgeServiceClient(conn)
	_, err := client.SetBridgeInfo(appctx, proto.Clone(b.info).(*api.BridgeInfo))
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to set bridge info: %w", err)
	}

	// Start receiving requests.
	requests, err := client.GetDeviceRequests(ctx)
	if err != nil {
		log.Printf("ERROR: failed to get device requests: %s", err)
	}
	err = requests.Send(&api.DeviceResponse{
		BridgeId: b.info.BridgeId,
	})
	if err != nil {
		log.Printf("ERROR: failed to get device requests: %s", err)
	}

	go func() {
		defer cancel()
		for {
			// Exit if the context is done.
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Receive the next request.
			request, err := requests.Recv()
			if err != nil {
				log.Printf("ERROR: failed to receive request: %s", err)
				return
			}
			log.Printf("received request: %s", request)

			b.devicesMu.RLock()
			if device, found := b.devices[request.DeviceId]; found {
				err := device.HandleRequest(request)
				if err != nil {
					log.Printf("ERROR: device %s failed to handle request: %s", request.DeviceId, err)
					requests.Send(&api.DeviceResponse{
						BridgeId:     b.info.BridgeId,
						DeviceId:     request.DeviceId,
						RequestId:    request.RequestId,
						ErrorMessage: err.Error(),
					})
				} else {
					requests.Send(&api.DeviceResponse{
						BridgeId:  b.info.BridgeId,
						DeviceId:  request.DeviceId,
						RequestId: request.RequestId,
					})
				}
			} else {
				log.Printf("ERROR: device %s not found to handle request", request.DeviceId)
				requests.Send(&api.DeviceResponse{
					BridgeId:     b.info.BridgeId,
					DeviceId:     request.DeviceId,
					RequestId:    request.RequestId,
					ErrorMessage: "device not found",
				})
			}
			b.devicesMu.RUnlock()
		}
	}()

	// Flush queues before fill them with fresh items.
	b.deviceInfos.Flush()
	b.deviceStates.Flush()

	// Notify that the bridge is connected.
	if b.OnConnected != nil {
		b.OnConnected()
	}
	defer func() {
		// And also notify on disconnection.
		if b.OnDisconnected != nil {
			b.OnDisconnected()
		}
	}()

	// Send all devices to woodhouse core.
	b.devicesMu.RLock()
	for _, device := range b.devices {
		device.SendFullUpdate()
	}
	b.devicesMu.RUnlock()

	for {
		select {
		case <-ctx.Done():
			return nil

		case info := <-b.deviceInfos.Pop():
			info.BridgeId = b.info.BridgeId
			_, err := client.SetDeviceInfo(ctx, info)
			if err != nil {
				log.Printf("ERROR: failed to set device info: %s", err)
			}

		case state := <-b.deviceStates.Pop():
			state.BridgeId = b.info.BridgeId
			_, err := client.SetDeviceState(ctx, state)
			if err != nil {
				log.Printf("ERROR: failed to set device state: %s", err)
			}
		}
	}
}
