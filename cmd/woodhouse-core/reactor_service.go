package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jimjibone/queue/v2"
	api "github.com/jimjibone/woodhouse-4/api/go"
)

type ReactorService struct {
	api.UnimplementedReactorServiceServer
	ds       *DeviceStore
	requests *queue.Pub[*api.DeviceRequest]
}

func NewReactorService(ds *DeviceStore) *ReactorService {
	return &ReactorService{
		ds:       ds,
		requests: queue.NewPub[*api.DeviceRequest](),
	}
}

func (rs *ReactorService) GetBridgeInfos(in *api.GetBridgeInfosRequest, server api.ReactorService_GetBridgeInfosServer) error {
	log.Printf("GetBridgeInfos started")
	defer log.Printf("GetBridgeInfos finished")

	sub := rs.ds.bridgesPub.NewSub()
	defer sub.Close()

	for _, item := range rs.ds.GetBridgeInfos() {
		err := server.Send(item)
		if err != nil {
			log.Printf("ERROR: GetBridgeInfos during send: %s", err)
			return err
		}
	}

	err := server.Send(&api.BridgeInfo{})
	if err != nil {
		log.Printf("ERROR: GetBridgeInfos during send: %s", err)
		return err
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-server.Context().Done():
			return nil

		case <-ticker.C:
			err := server.Send(&api.BridgeInfo{})
			if err != nil {
				log.Printf("ERROR: GetBridgeInfos during send: %s", err)
				return err
			}

		case item := <-sub.Sub():
			err := server.Send(item)
			if err != nil {
				log.Printf("ERROR: GetBridgeInfos during send: %s", err)
				return err
			}
		}
	}
}

func (rs *ReactorService) GetDeviceInfos(in *api.GetDeviceInfosRequest, server api.ReactorService_GetDeviceInfosServer) error {
	log.Printf("GetDeviceInfos started")
	defer log.Printf("GetDeviceInfos finished")

	sub := rs.ds.infosPub.NewSub()
	defer sub.Close()

	for _, item := range rs.ds.GetDeviceExtendedInfos() {
		err := server.Send(item)
		if err != nil {
			log.Printf("ERROR: GetDeviceInfos during send: %s", err)
			return err
		}
	}

	err := server.Send(&api.DeviceExtendedInfo{})
	if err != nil {
		log.Printf("ERROR: GetDeviceInfos during send: %s", err)
		return err
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-server.Context().Done():
			return nil

		case <-ticker.C:
			err := server.Send(&api.DeviceExtendedInfo{})
			if err != nil {
				log.Printf("ERROR: GetDeviceInfos during send: %s", err)
				return err
			}

		case item := <-sub.Sub():
			err := server.Send(item)
			if err != nil {
				log.Printf("ERROR: GetDeviceInfos during send: %s", err)
				return err
			}
		}
	}
}

func (rs *ReactorService) GetDeviceStates(in *api.GetDeviceStatesRequest, server api.ReactorService_GetDeviceStatesServer) error {
	log.Printf("GetDeviceStates started")
	defer log.Printf("GetDeviceStates finished")

	sub := rs.ds.statesPub.NewSub()
	defer sub.Close()

	for _, item := range rs.ds.GetDeviceStates() {
		err := server.Send(item)
		if err != nil {
			log.Printf("ERROR: GetDeviceStates during send: %s", err)
			return err
		}
	}

	err := server.Send(&api.DeviceState{})
	if err != nil {
		log.Printf("ERROR: GetDeviceStates during send: %s", err)
		return err
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-server.Context().Done():
			return nil

		case <-ticker.C:
			err := server.Send(&api.DeviceState{})
			if err != nil {
				log.Printf("ERROR: GetDeviceStates during send: %s", err)
				return err
			}

		case item := <-sub.Sub():
			err := server.Send(item)
			if err != nil {
				log.Printf("ERROR: GetDeviceStates during send: %s", err)
				return err
			}
		}
	}
}

func (rs *ReactorService) SetDeviceHidden(ctx context.Context, in *api.SetDeviceHiddenRequest) (*api.SetDeviceHiddenResponse, error) {
	if in.BridgeId == "" {
		return nil, fmt.Errorf("bridge_id must be set")
	}
	if in.DeviceId == "" {
		return nil, fmt.Errorf("device_id must be set")
	}
	log.Printf("SetDeviceHidden %s", in)
	err := rs.ds.SetDeviceHidden(in.BridgeId, in.DeviceId, in.Hidden)
	if err != nil {
		return nil, err
	}
	return &api.SetDeviceHiddenResponse{}, nil
}

func (rs *ReactorService) SendDeviceRequest(ctx context.Context, in *api.DeviceRequest) (*api.DeviceResponse, error) {
	if in.BridgeId == "" {
		return nil, fmt.Errorf("bridge_id must be set")
	}
	if in.DeviceId == "" {
		return nil, fmt.Errorf("device_id must be set")
	}
	log.Printf("SendDeviceRequest %s", in)
	rs.requests.Pub(in)
	return &api.DeviceResponse{}, nil
}
