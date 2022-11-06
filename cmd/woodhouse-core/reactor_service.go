package main

import (
	"context"
	"log"

	"github.com/jimjibone/queue/v2"
	api "github.com/jimjibone/woodhouse-4/api/go"
)

type ReactorService struct {
	api.UnimplementedReactorServiceServer
	requests *queue.Pub[*api.DeviceRequest]
}

func NewReactorService() *ReactorService {
	return &ReactorService{
		requests: queue.NewPub[*api.DeviceRequest](),
	}
}

func (rs *ReactorService) GetDeviceInfos(in *api.GetDeviceInfosRequest, server api.ReactorService_GetDeviceInfosServer) error {
	log.Printf("GetDeviceInfos started")
	defer log.Printf("GetDeviceInfos finished")
	<-server.Context().Done()
	return nil
	// return status.Errorf(codes.Unimplemented, "method GetDeviceInfos not implemented")
}

func (rs *ReactorService) GetDeviceStates(in *api.GetDeviceStatesRequest, server api.ReactorService_GetDeviceStatesServer) error {
	log.Printf("GetDeviceStates started")
	defer log.Printf("GetDeviceStates finished")
	<-server.Context().Done()
	return nil
	// return status.Errorf(codes.Unimplemented, "method GetDeviceStates not implemented")
}

func (rs *ReactorService) SendDeviceRequest(ctx context.Context, in *api.DeviceRequest) (*api.DeviceResponse, error) {
	log.Printf("SendDeviceRequest %s", in)
	rs.requests.Pub(in)
	return &api.DeviceResponse{}, nil
}
