package core

import (
	"testing"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
)

// Last seen follows the newest attribute update, not only online.last_seen,
// and ignores the time the server stamps on the online service.
func TestFavoriteLastSeenFollowsAttributeUpdates(t *testing.T) {
	base := time.Unix(1_800_000_000, 0)
	attr := func(id string, v *clientsapi.Value, at time.Time) *clientsapi.Attribute {
		return &clientsapi.Attribute{Id: id, Value: v, Time: values.TimeToPb(at)}
	}
	device := func(fullState bool, services ...*clientsapi.Service) *clientsapi.Device {
		return &clientsapi.Device{Id: "dev", FullState: fullState, Endpoints: []*clientsapi.Endpoint{{Id: RootEndpointID, Services: services}}}
	}
	online := func(lastSeen, stamped time.Time) *clientsapi.Service {
		return &clientsapi.Service{Id: "online", Typ: clientsapi.Service_ONLINE, Attrs: []*clientsapi.Attribute{
			attr("online", values.Bool(true), stamped),
			attr("last_seen", values.Time(lastSeen), stamped),
		}}
	}
	onOff := func(at time.Time) *clientsapi.Service {
		return &clientsapi.Service{Id: "on_off", Typ: clientsapi.Service_ON_OFF, Attrs: []*clientsapi.Attribute{attr("on", values.Bool(true), at)}}
	}

	fav := &Favorite{DeviceID: "dev", EndpointID: RootEndpointID, ServiceID: "on_off"}
	fav.Update(device(true, online(base, base), onOff(base.Add(time.Minute))))
	if got := fav.LastSeen.Get(); !got.Equal(base.Add(time.Minute)) {
		t.Fatalf("last seen %s, want the on_off update time", got)
	}

	// A later attribute update moves it forward.
	if !fav.Update(device(false, onOff(base.Add(2*time.Minute)))) {
		t.Fatal("update should report a change")
	}
	if got := fav.LastSeen.Get(); !got.Equal(base.Add(2 * time.Minute)) {
		t.Fatalf("last seen %s, want the newer update time", got)
	}

	// The server's offline stamp on the online service does not count, and an
	// older last_seen never moves it back.
	fav.Update(device(false, online(base, base.Add(time.Hour))))
	if got := fav.LastSeen.Get(); !got.Equal(base.Add(2 * time.Minute)) {
		t.Fatalf("last seen %s, want it unchanged", got)
	}
}
