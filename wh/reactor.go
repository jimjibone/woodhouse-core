package wh

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"google.golang.org/grpc"
)

var (
	ErrNotConnected = errors.New("not connected")
)

type Reactor struct {
	OnConnected    func()
	OnDisconnected func()
	devicesMu      sync.RWMutex
	devices        map[string]*ReactorDevice // devices map with device ID as the key
	clientMu       sync.RWMutex
	client         api.ReactorServiceClient
}

func NewReactor() *Reactor {
	r := &Reactor{
		devices: make(map[string]*ReactorDevice),
	}
	return r
}

func (r *Reactor) Device(deviceID string) *ReactorDevice {
	r.devicesMu.Lock()
	defer r.devicesMu.Unlock()
	if device, found := r.devices[deviceID]; found {
		return device
	}
	device := newReactorDevice(r, deviceID)
	r.devices[deviceID] = device
	return device
}

func (r *Reactor) Request(request *api.DeviceRequest) error {
	r.clientMu.RLock()
	defer r.clientMu.RUnlock()
	if r.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		response, err := r.client.SendDeviceRequest(ctx, request)
		if err != nil {
			return err
		}
		if response.ErrorMessage != "" {
			return errors.New(response.ErrorMessage)
		}
		return nil
	}
	return ErrNotConnected
}

func (r *Reactor) Run(appctx context.Context, conn *grpc.ClientConn) error {
	ctx, cancel := context.WithCancel(appctx)
	defer cancel()

	// Send the Reactor info.
	client := api.NewReactorServiceClient(conn)

	// Share the reactor client.
	r.clientMu.Lock()
	r.client = client
	r.clientMu.Unlock()

	defer func() {
		// Unshare the reactor client.
		r.clientMu.Lock()
		r.client = nil
		r.clientMu.Unlock()
	}()

	// Start receiving device infos.
	infos, err := client.GetDeviceInfos(ctx, &api.GetDeviceInfosRequest{})
	if err != nil {
		return fmt.Errorf("failed to get device infos: %w", err)
	}

	// Start receiving device states.
	states, err := client.GetDeviceStates(ctx, &api.GetDeviceStatesRequest{})
	if err != nil {
		return fmt.Errorf("failed to get device states: %w", err)
	}

	// Handle device infos.
	go func() {
		defer cancel()
		for {
			// Receive the next device info.
			info, err := infos.Recv()
			if err != nil {
				log.Printf("ERROR: failed to receive device info: %s", err)
				return
			}
			// log.Printf("received device info: %s", info)

			// Exit if the context is done.
			select {
			case <-ctx.Done():
				return
			default:
			}

			r.devicesMu.RLock()
			if device, found := r.devices[info.DeviceId]; found {
				device.handleInfo(info)
			}
			r.devicesMu.RUnlock()
		}
	}()

	// Handle device states.
	go func() {
		defer cancel()
		for {
			// Receive the next device state.
			state, err := states.Recv()
			if err != nil {
				log.Printf("ERROR: failed to receive device state: %s", err)
				return
			}
			// log.Printf("received device state: %s", state)

			// Exit if the context is done.
			select {
			case <-ctx.Done():
				return
			default:
			}

			r.devicesMu.RLock()
			if device, found := r.devices[state.DeviceId]; found {
				device.handleState(state)
			}
			r.devicesMu.RUnlock()
		}
	}()

	// Notify that the reactor is connected.
	if r.OnConnected != nil {
		r.OnConnected()
	}
	defer func() {
		// And also notify on disconnection.
		if r.OnDisconnected != nil {
			r.OnDisconnected()
		}
	}()

	// Wait for the context to be done.
	<-ctx.Done()
	return nil
}
