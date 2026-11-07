package core

import (
	"testing"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
)

func timedAttr(id string, v *clientsapi.Value, at time.Time) *clientsapi.Attribute {
	return &clientsapi.Attribute{Id: id, Value: v, Time: values.TimeToPb(at)}
}

func lastSeenDevice(fullState bool, services ...*clientsapi.Service) *clientsapi.Device {
	return &clientsapi.Device{Id: "dev", FullState: fullState, Endpoints: []*clientsapi.Endpoint{{Id: RootEndpointID, Services: services}}}
}

func lastSeenOnline(lastSeen, stamped time.Time) *clientsapi.Service {
	return &clientsapi.Service{Id: "online", Typ: clientsapi.Service_ONLINE, Attrs: []*clientsapi.Attribute{
		timedAttr("online", values.Bool(true), stamped),
		timedAttr("last_seen", values.Time(lastSeen), stamped),
	}}
}

func lastSeenOnOff(at time.Time) *clientsapi.Service {
	return &clientsapi.Service{Id: "on_off", Typ: clientsapi.Service_ON_OFF, Attrs: []*clientsapi.Attribute{timedAttr("on", values.Bool(true), at)}}
}

// Last seen follows the newest attribute update and event, not only
// online.last_seen, and ignores the time the server stamps on the online
// service. It is sent when it moves, and never taken from a bridge.
func TestDeviceLastSeen(t *testing.T) {
	base := time.Unix(1_800_000_000, 0)
	wantSeen := func(t *testing.T, got *clientsapi.TimeValue, want time.Time) {
		t.Helper()
		if got == nil || !values.TimeFromPb(got).Equal(want) {
			t.Fatalf("last_seen %v, want %s", got, want)
		}
	}

	full := lastSeenDevice(true, lastSeenOnline(base, base), lastSeenOnOff(base.Add(time.Minute)))
	full.LastSeen = values.TimeToPb(base.Add(time.Hour)) // A bridge cannot set it.
	dev, err := newDevice(nil, "client-1", full)
	if err != nil {
		t.Fatal(err)
	}
	wantSeen(t, dev.pb().GetLastSeen(), base.Add(time.Minute))

	// A later attribute update moves it forward and is sent.
	changes, _ := dev.update(nil, "client-1", lastSeenDevice(false, lastSeenOnOff(base.Add(2*time.Minute))))
	wantSeen(t, changes.GetLastSeen(), base.Add(2*time.Minute))

	// The server's offline stamp does not count, an older last_seen never
	// moves it back, and an unchanged last seen is not sent.
	changes, _ = dev.update(nil, "client-1", lastSeenDevice(false, lastSeenOnline(base, base.Add(time.Hour))))
	if changes.GetLastSeen() != nil {
		t.Fatalf("last_seen sent unchanged: %v", changes.GetLastSeen())
	}
	wantSeen(t, dev.pb().GetLastSeen(), base.Add(2*time.Minute))

	// A newer online.last_seen counts.
	changes, _ = dev.update(nil, "client-1", lastSeenDevice(false, lastSeenOnline(base.Add(3*time.Minute), base)))
	wantSeen(t, changes.GetLastSeen(), base.Add(3*time.Minute))

	// So do events.
	if !dev.seenEvents([]*clientsapi.Event{{Id: "press", Time: values.TimeToPb(base.Add(4 * time.Minute))}}) {
		t.Fatal("event should move last seen")
	}
	wantSeen(t, dev.pb().GetLastSeen(), base.Add(4*time.Minute))
}

// An events-only update moves last seen and sends it.
func TestEventsMoveLastSeen(t *testing.T) {
	manager := newEventManager(t, 0, 0, nil)
	sub := manager.GetDeviceUpdates()
	defer sub.Close()

	at := time.Now().Add(time.Hour)
	event := pressEvent(1)
	event.Time = values.TimeToPb(at)
	pushUpdate(manager, eventsUpdate("dev-a", event))
	if got := recvUpdate(t, sub).Update.GetLastSeen(); got == nil || !values.TimeFromPb(got).Equal(at) {
		t.Fatalf("last_seen %v, want the event time", got)
	}
	if got := manager.GetDevice("dev-a").GetLastSeen(); got == nil || !values.TimeFromPb(got).Equal(at) {
		t.Fatalf("stored last_seen %v, want the event time", got)
	}
}

// Favourites take last seen from the device update and keep it when unset.
func TestFavoriteLastSeenFromDevice(t *testing.T) {
	base := time.Unix(1_800_000_000, 0)
	fav := &Favorite{DeviceID: "dev", EndpointID: RootEndpointID, ServiceID: "on_off"}
	if !fav.Update(&clientsapi.Device{Id: "dev", LastSeen: values.TimeToPb(base)}) {
		t.Fatal("update should report a change")
	}
	if fav.Update(&clientsapi.Device{Id: "dev"}) || !fav.LastSeen.Get().Equal(base) {
		t.Fatalf("unset last_seen changed it to %s", fav.LastSeen.Get())
	}
}
