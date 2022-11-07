package main

import (
	"context"
	"log"

	"github.com/jimjibone/queue/v2"
	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/internal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BridgeService struct {
	api.UnimplementedBridgeServiceServer
	ds       *DeviceStore
	requests *queue.Pub[*api.DeviceRequest]
}

func NewBridgeService(ds *DeviceStore, rs *ReactorService) *BridgeService {
	return &BridgeService{
		ds:       ds,
		requests: rs.requests,
	}
}

func (b *BridgeService) SetBridgeInfo(ctx context.Context, in *api.BridgeInfo) (*api.SetBridgeInfoResponse, error) {
	err := b.ds.SetBridgeInfo(in)
	if err != nil {
		return nil, err
	}
	return &api.SetBridgeInfoResponse{}, nil
}

func (b *BridgeService) SetDeviceInfo(ctx context.Context, in *api.DeviceInfo) (*api.SetDeviceInfoResponse, error) {
	err := b.ds.SetDeviceInfo(in)
	if err != nil {
		return nil, err
	}
	return &api.SetDeviceInfoResponse{}, nil
}

func (b *BridgeService) SetDeviceState(ctx context.Context, in *api.DeviceState) (*api.SetDeviceStateResponse, error) {
	err := b.ds.SetDeviceState(in)
	if err != nil {
		return nil, err
	}
	return &api.SetDeviceStateResponse{}, nil
}

func (b *BridgeService) GetDeviceRequests(server api.BridgeService_GetDeviceRequestsServer) error {
	response, err := server.Recv()
	if err != nil {
		log.Printf("ERROR: GetDeviceRequests failed to receive first message: %s", err)
		return err
	}
	bridgeID := response.BridgeId
	if bridgeID == "" {
		return status.Errorf(codes.InvalidArgument, "bridge_id not set")
	}

	log.Printf("GetDeviceRequests started for %s", bridgeID)
	defer log.Printf("GetDeviceRequests finished for %s", bridgeID)

	requests := b.requests.NewSub()
	defer requests.Close()

	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	// Inject some fake requests.
	// go func() {
	// 	ticker := time.NewTicker(5 * time.Second)
	// 	defer ticker.Stop()
	// 	lastval := false
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return
	// 		case <-ticker.C:
	// 			lastval = !lastval
	// 			// b.requests.Pub(&api.DeviceRequest{
	// 			// 	BridgeId: bridgeID,
	// 			// 	DeviceId: "shellydimmer2-redacted",
	// 			// 	Values: []*api.DeviceValue{
	// 			// 		{
	// 			// 			Name: "On",
	// 			// 			Bool: &api.BoolValue{
	// 			// 				Value: lastval,
	// 			// 			},
	// 			// 		},
	// 			// 	},
	// 			// })
	// 			b.requests.Pub(&api.DeviceRequest{
	// 				BridgeId: bridgeID,
	// 				DeviceId: "zigbeeredacted",
	// 				Values: []*api.DeviceValue{
	// 					{
	// 						Name: "state",
	// 						Bool: &api.BoolValue{
	// 							Value: lastval,
	// 						},
	// 					},
	// 				},
	// 			})
	// 		}
	// 	}
	// }()

	go func() {
		defer cancel()
		for {
			// Exit if the context is done.
			select {
			case <-ctx.Done():
				return
			default:
			}

			response, err := server.Recv()
			if err != nil {
				log.Printf("ERROR: GetDeviceRequests failed to receive message: %s", err)
				return
			}
			log.Printf("GetDeviceRequests %s received response: %s", bridgeID, response)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil

		case request := <-requests.Sub():
			if request.RequestId == "" {
				requestID, err := internal.GenerateRandomString(6)
				if err != nil {
					log.Printf("ERROR: GetDeviceRequests failed to generate requestID: %s", err)
				}
				request.RequestId = requestID
			}

			log.Printf("GetDeviceRequests %s sending request: %s", bridgeID, request)
			if err := server.Send(request); err != nil {
				log.Printf("ERROR: GetDeviceRequests failed to receive send request: %s", err)
				return err
			}
		}
	}
}
