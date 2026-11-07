package core

import (
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jimjibone/queue/v2"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

// buttonDevice is a full state device whose endpoint "1" holds a button
// service declaring a "press" event with a required "count" field.
func buttonDevice(id string) *clientsapi.Device {
	dev := fullStateDevice(id, true)
	dev.Endpoints = append(dev.Endpoints, &clientsapi.Endpoint{
		Id: "1",
		Services: []*clientsapi.Service{{
			Id:  "button",
			Typ: clientsapi.Service_BUTTON,
			Events: []*clientsapi.EventSpec{{
				Id:     "press",
				Fields: []*clientsapi.FieldSpec{{Id: "count", Spec: intSpec(0, 100), Required: true}},
			}},
		}},
	})
	return dev
}

func pressEvent(count int64) *clientsapi.Event {
	return &clientsapi.Event{
		EndpointId: "1",
		ServiceId:  "button",
		Id:         "press",
		Fields:     []*clientsapi.FieldValue{values.Field("count", values.Int(count))},
	}
}

func eventsUpdate(id string, events ...*clientsapi.Event) *clientsapi.Device {
	return &clientsapi.Device{Id: id, Events: events}
}

// fakeClock is a settable clock.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// newEventManager returns a manager with a button device dev-a (and dev-b)
// already known, owned by client-1.
func newEventManager(t *testing.T, size int, age time.Duration, clock *fakeClock) *DeviceManager {
	t.Helper()
	manager, err := NewDeviceManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("NewDeviceManager: %s", err)
	}
	t.Cleanup(manager.Close)
	manager.mu.Lock()
	manager.eventBufferSize = size
	manager.eventBufferAge = age
	if clock != nil {
		manager.now = clock.Now
	}
	manager.mu.Unlock()
	pushUpdate(manager, buttonDevice("dev-a"))
	pushUpdate(manager, buttonDevice("dev-b"))
	return manager
}

// pushUpdate hands an update to the manager and handles it synchronously.
func pushUpdate(manager *DeviceManager, update *clientsapi.Device) {
	manager.handleDeviceUpdate(rxDeviceUpdate{ClientID: "client-1", Update: update})
}

func recvUpdate(t *testing.T, sub *queue.Sub[txDeviceUpdate]) txDeviceUpdate {
	t.Helper()
	select {
	case update := <-sub.Sub():
		return update
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for an update")
		return txDeviceUpdate{}
	}
}

func noUpdate(t *testing.T, sub *queue.Sub[txDeviceUpdate]) {
	t.Helper()
	select {
	case update := <-sub.Sub():
		t.Fatalf("unexpected update: %v", update)
	case <-time.After(50 * time.Millisecond):
	}
}

func seqs(events []DeviceEvent) []uint64 {
	var out []uint64
	for _, ev := range events {
		out = append(out, ev.Event.GetSeq())
	}
	return out
}

func wantSeqs(t *testing.T, events []DeviceEvent, want ...uint64) {
	t.Helper()
	if got := seqs(events); !slices.Equal(got, want) {
		t.Errorf("seqs = %v, want %v", got, want)
	}
}

// The manager sets epoch and seq, overwrites what the bridge set and defaults
// the time.
func TestEventsAreStamped(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1700000000, 0)}
	manager := newEventManager(t, 0, 0, clock)
	sub := manager.GetDeviceUpdates()
	defer sub.Close()
	live := manager.GetEvents()
	defer live.Close()

	bridgeTime := values.TimeToPb(time.Unix(1600000000, 5))
	forged := pressEvent(1)
	forged.Epoch = "bridge-epoch"
	forged.Seq = 99
	forged.Replayed = true
	forged.Time = bridgeTime
	pushUpdate(manager, eventsUpdate("dev-a", forged, pressEvent(2)))
	pushUpdate(manager, eventsUpdate("dev-b", pressEvent(3)))

	var got []*clientsapi.Event
	got = append(got, recvUpdate(t, sub).Update.GetEvents()...)
	got = append(got, recvUpdate(t, sub).Update.GetEvents()...)
	if len(got) != 3 {
		t.Fatalf("forwarded %d events, want 3", len(got))
	}
	if manager.epoch == "" {
		t.Fatalf("manager has no epoch")
	}
	for i, ev := range got {
		if ev.GetEpoch() != manager.epoch {
			t.Errorf("event %d epoch = %q, want %q", i, ev.GetEpoch(), manager.epoch)
		}
		if ev.GetSeq() != uint64(i+1) {
			t.Errorf("event %d seq = %d, want %d", i, ev.GetSeq(), i+1)
		}
		if ev.GetReplayed() {
			t.Errorf("event %d is flagged replayed", i)
		}
	}
	if got[0].GetTime().GetSeconds() != 1600000000 || got[0].GetTime().GetNanos() != 5 {
		t.Errorf("bridge time not kept: %v", got[0].GetTime())
	}
	if got[1].GetTime().GetSeconds() != 1700000000 {
		t.Errorf("unset time not defaulted to ingest time: %v", got[1].GetTime())
	}

	// The live subscription sees the same events, in order.
	for i := range got {
		select {
		case ev := <-live.Sub():
			if ev.Event.GetSeq() != uint64(i+1) {
				t.Errorf("live event %d seq = %d", i, ev.Event.GetSeq())
			}
			wantDev := "dev-a"
			if i == 2 {
				wantDev = "dev-b"
			}
			if ev.DeviceID != wantDev {
				t.Errorf("live event %d device = %q, want %q", i, ev.DeviceID, wantDev)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for live event %d", i)
		}
	}
}

// A new manager has another epoch and starts counting again.
func TestEventEpochChangesOnRestart(t *testing.T) {
	a := newEventManager(t, 0, 0, nil)
	b := newEventManager(t, 0, 0, nil)
	if a.epoch == b.epoch {
		t.Errorf("both managers have epoch %q", a.epoch)
	}
	pushUpdate(b, eventsUpdate("dev-a", pressEvent(1)))
	replay := b.EventsSince(nil, &clientsapi.EventCursor{})
	wantSeqs(t, replay.Events, 1)
}

func TestInvalidEventsAreDropped(t *testing.T) {
	manager := newEventManager(t, 0, 0, nil)
	sub := manager.GetDeviceUpdates()
	defer sub.Close()

	unknownEndpoint := pressEvent(1)
	unknownEndpoint.EndpointId = "9"
	unknownService := pressEvent(1)
	unknownService.ServiceId = "other"
	unknownEvent := pressEvent(1)
	unknownEvent.Id = "release"
	badValue := pressEvent(1)
	badValue.Fields = []*clientsapi.FieldValue{values.Field("count", values.Text("one"))}
	outOfRange := pressEvent(101)
	missingField := pressEvent(1)
	missingField.Fields = nil
	extraField := pressEvent(1)
	extraField.Fields = append(extraField.Fields, values.Field("extra", values.Int(1)))

	pushUpdate(manager, eventsUpdate("dev-a",
		unknownEndpoint, unknownService, unknownEvent, badValue, outOfRange, missingField, extraField,
		pressEvent(7)))
	update := recvUpdate(t, sub)
	if len(update.Update.GetEvents()) != 1 {
		t.Fatalf("forwarded %d events, want 1: %v", len(update.Update.GetEvents()), update.Update.GetEvents())
	}
	if ev := update.Update.GetEvents()[0]; ev.GetSeq() != 1 || ev.GetFields()[0].GetValue().GetInt() != 7 {
		t.Errorf("forwarded event = %v", ev)
	}
	wantSeqs(t, manager.EventsSince(nil, &clientsapi.EventCursor{}).Events, 1)

	// Only invalid events: nothing is forwarded.
	pushUpdate(manager, eventsUpdate("dev-a", unknownEvent))
	noUpdate(t, sub)

	// An unknown device, or one owned by another client.
	pushUpdate(manager, eventsUpdate("dev-none", pressEvent(1)))
	manager.handleDeviceUpdate(rxDeviceUpdate{ClientID: "client-2", Update: eventsUpdate("dev-a", pressEvent(1))})
	noUpdate(t, sub)
	wantSeqs(t, manager.EventsSince(nil, &clientsapi.EventCursor{}).Events, 1)
}

// Event specs may arrive in the same update as the events.
func TestEventsValidatedAgainstSpecsInSameUpdate(t *testing.T) {
	manager := newEventManager(t, 0, 0, nil)
	sub := manager.GetDeviceUpdates()
	defer sub.Close()

	update := buttonDevice("dev-c")
	update.Events = []*clientsapi.Event{pressEvent(4)}
	pushUpdate(manager, update)
	got := recvUpdate(t, sub).Update
	if !got.GetFullState() || len(got.GetEvents()) != 1 {
		t.Errorf("update = %v", got)
	}
}

// An update with only events is forwarded as a partial update without
// endpoints, and does not touch the state.
func TestEventsOnlyUpdateIsForwarded(t *testing.T) {
	manager := newEventManager(t, 0, 0, nil)
	sub := manager.GetDeviceUpdates()
	defer sub.Close()
	before := manager.GetDevice("dev-a")

	pushUpdate(manager, eventsUpdate("dev-a", pressEvent(1), pressEvent(2)))
	got := recvUpdate(t, sub).Update
	if got.GetId() != "dev-a" || got.GetClientId() != "client-1" {
		t.Errorf("update id/client = %q/%q", got.GetId(), got.GetClientId())
	}
	if got.GetFullState() || len(got.GetEndpoints()) != 0 {
		t.Errorf("update is not events only: %v", got)
	}
	if len(got.GetEvents()) != 2 || got.GetEvents()[0].GetSeq() != 1 || got.GetEvents()[1].GetSeq() != 2 {
		t.Errorf("events = %v", got.GetEvents())
	}
	after := manager.GetDevice("dev-a")
	before.LastSeen, after.LastSeen = nil, nil // Events move last seen.
	if len(after.GetEvents()) != 0 || after.String() != before.String() {
		t.Errorf("state changed: %v -> %v", before, after)
	}
}

// State and events in one update are forwarded together, so their order is
// kept.
func TestEventsForwardedWithState(t *testing.T) {
	manager := newEventManager(t, 0, 0, nil)
	sub := manager.GetDeviceUpdates()
	defer sub.Close()

	update := &clientsapi.Device{
		Id: "dev-a",
		Endpoints: []*clientsapi.Endpoint{{
			Id:       RootEndpointID,
			Services: []*clientsapi.Service{{Id: "online", Typ: clientsapi.Service_ONLINE, Attrs: []*clientsapi.Attribute{{Id: "online", Value: values.Bool(false)}}}},
		}},
		Events: []*clientsapi.Event{pressEvent(1)},
	}
	pushUpdate(manager, update)
	got := recvUpdate(t, sub).Update
	if len(got.GetEndpoints()) != 1 || len(got.GetEvents()) != 1 {
		t.Errorf("update = %v", got)
	}
	if snap := manager.GetDevice("dev-a"); len(snap.GetEvents()) != 0 {
		t.Errorf("events in snapshot: %v", snap.GetEvents())
	}
}

func TestEventsNotInSnapshots(t *testing.T) {
	manager := newEventManager(t, 0, 0, nil)
	pushUpdate(manager, eventsUpdate("dev-a", pressEvent(1)))

	if got := manager.GetDevice("dev-a"); len(got.GetEvents()) != 0 {
		t.Errorf("GetDevice has events: %v", got.GetEvents())
	}
	n := 0
	for dev := range manager.GetDevices() {
		n++
		if len(dev.GetEvents()) != 0 {
			t.Errorf("GetDevices has events: %v", dev.GetEvents())
		}
	}
	for dev := range manager.GetDevicesByIDs([]string{"dev-a"}) {
		if len(dev.GetEvents()) != 0 {
			t.Errorf("GetDevicesByIDs has events: %v", dev.GetEvents())
		}
	}
	if n == 0 {
		t.Fatalf("no devices")
	}

	manager.mu.Lock()
	manager.changed = true
	manager.mu.Unlock()
	if err := manager.save(); err != nil {
		t.Fatalf("save: %s", err)
	}
	data, err := manager.store.Get("devices.json")
	if err != nil {
		t.Fatalf("get devices.json: %s", err)
	}
	if strings.Contains(string(data), `"seq"`) || strings.Contains(string(data), manager.epoch) {
		t.Errorf("devices.json holds events: %s", data)
	}
}

func TestEventBufferLimitsByCount(t *testing.T) {
	manager := newEventManager(t, 3, time.Hour, nil)
	for i := int64(1); i <= 5; i++ {
		pushUpdate(manager, eventsUpdate("dev-a", pressEvent(i)))
	}
	pushUpdate(manager, eventsUpdate("dev-b", pressEvent(6)))

	replay := manager.EventsSince([]string{"dev-a"}, &clientsapi.EventCursor{})
	wantSeqs(t, replay.Events, 3, 4, 5)
	// dev-b is not held to dev-a's limit.
	wantSeqs(t, manager.EventsSince([]string{"dev-b"}, &clientsapi.EventCursor{}).Events, 6)
}

func TestEventBufferLimitsByAge(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1700000000, 0)}
	manager := newEventManager(t, 64, time.Minute, clock)

	pushUpdate(manager, eventsUpdate("dev-a", pressEvent(1)))
	clock.Advance(40 * time.Second)
	pushUpdate(manager, eventsUpdate("dev-a", pressEvent(2)))
	clock.Advance(40 * time.Second) // The first is 80s old, the second 40s.
	wantSeqs(t, manager.EventsSince(nil, &clientsapi.EventCursor{}).Events, 2)

	// Added events expire older ones too.
	clock.Advance(30 * time.Second)
	pushUpdate(manager, eventsUpdate("dev-a", pressEvent(3)))
	manager.mu.RLock()
	n := len(manager.eventBuffers["dev-a"].items)
	manager.mu.RUnlock()
	if n != 1 {
		t.Errorf("buffer holds %d events, want 1", n)
	}
}

func TestEventBufferDefaults(t *testing.T) {
	manager := &DeviceManager{}
	size, age := manager.eventBufferLimits()
	if size != 64 || age != 10*time.Minute {
		t.Errorf("defaults = %d, %s", size, age)
	}
}

func TestEventsSinceCursors(t *testing.T) {
	manager := newEventManager(t, 4, time.Hour, nil)
	for i := int64(1); i <= 3; i++ {
		pushUpdate(manager, eventsUpdate("dev-a", pressEvent(i)))
	}
	pushUpdate(manager, eventsUpdate("dev-b", pressEvent(4)))
	epoch := manager.epoch

	t.Run("nil cursor replays nothing", func(t *testing.T) {
		replay := manager.EventsSince(nil, nil)
		if len(replay.Events) != 0 || replay.Lost {
			t.Errorf("replay = %+v", replay)
		}
	})
	t.Run("current epoch replays newer events across devices in order", func(t *testing.T) {
		replay := manager.EventsSince(nil, &clientsapi.EventCursor{Epoch: epoch, Seq: 2})
		wantSeqs(t, replay.Events, 3, 4)
		if replay.Lost {
			t.Errorf("lost")
		}
		if replay.Last != 4 {
			t.Errorf("last = %d, want 4", replay.Last)
		}
		for _, ev := range replay.Events {
			if !ev.Event.GetReplayed() {
				t.Errorf("event %d not flagged replayed", ev.Event.GetSeq())
			}
		}
		if replay.Events[0].DeviceID != "dev-a" || replay.Events[1].DeviceID != "dev-b" {
			t.Errorf("devices = %q, %q", replay.Events[0].DeviceID, replay.Events[1].DeviceID)
		}
	})
	t.Run("current epoch at the last seq replays nothing", func(t *testing.T) {
		replay := manager.EventsSince(nil, &clientsapi.EventCursor{Epoch: epoch, Seq: 4})
		if len(replay.Events) != 0 || replay.Lost {
			t.Errorf("replay = %+v", replay)
		}
	})
	t.Run("device filter", func(t *testing.T) {
		replay := manager.EventsSince([]string{"dev-b", "dev-b", "dev-none"}, &clientsapi.EventCursor{Epoch: epoch})
		wantSeqs(t, replay.Events, 4)
		if replay.Lost {
			t.Errorf("lost")
		}
	})
	t.Run("empty epoch replays the whole buffer", func(t *testing.T) {
		replay := manager.EventsSince(nil, &clientsapi.EventCursor{})
		wantSeqs(t, replay.Events, 1, 2, 3, 4)
		if replay.Lost {
			t.Errorf("lost")
		}
	})
	t.Run("another epoch replays the whole buffer and is lost", func(t *testing.T) {
		replay := manager.EventsSince(nil, &clientsapi.EventCursor{Epoch: "old-epoch", Seq: 3})
		wantSeqs(t, replay.Events, 1, 2, 3, 4)
		if !replay.Lost {
			t.Errorf("not lost")
		}
	})
	t.Run("replays are copies", func(t *testing.T) {
		replay := manager.EventsSince(nil, &clientsapi.EventCursor{})
		replay.Events[0].Event.Fields = nil
		again := manager.EventsSince(nil, &clientsapi.EventCursor{})
		if len(again.Events[0].Event.GetFields()) == 0 {
			t.Errorf("buffer modified through a replay")
		}
	})

	// Overflow dev-a: seqs 5, 6, 7 push out 1 and 2.
	for i := int64(5); i <= 7; i++ {
		pushUpdate(manager, eventsUpdate("dev-a", pressEvent(i)))
	}
	t.Run("lost when evicted events were wanted", func(t *testing.T) {
		replay := manager.EventsSince(nil, &clientsapi.EventCursor{Epoch: epoch, Seq: 1})
		if !replay.Lost {
			t.Errorf("not lost")
		}
		wantSeqs(t, replay.Events, 3, 4, 5, 6, 7)
	})
	t.Run("not lost when the cursor is at the newest evicted event", func(t *testing.T) {
		replay := manager.EventsSince(nil, &clientsapi.EventCursor{Epoch: epoch, Seq: 2})
		if replay.Lost {
			t.Errorf("lost")
		}
		wantSeqs(t, replay.Events, 3, 4, 5, 6, 7)
	})
	t.Run("eviction in an unrequested device is not a loss", func(t *testing.T) {
		replay := manager.EventsSince([]string{"dev-b"}, &clientsapi.EventCursor{Epoch: epoch, Seq: 0})
		if replay.Lost {
			t.Errorf("lost")
		}
		wantSeqs(t, replay.Events, 4)
	})
	t.Run("empty epoch is never lost", func(t *testing.T) {
		replay := manager.EventsSince(nil, &clientsapi.EventCursor{})
		if replay.Lost {
			t.Errorf("lost")
		}
	})
}

func TestEventsSinceAfterAgeEviction(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1700000000, 0)}
	manager := newEventManager(t, 64, time.Minute, clock)
	pushUpdate(manager, eventsUpdate("dev-a", pressEvent(1)))
	clock.Advance(2 * time.Minute)
	pushUpdate(manager, eventsUpdate("dev-a", pressEvent(2)))

	replay := manager.EventsSince(nil, &clientsapi.EventCursor{Epoch: manager.epoch, Seq: 0})
	wantSeqs(t, replay.Events, 2)
	if !replay.Lost {
		t.Errorf("not lost")
	}
}

func TestRemovingDeviceDropsEventBuffer(t *testing.T) {
	manager := newEventManager(t, 0, 0, nil)
	pushUpdate(manager, eventsUpdate("dev-a", pressEvent(1)))
	pushUpdate(manager, eventsUpdate("dev-b", pressEvent(2)))

	manager.handleDeviceUpdate(rxDeviceUpdate{ClientID: "client-1", RemovedID: "dev-a"})
	wantSeqs(t, manager.EventsSince(nil, &clientsapi.EventCursor{}).Events, 2)

	if err := manager.RemoveDevice("dev-b", true); err != nil {
		t.Fatalf("RemoveDevice: %s", err)
	}
	wantSeqs(t, manager.EventsSince(nil, &clientsapi.EventCursor{}).Events)
}

func TestReplayUpdatesGroupsRuns(t *testing.T) {
	ev := func(seq uint64) *clientsapi.Event { return &clientsapi.Event{Seq: seq, Replayed: true} }
	updates := ReplayUpdates([]DeviceEvent{
		{DeviceID: "a", Event: ev(1)}, {DeviceID: "a", Event: ev(2)},
		{DeviceID: "b", Event: ev(3)}, {DeviceID: "a", Event: ev(4)},
	})
	if len(updates) != 3 {
		t.Fatalf("got %d updates, want 3", len(updates))
	}
	if updates[0].GetId() != "a" || len(updates[0].GetEvents()) != 2 || updates[1].GetId() != "b" || updates[2].GetId() != "a" {
		t.Errorf("updates = %v", updates)
	}
	for _, upd := range updates {
		if upd.GetFullState() || len(upd.GetEndpoints()) != 0 {
			t.Errorf("update carries state: %v", upd)
		}
	}
}

func TestTrimReplayedEvents(t *testing.T) {
	ev := func(seq uint64) *clientsapi.Event { return &clientsapi.Event{Seq: seq} }
	endpoints := []*clientsapi.Endpoint{{Id: "1"}}

	update := &clientsapi.Device{Id: "a", Events: []*clientsapi.Event{ev(3), ev(4)}}
	if got := TrimReplayedEvents(update, 0); got != update {
		t.Errorf("last 0 changed the update")
	}
	if got := TrimReplayedEvents(update, 2); got != update {
		t.Errorf("nothing to trim but the update was copied")
	}
	if got := TrimReplayedEvents(update, 4); got != nil {
		t.Errorf("fully replayed events-only update not dropped: %v", got)
	}
	got := TrimReplayedEvents(update, 3)
	if got == update || len(got.GetEvents()) != 1 || got.GetEvents()[0].GetSeq() != 4 {
		t.Errorf("trimmed = %v", got)
	}
	if len(update.GetEvents()) != 2 {
		t.Errorf("shared update modified: %v", update)
	}

	// State is kept when all events were replayed.
	withState := &clientsapi.Device{Id: "a", Endpoints: endpoints, Events: []*clientsapi.Event{ev(1)}}
	got = TrimReplayedEvents(withState, 5)
	if got == nil || len(got.GetEndpoints()) != 1 || len(got.GetEvents()) != 0 {
		t.Errorf("trimmed = %v", got)
	}
	full := &clientsapi.Device{Id: "a", FullState: true, Events: []*clientsapi.Event{ev(1)}}
	if got := TrimReplayedEvents(full, 5); got == nil {
		t.Errorf("full state update dropped")
	}
}

// Groups do not emit events: events in member updates are ignored.
func TestGroupIgnoresEvents(t *testing.T) {
	group := testGroup(&GroupMember{DeviceID: "dev-a", EndpointID: "1", ServiceID: "light"})
	ev := &clientsapi.Event{EndpointId: "1", ServiceId: "light", Id: "flash", Seq: 1}

	if got := group.Update(&clientsapi.Device{Id: "dev-a", Events: []*clientsapi.Event{ev}}); got != nil {
		t.Errorf("events-only update produced a group update: %v", got)
	}

	dev := lightDevice("dev-a", nil)
	dev.Events = []*clientsapi.Event{ev}
	got := group.Update(dev)
	if got == nil {
		t.Fatalf("no group update for a member state update")
	}
	if len(got.GetEvents()) != 0 {
		t.Errorf("group update has events: %v", got.GetEvents())
	}
}
