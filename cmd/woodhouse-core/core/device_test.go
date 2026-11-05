package core

import (
	"slices"
	"testing"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
)

const (
	ro = clientsapi.Permissions_PERM_READONLY
	rw = clientsapi.Permissions_PERM_READWRITE
)

func attrIDs(srv *clientsapi.Service) []string {
	var ids []string
	for _, attr := range srv.GetAttrs() {
		ids = append(ids, attr.GetId())
	}
	return ids
}

func findEndpoint(dev *clientsapi.Device, id string) *clientsapi.Endpoint {
	for _, ep := range dev.GetEndpoints() {
		if ep.GetId() == id {
			return ep
		}
	}
	return nil
}

func findService(ep *clientsapi.Endpoint, id string) *clientsapi.Service {
	for _, srv := range ep.GetServices() {
		if srv.GetId() == id {
			return srv
		}
	}
	return nil
}

func findAttr(srv *clientsapi.Service, id string) *clientsapi.Attribute {
	for _, attr := range srv.GetAttrs() {
		if attr.GetId() == id {
			return attr
		}
	}
	return nil
}

func commandSpec(id string) *clientsapi.CommandSpec {
	return &clientsapi.CommandSpec{Id: id}
}

// fullDevice is a device with the root endpoint and two light endpoints.
func fullDevice() *clientsapi.Device {
	return &clientsapi.Device{
		Id:        "dev-1",
		FullState: true,
		Typ:       clientsapi.Device_DEVICE,
		Endpoints: []*clientsapi.Endpoint{
			{
				Id: "0",
				Services: []*clientsapi.Service{
					{Id: "info", Typ: clientsapi.Service_INFO, Attrs: []*clientsapi.Attribute{textAttr("name", "banana", rw)}},
					{Id: "online", Typ: clientsapi.Service_ONLINE, Attrs: []*clientsapi.Attribute{boolAttr("online", true, ro)}},
				},
			},
			{
				Id:    "2",
				Label: "Right",
				Services: []*clientsapi.Service{{
					Id:       "light",
					Typ:      clientsapi.Service_LIGHT,
					Alias:    "main",
					Features: []string{"brightness"},
					Commands: []*clientsapi.CommandSpec{commandSpec("toggle")},
					Events:   []*clientsapi.EventSpec{{Id: "changed"}},
					Attrs: []*clientsapi.Attribute{
						boolAttr("on", false, rw),
						intAttr("brightness", 10, 0, 100, rw),
						intAttr("color_temp", 3000, 2000, 6500, rw),
					},
				}},
			},
			{
				Id:    "1",
				Label: "Left",
				Services: []*clientsapi.Service{
					{Id: "light", Typ: clientsapi.Service_LIGHT, Attrs: []*clientsapi.Attribute{boolAttr("on", true, rw)}},
					{Id: "button", Typ: clientsapi.Service_BUTTON},
				},
			},
		},
	}
}

func newTestDevice(t *testing.T, update *clientsapi.Device) *Device {
	t.Helper()
	dev, err := newDevice(nil, "client-1", update)
	if err != nil {
		t.Fatalf("newDevice: %s", err)
	}
	return dev
}

func TestDevicePbSorted(t *testing.T) {
	dev := newTestDevice(t, fullDevice())
	pb := dev.pb()

	var epIDs []string
	for _, ep := range pb.Endpoints {
		epIDs = append(epIDs, ep.GetId())
	}
	if !slices.Equal(epIDs, []string{"0", "1", "2"}) {
		t.Errorf("endpoint order = %v", epIDs)
	}
	var srvIDs []string
	for _, srv := range findEndpoint(pb, "1").Services {
		srvIDs = append(srvIDs, srv.GetId())
	}
	if !slices.Equal(srvIDs, []string{"button", "light"}) {
		t.Errorf("service order = %v", srvIDs)
	}
	// Attributes keep the order the bridge reported them in.
	light := findService(findEndpoint(pb, "2"), "light")
	if !slices.Equal(attrIDs(light), []string{"on", "brightness", "color_temp"}) {
		t.Errorf("attribute order = %v", attrIDs(light))
	}
	if !pb.FullState || pb.ClientId != "client-1" || pb.Typ != clientsapi.Device_DEVICE {
		t.Errorf("pb = %v", pb)
	}

	// The snapshot is a copy.
	light.Attrs[0].Id = "changed"
	if got := dev.service("2", "light").Attrs[0].GetId(); got != "on" {
		t.Errorf("stored attribute changed through the snapshot: %q", got)
	}
}

func TestDevicePartialUpdateMerges(t *testing.T) {
	dev := newTestDevice(t, fullDevice())

	// Only the changed brightness, a new attribute and a changed alias. No
	// label, features, commands, events, specs or perms.
	changes, err := dev.update(nil, "client-1", &clientsapi.Device{
		Id:  "dev-1",
		Typ: clientsapi.Device_DEVICE,
		Endpoints: []*clientsapi.Endpoint{{
			Id: "2",
			Services: []*clientsapi.Service{{
				Id:    "light",
				Typ:   clientsapi.Service_LIGHT,
				Alias: "reading",
				Attrs: []*clientsapi.Attribute{
					{Id: "brightness", Value: values.Int(80)},
					{Id: "color", Value: values.Int(1), Spec: intSpec(0, 5), Perms: ro},
				},
			}},
		}},
	})
	if err != nil {
		t.Fatalf("update: %s", err)
	}

	srv := dev.service("2", "light")
	if got := dev.Endpoints["2"].Label; got != "Right" {
		t.Errorf("label = %q, want it kept", got)
	}
	if srv.Alias != "reading" {
		t.Errorf("alias = %q", srv.Alias)
	}
	if !slices.Equal(srv.Features, []string{"brightness"}) || len(srv.Commands) != 1 || len(srv.Events) != 1 {
		t.Errorf("features %v, commands %v, events %v: want kept", srv.Features, srv.Commands, srv.Events)
	}
	brightness := findAttr(srv, "brightness")
	if brightness.GetValue().GetInt() != 80 {
		t.Errorf("brightness = %v", brightness.GetValue())
	}
	if brightness.GetSpec().GetInt() == nil || brightness.GetSpec().GetInt().GetMax() != 100 {
		t.Errorf("brightness spec = %v, want kept", brightness.GetSpec())
	}
	if brightness.GetPerms() != rw {
		t.Errorf("brightness perms = %v, want kept", brightness.GetPerms())
	}
	if !slices.Equal(attrIDs(srv), []string{"on", "brightness", "color_temp", "color"}) {
		t.Errorf("attribute order = %v", attrIDs(srv))
	}
	if findEndpoint(dev.pb(), "1") == nil || dev.service("0", "info") == nil {
		t.Errorf("a partial update removed something")
	}

	// A new label is taken.
	_, err = dev.update(nil, "client-1", &clientsapi.Device{
		Id:        "dev-1",
		Endpoints: []*clientsapi.Endpoint{{Id: "2", Label: "Far right"}},
	})
	if err != nil {
		t.Fatalf("update: %s", err)
	}
	if got := dev.Endpoints["2"].Label; got != "Far right" {
		t.Errorf("label = %q", got)
	}

	// The forwarded changes hold the merged state of the services touched.
	if changes.FullState || len(changes.Endpoints) != 1 {
		t.Fatalf("changes = %v", changes)
	}
	ep := changes.Endpoints[0]
	if ep.GetId() != "2" || ep.GetLabel() != "Right" || len(ep.Services) != 1 {
		t.Fatalf("changed endpoint = %v", ep)
	}
	forwarded := ep.Services[0]
	if len(forwarded.Attrs) != 4 || len(forwarded.Features) != 1 || forwarded.Alias != "reading" {
		t.Errorf("forwarded service = %v", forwarded)
	}
	if findAttr(forwarded, "brightness").GetSpec() == nil {
		t.Errorf("forwarded brightness has no spec")
	}
}

func TestDeviceFullUpdateReplacesAndCollects(t *testing.T) {
	dev := newTestDevice(t, fullDevice())

	changes, err := dev.update(nil, "client-1", &clientsapi.Device{
		Id:        "dev-1",
		FullState: true,
		Typ:       clientsapi.Device_DEVICE,
		Endpoints: []*clientsapi.Endpoint{
			{
				Id: "0",
				Services: []*clientsapi.Service{
					{Id: "info", Typ: clientsapi.Service_INFO, Attrs: []*clientsapi.Attribute{textAttr("name", "banana", rw)}},
					{Id: "online", Typ: clientsapi.Service_ONLINE, Attrs: []*clientsapi.Attribute{boolAttr("online", true, ro)}},
				},
			},
			// Endpoint 1 is gone. Endpoint 2 lost its label, its alias, features,
			// commands, events and two attributes, and gained one.
			{
				Id: "2",
				Services: []*clientsapi.Service{{
					Id:  "light",
					Typ: clientsapi.Service_LIGHT,
					Attrs: []*clientsapi.Attribute{
						intAttr("brightness", 20, 0, 50, rw),
						boolAttr("on", true, rw),
						intAttr("saturation", 1, 0, 100, rw),
					},
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("update: %s", err)
	}

	pb := dev.pb()
	if findEndpoint(pb, "1") != nil {
		t.Errorf("endpoint 1 not removed")
	}
	ep := findEndpoint(pb, "2")
	if ep.GetLabel() != "" {
		t.Errorf("label = %q, want cleared", ep.GetLabel())
	}
	light := findService(ep, "light")
	if light.Alias != "" || len(light.Features) != 0 || len(light.Commands) != 0 || len(light.Events) != 0 {
		t.Errorf("light = %v, want description cleared", light)
	}
	// Kept attributes keep their place, new ones are appended, absent ones go.
	if !slices.Equal(attrIDs(light), []string{"on", "brightness", "saturation"}) {
		t.Errorf("attribute order = %v", attrIDs(light))
	}
	if got := findAttr(light, "brightness").GetSpec().GetInt().GetMax(); got != 50 {
		t.Errorf("brightness max = %d, want the new spec", got)
	}

	// Services are collected too.
	_, err = dev.update(nil, "client-1", &clientsapi.Device{
		Id:        "dev-1",
		FullState: true,
		Endpoints: []*clientsapi.Endpoint{
			{
				Id: "0",
				Services: []*clientsapi.Service{
					{Id: "info", Typ: clientsapi.Service_INFO, Attrs: []*clientsapi.Attribute{textAttr("name", "banana", rw)}},
				},
			},
			{Id: "2"},
		},
	})
	if err != nil {
		t.Fatalf("update: %s", err)
	}
	if dev.service("0", "online") != nil || dev.service("2", "light") != nil {
		t.Errorf("services not removed")
	}
	if dev.Endpoints["2"] == nil {
		t.Errorf("endpoint 2 with no services removed")
	}

	if !changes.FullState || len(changes.Endpoints) != 2 {
		t.Errorf("changes = %v", changes)
	}
}

func TestDeviceStampsIngestTime(t *testing.T) {
	observed := values.TimeToPb(time.Date(2026, 1, 2, 3, 4, 5, 6, time.UTC))

	update := fullDevice()
	light := update.Endpoints[1].Services[0]
	light.Attrs[1].Time = observed

	before := time.Now()
	dev := newTestDevice(t, update)
	after := time.Now()

	stored := dev.service("2", "light")
	if got := findAttr(stored, "brightness").GetTime(); got.GetSeconds() != observed.GetSeconds() || got.GetNanos() != observed.GetNanos() {
		t.Errorf("a time set by the bridge was replaced: %v", got)
	}
	got := values.TimeFromPb(findAttr(stored, "on").GetTime())
	if got.Before(before.Truncate(time.Second)) || got.After(after) || got.IsZero() {
		t.Errorf("ingest time = %s, want between %s and %s", got, before, after)
	}

	// Partial updates are stamped too, and so is what is forwarded.
	before = time.Now()
	changes, err := dev.update(nil, "client-1", &clientsapi.Device{
		Id: "dev-1",
		Endpoints: []*clientsapi.Endpoint{{
			Id: "2",
			Services: []*clientsapi.Service{{
				Id:    "light",
				Typ:   clientsapi.Service_LIGHT,
				Attrs: []*clientsapi.Attribute{{Id: "on", Value: values.Bool(true)}},
			}},
		}},
	})
	if err != nil {
		t.Fatalf("update: %s", err)
	}
	if got := values.TimeFromPb(findAttr(dev.service("2", "light"), "on").GetTime()); got.Before(before.Truncate(time.Second)) {
		t.Errorf("partial ingest time = %s", got)
	}
	fwd := findAttr(changes.Endpoints[0].Services[0], "on")
	if values.TimeFromPb(fwd.GetTime()).IsZero() {
		t.Errorf("forwarded attribute has no time")
	}
}

func TestDeviceDoesNotStoreEvents(t *testing.T) {
	update := fullDevice()
	update.Events = []*clientsapi.Event{{EndpointId: "2", ServiceId: "light", Id: "changed"}}
	dev := newTestDevice(t, update)

	if got := dev.pb(); len(got.Events) != 0 {
		t.Errorf("snapshot has events: %v", got.Events)
	}

	update = &clientsapi.Device{
		Id:     "dev-1",
		Events: []*clientsapi.Event{{EndpointId: "2", ServiceId: "light", Id: "changed"}},
	}
	changes, err := dev.update(nil, "client-1", update)
	if err != nil {
		t.Fatalf("update: %s", err)
	}
	if len(changes.Events) != 0 || len(dev.pb().Events) != 0 {
		t.Errorf("events kept: changes %v, snapshot %v", changes.Events, dev.pb().Events)
	}
}

func TestDeviceOnline(t *testing.T) {
	dev := newTestDevice(t, fullDevice())
	if !dev.isOnline() {
		t.Fatalf("device not online")
	}

	changes := dev.setOffline(nil)
	if dev.isOnline() {
		t.Errorf("device online after setOffline")
	}
	if changes == nil || changes.FullState || len(changes.Endpoints) != 1 {
		t.Fatalf("changes = %v", changes)
	}
	ep := changes.Endpoints[0]
	if ep.GetId() != RootEndpointID || len(ep.Services) != 1 || ep.Services[0].GetTyp() != clientsapi.Service_ONLINE {
		t.Errorf("changed endpoint = %v", ep)
	}
	if got := findAttr(ep.Services[0], "online"); got.GetValue().GetBool() || got.GetTime() == nil {
		t.Errorf("online attribute = %v", got)
	}

	// The online service of another endpoint does not count.
	update := fullDevice()
	update.Endpoints[0].Services = update.Endpoints[0].Services[:1]
	update.Endpoints[1].Services = append(update.Endpoints[1].Services,
		&clientsapi.Service{Id: "online", Typ: clientsapi.Service_ONLINE, Attrs: []*clientsapi.Attribute{boolAttr("online", true, ro)}})
	dev = newTestDevice(t, update)
	if dev.isOnline() {
		t.Errorf("device online without a root online service")
	}
	if dev.setOffline(nil) != nil {
		t.Errorf("setOffline returned changes without a root online service")
	}
}
