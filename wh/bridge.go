package wh

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jimjibone/queue/v2"
	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/discovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

const (
	minBackoff = time.Second
	maxBackoff = 30 * time.Second
)

type Bridge struct {
	info             *api.BridgeInfo
	disableDiscovery bool
	serverAddr       string
	lastBackoff      time.Time
	backoffDuration  time.Duration
	devicesMu        sync.RWMutex
	devices          map[string]Device // devices map with device ID as the key
	deviceInfos      *queue.Queue[*api.DeviceInfo]
	deviceStates     *queue.Queue[*api.DeviceState]
	OnConnected      func()
	OnDisconnected   func()
}

func NewBridge(info *api.BridgeInfo) *Bridge {
	return &Bridge{
		info:         info,
		devices:      make(map[string]Device),
		deviceInfos:  queue.New[*api.DeviceInfo](),
		deviceStates: queue.New[*api.DeviceState](),
	}
}

func (b *Bridge) SetServerAddr(addr string) {
	b.disableDiscovery = true
	b.serverAddr = addr
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

func (b *Bridge) Run(ctx context.Context) error {
	log.Printf("bridge started")
	defer log.Printf("bridge finished")

	for {
		found, err := b.discover(ctx)
		if err != nil {
			return err
		}
		if found {
			conn, err := b.connect(ctx)
			if err != nil {
				return err
			}
			if conn != nil {
				err = b.run(ctx, conn)
				conn.Close()
				if err != nil {
					return err
				}
			}
		}

		// Check if we're done.
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Backoff for a short while before the next connection attempt.
		b.backoff(ctx)
	}
}

func (b *Bridge) discover(ctx context.Context) (found bool, err error) {
	if !b.disableDiscovery {
		found = false

		log.Printf("starting discovery")

		// Start listening for woodhouse cores.
		listener := discovery.NewListener("woodhouse-core")
		if err := listener.Start(); err != nil {
			return false, fmt.Errorf("failed to start discovery: %w", err)
		}
		defer listener.Stop()

		done := false
		for !done {
			select {
			case <-ctx.Done():
				done = true
			case result := <-listener.Results():
				log.Printf("discovered instance: %s, hostname: %s, addr: %s", result.Instance, result.Hostname, result.Addr)
				done = true
				found = true
				b.serverAddr = result.Addr
			}
		}
	} else {
		found = true
		log.Printf("using predefined server address: %s", b.serverAddr)
	}
	return found, nil
}

func (b *Bridge) connect(ctx context.Context) (conn *grpc.ClientConn, err error) {
	// Connect and send our bridge info.
	log.Printf("connecting to: %s", b.serverAddr)
	// TODO: require valid certs
	// creds := credentials.NewTLS(&tls.Config{})
	creds := insecure.NewCredentials()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err = grpc.DialContext(
		ctx,
		b.serverAddr,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	client := api.NewBridgeServiceClient(conn)
	_, err = client.SetBridgeInfo(ctx, proto.Clone(b.info).(*api.BridgeInfo))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set bridge info: %w", err)
	}
	log.Printf("connection complete!")
	return conn, nil
}

func (b *Bridge) backoff(ctx context.Context) {
	// Reset the backoff duration if the backoff has not been used for a
	// suitable amount of time.
	dt := time.Since(b.lastBackoff)
	if dt > b.backoffDuration {
		log.Printf("backoff reset after %s", dt)
		b.backoffDuration = minBackoff
	}
	b.lastBackoff = time.Now()
	log.Printf("starting backoff for %s", b.backoffDuration)
	timer := time.NewTimer(b.backoffDuration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
	log.Printf("backoff finished")
	b.backoffDuration = b.backoffDuration * 2
}

func (b *Bridge) run(appctx context.Context, conn *grpc.ClientConn) error {
	log.Printf("connected!")

	ctx, cancel := context.WithCancel(appctx)
	defer cancel()

	client := api.NewBridgeServiceClient(conn)

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
