package users

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jimjibone/log"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"google.golang.org/grpc"
)

// fakeDevicesStream is the server side of a DevicesStream call.
type fakeDevicesStream struct {
	grpc.ServerStream
	ctx       context.Context
	responses chan *clientsapi.DevicesStreamResponse
}

func (s *fakeDevicesStream) Context() context.Context { return s.ctx }

func (s *fakeDevicesStream) Send(res *clientsapi.DevicesStreamResponse) error {
	select {
	case s.responses <- res:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *fakeDevicesStream) next(t *testing.T) *clientsapi.DevicesStreamResponse {
	t.Helper()
	select {
	case res := <-s.responses:
		return res
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for a stream response")
		return nil
	}
}

func buttonDevice(id string) *clientsapi.Device {
	return &clientsapi.Device{
		Id:        id,
		FullState: true,
		Typ:       clientsapi.Device_DEVICE,
		Endpoints: []*clientsapi.Endpoint{
			{
				Id: core.RootEndpointID,
				Services: []*clientsapi.Service{
					{Id: "online", Typ: clientsapi.Service_ONLINE},
				},
			},
			{
				Id: "1",
				Services: []*clientsapi.Service{{
					Id:     "button",
					Typ:    clientsapi.Service_BUTTON,
					Events: []*clientsapi.EventSpec{{Id: "press"}},
				}},
			},
		},
	}
}

func pressUpdate(id string) *clientsapi.Device {
	return &clientsapi.Device{Id: id, Events: []*clientsapi.Event{{EndpointId: "1", ServiceId: "button", Id: "press"}}}
}

func eventSeqs(res *clientsapi.DevicesStreamResponse) []uint64 {
	var out []uint64
	for _, ev := range res.GetDevice().GetEvents() {
		out = append(out, ev.GetSeq())
	}
	return out
}

// newDevicesStreamTest returns a device manager holding devices dev-a and dev-b, each
// with the given number of buffered events (dev-a's first), the epoch, and a
// function that starts a DevicesStream call.
func newDevicesStreamTest(t *testing.T, eventsA, eventsB int) (*core.DeviceManager, string, func(*clientsapi.DevicesStreamRequest) *fakeDevicesStream) {
	t.Helper()
	manager, err := core.NewDeviceManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("NewDeviceManager: %s", err)
	}
	t.Cleanup(manager.Close)
	userJwt, err := NewJWTManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("NewJWTManager: %s", err)
	}
	service := &UserService{
		log:           log.NewContext(log.DefaultLogger, "user-service", log.DebugLevel),
		deviceManager: manager,
		userJwt:       userJwt,
	}

	devices := manager.GetDeviceUpdates()
	defer devices.Close()
	events := manager.GetEvents()
	defer events.Close()
	manager.PushDeviceUpdate("client-1", buttonDevice("dev-a"))
	manager.PushDeviceUpdate("client-1", buttonDevice("dev-b"))
	for i := 0; i < 2; i++ {
		select {
		case <-devices.Sub():
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for devices")
		}
	}
	for i := 0; i < eventsA; i++ {
		manager.PushDeviceUpdate("client-1", pressUpdate("dev-a"))
	}
	for i := 0; i < eventsB; i++ {
		manager.PushDeviceUpdate("client-1", pressUpdate("dev-b"))
	}
	epoch := ""
	for i := 0; i < eventsA+eventsB; i++ {
		select {
		case ev := <-events.Sub():
			epoch = ev.Event.GetEpoch()
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for events")
		}
	}

	start := func(req *clientsapi.DevicesStreamRequest) *fakeDevicesStream {
		ctx, cancel := context.WithCancel(context.WithValue(context.Background(), "claims", &AccessTokenClaims{}))
		stream := &fakeDevicesStream{ctx: ctx, responses: make(chan *clientsapi.DevicesStreamResponse, 1024)}
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = service.DevicesStream(req, stream)
		}()
		t.Cleanup(func() {
			cancel()
			<-done
		})
		return stream
	}
	return manager, epoch, start
}

// Snapshots, then replayed events of every device in seq order, then live
// events.
func TestDevicesStreamReplaysThenGoesLive(t *testing.T) {
	manager, epoch, start := newDevicesStreamTest(t, 2, 1) // dev-a: 1, 2. dev-b: 3.
	stream := start(&clientsapi.DevicesStreamRequest{
		EventsSince: &clientsapi.EventCursor{Epoch: epoch, Seq: 1},
	})

	for i := 0; i < 2; i++ {
		if res := stream.next(t); len(res.GetDevice().GetEndpoints()) == 0 || len(res.GetDevice().GetEvents()) != 0 || res.GetEventsLost() {
			t.Errorf("snapshot %d = %v", i, res)
		}
	}
	if marker := stream.next(t); marker.GetDevice() != nil {
		t.Errorf("marker = %v", marker)
	}
	a := stream.next(t)
	b := stream.next(t)
	if a.GetDevice().GetId() != "dev-a" || !slices.Equal(eventSeqs(a), []uint64{2}) ||
		b.GetDevice().GetId() != "dev-b" || !slices.Equal(eventSeqs(b), []uint64{3}) {
		t.Errorf("replay = %v, %v", a, b)
	}
	if !a.GetDevice().GetEvents()[0].GetReplayed() || len(a.GetDevice().GetEndpoints()) != 0 {
		t.Errorf("replay = %v", a)
	}

	manager.PushDeviceUpdate("client-1", pressUpdate("dev-a"))
	live := stream.next(t)
	if !slices.Equal(eventSeqs(live), []uint64{4}) || live.GetDevice().GetEvents()[0].GetReplayed() {
		t.Errorf("live = %v", live)
	}
}

// The device filter applies to replayed and live events.
func TestDevicesStreamFiltersEvents(t *testing.T) {
	manager, _, start := newDevicesStreamTest(t, 1, 1)
	stream := start(&clientsapi.DevicesStreamRequest{
		IncludeDeviceIds: []string{"dev-b"},
		EventsSince:      &clientsapi.EventCursor{}, // The whole buffer.
	})

	if res := stream.next(t); res.GetDevice().GetId() != "dev-b" || res.GetEventsLost() {
		t.Errorf("snapshot = %v", res)
	}
	stream.next(t) // Marker.
	if res := stream.next(t); res.GetDevice().GetId() != "dev-b" || !slices.Equal(eventSeqs(res), []uint64{2}) {
		t.Errorf("replay = %v", res)
	}

	manager.PushDeviceUpdate("client-1", pressUpdate("dev-a"))
	manager.PushDeviceUpdate("client-1", pressUpdate("dev-b"))
	if res := stream.next(t); res.GetDevice().GetId() != "dev-b" || !slices.Equal(eventSeqs(res), []uint64{4}) {
		t.Errorf("live = %v", res)
	}
}

func TestDevicesStreamEventsLost(t *testing.T) {
	_, _, start := newDevicesStreamTest(t, 1, 0)
	stream := start(&clientsapi.DevicesStreamRequest{
		EventsSince: &clientsapi.EventCursor{Epoch: "old-epoch", Seq: 9},
	})
	if first := stream.next(t); !first.GetEventsLost() {
		t.Errorf("first = %v", first)
	}
	for {
		res := stream.next(t)
		if res.GetEventsLost() {
			t.Fatalf("events_lost repeated: %v", res)
		}
		if len(res.GetDevice().GetEvents()) > 0 {
			if !slices.Equal(eventSeqs(res), []uint64{1}) {
				t.Errorf("replay = %v", res)
			}
			break
		}
	}
}

func TestDevicesStreamWithoutCursorIsLiveOnly(t *testing.T) {
	manager, _, start := newDevicesStreamTest(t, 2, 0)
	stream := start(&clientsapi.DevicesStreamRequest{})
	stream.next(t)
	stream.next(t)
	stream.next(t) // Marker.

	manager.PushDeviceUpdate("client-1", pressUpdate("dev-a"))
	if res := stream.next(t); !slices.Equal(eventSeqs(res), []uint64{3}) {
		t.Errorf("live = %v", res)
	}
}
