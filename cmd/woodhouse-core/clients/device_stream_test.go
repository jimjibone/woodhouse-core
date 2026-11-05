package clients

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jimjibone/log"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"google.golang.org/grpc"
)

// fakeDeviceStream is the server side of a DeviceStream call.
type fakeDeviceStream struct {
	grpc.ServerStream
	ctx       context.Context
	requests  chan *clientsapi.DeviceStreamRequest
	responses chan *clientsapi.DeviceStreamResponse
}

func (s *fakeDeviceStream) Context() context.Context { return s.ctx }

func (s *fakeDeviceStream) Send(res *clientsapi.DeviceStreamResponse) error {
	select {
	case s.responses <- res:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *fakeDeviceStream) Recv() (*clientsapi.DeviceStreamRequest, error) {
	select {
	case req := <-s.requests:
		return req, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

func startDeviceStream(t *testing.T, service *ClientService, first *clientsapi.DeviceStreamRequest) *fakeDeviceStream {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	stream := &fakeDeviceStream{
		ctx:       ctx,
		requests:  make(chan *clientsapi.DeviceStreamRequest, 8),
		responses: make(chan *clientsapi.DeviceStreamResponse, 1024),
	}
	if first != nil {
		stream.requests <- first
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = service.DeviceStream(stream)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return stream
}

func (s *fakeDeviceStream) next(t *testing.T) *clientsapi.DeviceStreamResponse {
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
					{Id: "info", Typ: clientsapi.Service_INFO, Attrs: []*clientsapi.Attribute{
						{Id: "name", Perms: clientsapi.Permissions_PERM_READONLY, Spec: &clientsapi.Spec{Kind: &clientsapi.Spec_Text{Text: &clientsapi.TextSpec{}}}, Value: values.Text(id)},
					}},
					{Id: "online", Typ: clientsapi.Service_ONLINE, Attrs: []*clientsapi.Attribute{
						{Id: "online", Perms: clientsapi.Permissions_PERM_READONLY, Spec: &clientsapi.Spec{Kind: &clientsapi.Spec_Bool{Bool: &clientsapi.BoolSpec{}}}, Value: values.Bool(true)},
					}},
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

func newTestClientService(t *testing.T) (*ClientService, *core.DeviceManager) {
	t.Helper()
	manager, err := core.NewDeviceManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("NewDeviceManager: %s", err)
	}
	t.Cleanup(manager.Close)
	service := &ClientService{
		log:                 log.NewContext(log.DefaultLogger, "client-service", log.DebugLevel),
		deviceManager:       manager,
		firstRequestTimeout: 50 * time.Millisecond,
	}
	return service, manager
}

// pushEvents pushes n press events of device id and waits until all are
// handled, returning the epoch.
func pushEvents(t *testing.T, manager *core.DeviceManager, id string, n int) string {
	t.Helper()
	sub := manager.GetEvents()
	defer sub.Close()
	for i := 0; i < n; i++ {
		manager.PushDeviceUpdate("client-1", pressUpdate(id))
	}
	epoch := ""
	for i := 0; i < n; i++ {
		select {
		case ev := <-sub.Sub():
			epoch = ev.Event.GetEpoch()
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for event %d", i+1)
		}
	}
	return epoch
}

func pushDevice(t *testing.T, manager *core.DeviceManager, dev *clientsapi.Device) {
	t.Helper()
	sub := manager.GetDeviceUpdates()
	defer sub.Close()
	manager.PushDeviceUpdate("client-1", dev)
	select {
	case <-sub.Sub():
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for device %q", dev.GetId())
	}
}

func eventSeqs(res *clientsapi.DeviceStreamResponse) []uint64 {
	var out []uint64
	for _, ev := range res.GetDevice().GetEvents() {
		out = append(out, ev.GetSeq())
	}
	return out
}

// Snapshots come first, then the replayed events, then live ones.
func TestDeviceStreamReplaysThenGoesLive(t *testing.T) {
	service, manager := newTestClientService(t)
	pushDevice(t, manager, buttonDevice("dev-a"))
	epoch := pushEvents(t, manager, "dev-a", 3)

	stream := startDeviceStream(t, service, &clientsapi.DeviceStreamRequest{
		EventsSince: &clientsapi.EventCursor{Epoch: epoch, Seq: 1},
	})

	snapshot := stream.next(t)
	if snapshot.GetDevice().GetId() != "dev-a" || len(snapshot.GetDevice().GetEndpoints()) == 0 || len(snapshot.GetDevice().GetEvents()) != 0 {
		t.Errorf("snapshot = %v", snapshot)
	}
	if snapshot.GetEventsLost() {
		t.Errorf("events lost")
	}
	if marker := stream.next(t); marker.GetDevice() != nil || marker.GetEventsLost() {
		t.Errorf("marker = %v", marker)
	}
	replayed := stream.next(t)
	if got := eventSeqs(replayed); !slices.Equal(got, []uint64{2, 3}) {
		t.Fatalf("replayed seqs = %v, want [2 3]", got)
	}
	if len(replayed.GetDevice().GetEndpoints()) != 0 || replayed.GetDevice().GetFullState() {
		t.Errorf("replay carries state: %v", replayed)
	}
	for _, ev := range replayed.GetDevice().GetEvents() {
		if !ev.GetReplayed() {
			t.Errorf("event %d not flagged replayed", ev.GetSeq())
		}
	}

	manager.PushDeviceUpdate("client-1", pressUpdate("dev-a"))
	live := stream.next(t)
	if got := eventSeqs(live); !slices.Equal(got, []uint64{4}) || live.GetDevice().GetEvents()[0].GetReplayed() {
		t.Errorf("live = %v", live)
	}
}

func TestDeviceStreamEventsLostOnFirstResponse(t *testing.T) {
	service, manager := newTestClientService(t)
	pushDevice(t, manager, buttonDevice("dev-a"))
	pushEvents(t, manager, "dev-a", 2)

	stream := startDeviceStream(t, service, &clientsapi.DeviceStreamRequest{
		EventsSince: &clientsapi.EventCursor{Epoch: "old-epoch", Seq: 7},
	})
	first := stream.next(t)
	if !first.GetEventsLost() || first.GetDevice().GetId() != "dev-a" {
		t.Errorf("first = %v", first)
	}
	if marker := stream.next(t); marker.GetEventsLost() {
		t.Errorf("events_lost repeated: %v", marker)
	}
	if got := eventSeqs(stream.next(t)); !slices.Equal(got, []uint64{1, 2}) {
		t.Errorf("replayed seqs = %v, want [1 2]", got)
	}
}

// With no devices the marker is the first response and carries events_lost.
func TestDeviceStreamEventsLostWithoutDevices(t *testing.T) {
	service, _ := newTestClientService(t)
	stream := startDeviceStream(t, service, &clientsapi.DeviceStreamRequest{
		EventsSince: &clientsapi.EventCursor{Epoch: "old-epoch"},
	})
	if first := stream.next(t); !first.GetEventsLost() {
		t.Errorf("first = %v", first)
	}
}

func TestDeviceStreamWithoutCursorIsLiveOnly(t *testing.T) {
	service, manager := newTestClientService(t)
	pushDevice(t, manager, buttonDevice("dev-a"))
	pushEvents(t, manager, "dev-a", 2)

	stream := startDeviceStream(t, service, &clientsapi.DeviceStreamRequest{})
	stream.next(t) // Snapshot.
	stream.next(t) // Marker.

	manager.PushDeviceUpdate("client-1", pressUpdate("dev-a"))
	live := stream.next(t)
	if got := eventSeqs(live); !slices.Equal(got, []uint64{3}) {
		t.Errorf("live seqs = %v, want [3]", got)
	}
}

// A client that sends no first request is served with no replay.
func TestDeviceStreamWithoutFirstRequest(t *testing.T) {
	service, manager := newTestClientService(t)
	pushDevice(t, manager, buttonDevice("dev-a"))
	stream := startDeviceStream(t, service, nil)
	stream.next(t) // Snapshot.
	stream.next(t) // Marker.
}

// The first request may also ask for a device state, which is sent after the
// marker.
func TestDeviceStreamGetDeviceRequests(t *testing.T) {
	service, manager := newTestClientService(t)
	pushDevice(t, manager, buttonDevice("dev-a"))
	stream := startDeviceStream(t, service, &clientsapi.DeviceStreamRequest{})
	stream.next(t) // Snapshot.
	stream.next(t) // Marker.

	stream.requests <- &clientsapi.DeviceStreamRequest{GetDeviceId: "dev-a"}
	if got := stream.next(t); got.GetDevice().GetId() != "dev-a" || len(got.GetDevice().GetEndpoints()) == 0 {
		t.Errorf("requested device = %v", got)
	}
}

// Events raised while the stream starts are sent exactly once, in order,
// whether from the replay or live.
func TestDeviceStreamNoDuplicatesOrGaps(t *testing.T) {
	service, manager := newTestClientService(t)
	pushDevice(t, manager, buttonDevice("dev-a"))
	epoch := pushEvents(t, manager, "dev-a", 1)

	const total = 50
	pushed := make(chan struct{})
	go func() {
		defer close(pushed)
		for i := 1; i < total; i++ {
			manager.PushDeviceUpdate("client-1", pressUpdate("dev-a"))
			if i%5 == 0 {
				time.Sleep(time.Millisecond)
			}
		}
	}()

	stream := startDeviceStream(t, service, &clientsapi.DeviceStreamRequest{
		EventsSince: &clientsapi.EventCursor{Epoch: epoch, Seq: 1},
	})

	var got []uint64
	for len(got) == 0 || got[len(got)-1] < total {
		res := stream.next(t)
		if res.GetEventsLost() {
			t.Fatalf("events lost")
		}
		got = append(got, eventSeqs(res)...)
	}
	<-pushed
	var want []uint64
	for seq := uint64(2); seq <= total; seq++ {
		want = append(want, seq)
	}
	if !slices.Equal(got, want) {
		t.Errorf("seqs = %v, want %v", got, want)
	}
}
