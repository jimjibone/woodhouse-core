package core

import (
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/jimjibone/queue/v2"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
	"google.golang.org/protobuf/proto"
)

const (
	// DefaultEventBufferSize is the most events kept per device.
	DefaultEventBufferSize = 64
	// DefaultEventBufferAge is how long an event is kept.
	DefaultEventBufferAge = 10 * time.Minute
)

// DeviceEvent is a stamped event and the device it belongs to. Receivers must
// not modify the event.
type DeviceEvent struct {
	DeviceID string
	Event    *clientsapi.Event
}

// EventReplay is the result of DeviceManager.EventsSince.
type EventReplay struct {
	// Events to replay, oldest first, flagged as replayed.
	Events []DeviceEvent
	// Lost is set when events since the cursor are no longer available.
	Lost bool
	// Last is the highest seq the manager had stamped when the replay was
	// taken. Live events with a seq up to this one are covered by the replay
	// (or were not wanted) and must not be sent again.
	Last uint64
}

// bufferedEvent is a stamped event and when it was ingested. The ingest time
// is used for expiry rather than the event time, which comes from the bridge.
type bufferedEvent struct {
	event *clientsapi.Event
	at    time.Time
}

// eventBuffer holds the recent events of one device, oldest first.
type eventBuffer struct {
	items   []bufferedEvent
	evicted uint64 // Highest seq dropped from the buffer, 0 if none.
}

// add appends an event and drops whatever no longer fits the limits.
func (buf *eventBuffer) add(event *clientsapi.Event, now time.Time, size int, maxAge time.Duration) {
	buf.items = append(buf.items, bufferedEvent{event: event, at: now})
	buf.prune(now, size, maxAge)
}

// prune drops events beyond size and events older than maxAge, recording the
// highest seq dropped.
func (buf *eventBuffer) prune(now time.Time, size int, maxAge time.Duration) {
	drop := 0
	for i, item := range buf.items {
		if len(buf.items)-i > size || now.Sub(item.at) > maxAge {
			drop = i + 1
		}
	}
	if drop == 0 {
		return
	}
	buf.evicted = max(buf.evicted, buf.items[drop-1].event.GetSeq())
	buf.items = slices.Delete(buf.items, 0, drop)
}

func (manager *DeviceManager) eventBufferLimits() (int, time.Duration) {
	size, maxAge := manager.eventBufferSize, manager.eventBufferAge
	if size <= 0 {
		size = DefaultEventBufferSize
	}
	if maxAge <= 0 {
		maxAge = DefaultEventBufferAge
	}
	return size, maxAge
}

func (manager *DeviceManager) eventNow() time.Time {
	if manager.now != nil {
		return manager.now()
	}
	return time.Now()
}

// GetEvents returns a subscription to stamped live events, in order. Replayed
// events are not published. The manager holds its lock while publishing, so
// events are never published out of order.
func (manager *DeviceManager) GetEvents() *queue.Sub[DeviceEvent] {
	return manager.events.NewSub()
}

// stampEvents validates and stamps the events of a bridge update and adds them
// to the device's buffer. Invalid events are dropped with a warning. Must be
// called with the lock held and after the update's state was applied, as the
// event specs may arrive in the same update.
func (manager *DeviceManager) stampEvents(dev *Device, events []*clientsapi.Event) []*clientsapi.Event {
	if len(events) == 0 {
		return nil
	}
	size, maxAge := manager.eventBufferLimits()
	now := manager.eventNow()

	var stamped []*clientsapi.Event
	for _, in := range events {
		if err := validateEvent(dev, in); err != nil {
			manager.log.Warnf("device %q dropped event: %s", dev.ID, err)
			continue
		}

		// The bridge's own epoch, seq and replayed are never trusted.
		event := proto.Clone(in).(*clientsapi.Event)
		manager.eventSeq++
		event.Epoch = manager.epoch
		event.Seq = manager.eventSeq
		event.Replayed = false
		if values.TimeFromPb(event.GetTime()).IsZero() {
			event.Time = values.TimeToPb(now)
		}

		buf := manager.eventBuffers[dev.ID]
		if buf == nil {
			if manager.eventBuffers == nil {
				manager.eventBuffers = make(map[string]*eventBuffer)
			}
			buf = &eventBuffer{}
			manager.eventBuffers[dev.ID] = buf
		}
		buf.add(event, now, size, maxAge)

		stamped = append(stamped, event)
		manager.events.Pub(DeviceEvent{DeviceID: dev.ID, Event: event})
	}
	return stamped
}

// validateEvent checks that the event names a service of the device that
// declares it, with fields matching the spec.
func validateEvent(dev *Device, event *clientsapi.Event) error {
	srv := dev.service(event.GetEndpointId(), event.GetServiceId())
	if srv == nil {
		return fmt.Errorf("event %q: unknown endpoint %q or service %q", event.GetId(), event.GetEndpointId(), event.GetServiceId())
	}
	i := slices.IndexFunc(srv.GetEvents(), func(spec *clientsapi.EventSpec) bool {
		return spec.GetId() == event.GetId()
	})
	if i < 0 {
		return fmt.Errorf("service %q does not declare event %q", srv.GetId(), event.GetId())
	}
	if err := values.ValidateFields(srv.Events[i].GetFields(), event.GetFields()); err != nil {
		return fmt.Errorf("event %q: %s", event.GetId(), err)
	}
	return nil
}

// removeEventBuffer drops the events of a removed device. Must be called with
// the lock held.
func (manager *DeviceManager) removeEventBuffer(deviceID string) {
	delete(manager.eventBuffers, deviceID)
}

// EventsSince returns the buffered events a stream resuming from cursor needs,
// as copies flagged replayed, oldest first. deviceIDs limits the devices, nil
// means all.
//
//   - A nil cursor replays nothing (live events only).
//   - The current epoch replays events with a higher seq. Lost is set when
//     events after the cursor were already dropped from a buffer.
//   - An empty epoch replays the whole buffer without Lost.
//   - Any other epoch (the manager restarted) replays the whole buffer and sets
//     Lost.
func (manager *DeviceManager) EventsSince(deviceIDs []string, cursor *clientsapi.EventCursor) EventReplay {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	replay := EventReplay{Last: manager.eventSeq}
	if cursor == nil {
		return replay
	}
	after := uint64(0)
	sameEpoch := false
	switch {
	case cursor.GetEpoch() == "":
	case cursor.GetEpoch() == manager.epoch:
		after = cursor.GetSeq()
		sameEpoch = true
	default:
		replay.Lost = true
	}

	size, maxAge := manager.eventBufferLimits()
	now := manager.eventNow()
	collect := func(deviceID string, buf *eventBuffer) {
		buf.prune(now, size, maxAge)
		if sameEpoch && buf.evicted > after {
			replay.Lost = true
		}
		for _, item := range buf.items {
			if item.event.GetSeq() <= after {
				continue
			}
			event := proto.Clone(item.event).(*clientsapi.Event)
			event.Replayed = true
			replay.Events = append(replay.Events, DeviceEvent{DeviceID: deviceID, Event: event})
		}
	}
	if deviceIDs == nil {
		for id, buf := range manager.eventBuffers {
			collect(id, buf)
		}
	} else {
		seen := make(map[string]bool, len(deviceIDs))
		for _, id := range deviceIDs {
			if buf := manager.eventBuffers[id]; buf != nil && !seen[id] {
				collect(id, buf)
			}
			seen[id] = true
		}
	}
	sort.Slice(replay.Events, func(i, j int) bool {
		return replay.Events[i].Event.GetSeq() < replay.Events[j].Event.GetSeq()
	})
	return replay
}

// ReplayUpdates turns replayed events into device updates carrying only
// events, one per run of consecutive events of the same device so order is
// kept.
func ReplayUpdates(events []DeviceEvent) []*clientsapi.Device {
	var updates []*clientsapi.Device
	for _, ev := range events {
		if n := len(updates); n > 0 && updates[n-1].GetId() == ev.DeviceID {
			updates[n-1].Events = append(updates[n-1].Events, ev.Event)
			continue
		}
		updates = append(updates, &clientsapi.Device{Id: ev.DeviceID, Events: []*clientsapi.Event{ev.Event}})
	}
	return updates
}

// TrimReplayedEvents removes the events with a seq up to last from a live
// update, as a stream already sent them from the replay. The update is shared
// between streams and is not modified; a copy is returned when events were
// removed. Returns nil when nothing is left to send.
func TrimReplayedEvents(update *clientsapi.Device, last uint64) *clientsapi.Device {
	if update == nil || last == 0 {
		return update
	}
	if !slices.ContainsFunc(update.GetEvents(), func(ev *clientsapi.Event) bool { return ev.GetSeq() <= last }) {
		return update
	}
	trimmed := proto.Clone(update).(*clientsapi.Device)
	trimmed.Events = slices.DeleteFunc(trimmed.Events, func(ev *clientsapi.Event) bool {
		return ev.GetSeq() <= last
	})
	if len(trimmed.Events) == 0 && len(trimmed.Endpoints) == 0 && !trimmed.FullState {
		return nil
	}
	return trimmed
}
