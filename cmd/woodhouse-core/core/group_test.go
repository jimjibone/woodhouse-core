package core

import (
	"slices"
	"testing"
	"time"

	"github.com/jimjibone/queue/v2"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
)

func commandSpecWithParam(id, param string) *clientsapi.CommandSpec {
	return &clientsapi.CommandSpec{
		Id:     id,
		Params: []*clientsapi.FieldSpec{{Id: param, Spec: intSpec(0, 100)}},
	}
}

// lightDevice is a device whose endpoint "1" holds a light service.
func lightDevice(id string, features []string, commands ...*clientsapi.CommandSpec) *clientsapi.Device {
	return &clientsapi.Device{
		Id:        id,
		FullState: true,
		Typ:       clientsapi.Device_DEVICE,
		Endpoints: []*clientsapi.Endpoint{
			{
				Id: "0",
				Services: []*clientsapi.Service{
					{Id: "online", Typ: clientsapi.Service_ONLINE, Attrs: []*clientsapi.Attribute{boolAttr("online", true, ro)}},
				},
			},
			{
				Id: "1",
				Services: []*clientsapi.Service{{
					Id:       "light",
					Typ:      clientsapi.Service_LIGHT,
					Features: features,
					Commands: commands,
					Attrs: []*clientsapi.Attribute{
						boolAttr("on", true, rw),
						intAttr("brightness", 10, 0, 100, rw),
					},
				}},
			},
		},
	}
}

func testGroup(members ...*GroupMember) *Group {
	return NewGroup("group-1", "light", "Lights", clientsapi.Service_LIGHT, members)
}

func TestGroupFeaturesAndCommandsAreIntersected(t *testing.T) {
	group := testGroup(
		&GroupMember{DeviceID: "dev-a", EndpointID: "1", ServiceID: "light"},
		&GroupMember{DeviceID: "dev-b", EndpointID: "1", ServiceID: "light"},
		&GroupMember{DeviceID: "dev-c", EndpointID: "1", ServiceID: "light"},
	)

	// Before any member reported there is nothing to offer.
	if features, commands := group.serviceInfo(); len(features) != 0 || len(commands) != 0 {
		t.Errorf("features %v, commands %v before any update", features, commands)
	}

	group.Update(lightDevice("dev-a",
		[]string{"brightness", "color_temp", "hs"},
		commandSpecWithParam("set", "from_a"), commandSpec("toggle"), commandSpec("flash")))
	group.Update(lightDevice("dev-b",
		[]string{"hs", "brightness"},
		commandSpec("toggle"), commandSpecWithParam("set", "from_b")))

	// dev-c has not reported, so only dev-a and dev-b count. Features keep the
	// order of the first member and commands are the first member's specs.
	features, commands := group.serviceInfo()
	if !slices.Equal(features, []string{"brightness", "hs"}) {
		t.Errorf("features = %v", features)
	}
	var ids []string
	for _, command := range commands {
		ids = append(ids, command.GetId())
	}
	if !slices.Equal(ids, []string{"set", "toggle"}) {
		t.Errorf("commands = %v", ids)
	}
	if got := commands[0].GetParams()[0].GetId(); got != "from_a" {
		t.Errorf("set spec comes from %q, want the first member", got)
	}

	// Once dev-c reports with fewer, the group offers fewer.
	group.Update(lightDevice("dev-c", []string{"brightness"}, commandSpec("flash"), commandSpec("toggle")))
	features, commands = group.serviceInfo()
	if !slices.Equal(features, []string{"brightness"}) {
		t.Errorf("features = %v", features)
	}
	if len(commands) != 1 || commands[0].GetId() != "toggle" {
		t.Errorf("commands = %v", commands)
	}

	// The service in the group's updates carries them, and no events.
	update := group.fullUpdate()
	if update.Typ != clientsapi.Device_GROUPED || update.ClientId != GroupClientID || !update.FullState || len(update.Endpoints) != 1 {
		t.Fatalf("update = %v", update)
	}
	ep := update.Endpoints[0]
	if ep.GetId() != RootEndpointID {
		t.Errorf("endpoint = %q", ep.GetId())
	}
	srv := findService(ep, "light")
	if srv == nil || srv.Typ != clientsapi.Service_LIGHT {
		t.Fatalf("group service missing: %v", ep)
	}
	if !slices.Equal(srv.Features, []string{"brightness"}) || len(srv.Commands) != 1 || len(srv.Events) != 0 {
		t.Errorf("group service features %v, commands %v, events %v", srv.Features, srv.Commands, srv.Events)
	}

	// A partial update from a member carries the current features and commands.
	partial := group.Update(&clientsapi.Device{
		Id: "dev-a",
		Endpoints: []*clientsapi.Endpoint{{
			Id: "1",
			Services: []*clientsapi.Service{{
				Id:       "light",
				Typ:      clientsapi.Service_LIGHT,
				Features: []string{"brightness", "color_temp", "hs"},
				Commands: []*clientsapi.CommandSpec{commandSpec("toggle")},
				Attrs:    []*clientsapi.Attribute{intAttr("brightness", 70, 0, 100, rw)},
			}},
		}},
	})
	if partial == nil || partial.FullState {
		t.Fatalf("partial = %v", partial)
	}
	srv = findService(partial.Endpoints[0], "light")
	if srv == nil || len(srv.Features) != 1 || len(srv.Commands) != 1 {
		t.Errorf("partial group service = %v", srv)
	}
}

func TestGroupFullUpdate(t *testing.T) {
	group := testGroup(
		&GroupMember{DeviceID: "dev-a", EndpointID: "1", ServiceID: "light"},
		&GroupMember{DeviceID: "dev-b", EndpointID: "2", ServiceID: "light"},
	)

	// Only the service on the member's endpoint counts.
	other := lightDevice("dev-a", nil)
	other.Endpoints[1].Id = "5"
	if update := group.Update(other); update != nil {
		if srv := findService(update.Endpoints[0], "light"); srv != nil {
			t.Errorf("service on another endpoint matched: %v", srv)
		}
	}

	group.Update(lightDevice("dev-a", nil))
	update := group.fullUpdate()

	ep := update.Endpoints[0]
	for _, id := range []string{"info", "online", "light"} {
		if findService(ep, id) == nil {
			t.Errorf("service %q missing from the full update", id)
		}
	}
	name := findAttr(findService(ep, "info"), "name")
	if name.GetPerms() != rw || name.GetSpec().GetText() == nil || name.GetValue().GetText() != "Lights" {
		t.Errorf("name = %v", name)
	}
	srv := findService(ep, "light")
	if !slices.Equal(attrIDs(srv), []string{"brightness", "on"}) {
		t.Errorf("group attributes = %v", attrIDs(srv))
	}
	if findAttr(srv, "on").GetSpec().GetBool() == nil {
		t.Errorf("group attribute has no spec")
	}
	if !findAttr(findService(ep, "online"), "online").GetValue().GetBool() {
		t.Errorf("group not online")
	}

	// The device can be stored: a full update has all it needs.
	dev := newTestDevice(t, update)
	if !dev.isOnline() || dev.service("0", "info") == nil || dev.service("0", "light") == nil {
		t.Errorf("stored group device = %v", dev.pb())
	}
}

func TestVerifyGroupMembers(t *testing.T) {
	deviceManager := testDeviceManager()
	manager := &GroupManager{deviceManager: deviceManager}

	member := func(device, endpoint, service string) []*GroupMember {
		return []*GroupMember{{DeviceID: device, EndpointID: endpoint, ServiceID: service}}
	}

	tests := []struct {
		name    string
		typ     clientsapi.Service_ServiceType
		members []*GroupMember
		wantErr bool
	}{
		{"valid", clientsapi.Service_LIGHT, member("dev-1", "1", "light"), false},
		{"unknown device", clientsapi.Service_LIGHT, member("dev-9", "1", "light"), true},
		{"unknown endpoint", clientsapi.Service_LIGHT, member("dev-1", "9", "light"), true},
		{"endpoint missing", clientsapi.Service_LIGHT, member("dev-1", "", "light"), true},
		{"service on another endpoint", clientsapi.Service_LIGHT, member("dev-1", "0", "light"), true},
		{"unknown service", clientsapi.Service_LIGHT, member("dev-1", "1", "dimmer"), true},
		{"wrong type", clientsapi.Service_ON_OFF, member("dev-1", "1", "light"), true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// GetDevice reads the device under the manager lock.
			err := manager.verifyGroupMembers(test.typ, test.members)
			if (err != nil) != test.wantErr {
				t.Errorf("err = %v, want error %t", err, test.wantErr)
			}
		})
	}
}

func receive[T any](t *testing.T, sub *queue.Sub[T], what string) T {
	t.Helper()
	select {
	case v := <-sub.Sub():
		return v
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
	panic("unreachable")
}

// A command sent to the group is sent to every member's endpoint and service
// as it is, and the group completes without a result.
func TestGroupHandleRequestFansOutCommand(t *testing.T) {
	deviceManager := &DeviceManager{
		devices: map[string]*Device{
			"dev-a": {ClientID: "client-a", ID: "dev-a"},
			"dev-b": {ClientID: "client-b", ID: "dev-b"},
		},
		actionRequests:  queue.NewPub[ActionRequest](),
		actionResponses: queue.NewPub[ActionResponse](),
	}
	group := testGroup(
		&GroupMember{DeviceID: "dev-a", EndpointID: "1", ServiceID: "light"},
		&GroupMember{DeviceID: "dev-b", EndpointID: "3", ServiceID: "bulb"},
	)

	requests := deviceManager.GetActionRequests()
	defer requests.Close()
	responses := deviceManager.GetActionResponses()
	defer responses.Close()

	req := commandAction(group.GroupID, RootEndpointID, group.ServiceID, "set", values.Field("brightness", values.Int(40)))
	req.ActionId = "group-action"

	done := make(chan struct{})
	go func() {
		defer close(done)
		group.HandleRequest(req, deviceManager)
	}()

	type target struct{ device, endpoint, service, client string }
	want := map[target]bool{
		{"dev-a", "1", "light", "client-a"}: true,
		{"dev-b", "3", "bulb", "client-b"}:  true,
	}
	var memberReqs []*clientsapi.ActionRequest
	for range 2 {
		memberReq := receive(t, requests, "a member request")
		got := target{memberReq.Request.GetDeviceId(), memberReq.Request.GetEndpointId(), memberReq.Request.GetServiceId(), memberReq.ClientID}
		if !want[got] {
			t.Errorf("unexpected member request %v", got)
		}
		delete(want, got)

		cmd := memberReq.Request.GetCommand()
		if cmd == nil || cmd.GetId() != "set" || len(cmd.GetParams()) != 1 || cmd.GetParams()[0].GetValue().GetInt() != 40 {
			t.Errorf("member command = %v", memberReq.Request.GetAction())
		}
		if memberReq.Request.GetActionId() == "" || memberReq.Request.GetActionId() == req.GetActionId() {
			t.Errorf("member action id = %q", memberReq.Request.GetActionId())
		}
		memberReqs = append(memberReqs, memberReq.Request)
	}
	if len(want) != 0 {
		t.Errorf("members not sent a request: %v", want)
	}

	// The requester is told it was sent.
	if got := receive(t, responses, "the sent response"); got.ClientID != GroupClientID || got.Response.GetStatus() != clientsapi.ActionResponse_SENT {
		t.Errorf("first response = %v", got)
	}

	// Members complete, one with a result.
	for i, memberReq := range memberReqs {
		res := &clientsapi.ActionResponse{ActionId: memberReq.GetActionId(), Status: clientsapi.ActionResponse_COMPLETE}
		if i == 0 {
			res.Result = []*clientsapi.FieldValue{values.Field("answer", values.Int(42))}
		}
		deviceManager.PushActionResponse(deviceManager.devices[memberReq.GetDeviceId()].ClientID, res, false)
	}

	for {
		got := receive(t, responses, "the final response")
		if got.ClientID != GroupClientID {
			continue
		}
		if got.Response.GetActionId() != req.GetActionId() || got.Response.GetStatus() != clientsapi.ActionResponse_COMPLETE {
			t.Fatalf("final response = %v", got)
		}
		if len(got.Response.GetResult()) != 0 {
			t.Errorf("group result = %v, want none", got.Response.GetResult())
		}
		break
	}
	<-done
}

// A write sent to the group is sent to the members as it is too.
func TestGroupHandleRequestFansOutWrite(t *testing.T) {
	deviceManager := &DeviceManager{
		devices:         map[string]*Device{"dev-a": {ClientID: "client-a", ID: "dev-a"}},
		actionRequests:  queue.NewPub[ActionRequest](),
		actionResponses: queue.NewPub[ActionResponse](),
	}
	group := testGroup(&GroupMember{DeviceID: "dev-a", EndpointID: "2", ServiceID: "light"})

	requests := deviceManager.GetActionRequests()
	defer requests.Close()

	req := writeAction(group.GroupID, RootEndpointID, group.ServiceID, values.Field("on", values.Bool(true)))
	req.ActionId = "group-action"
	go group.HandleRequest(req, deviceManager)

	memberReq := receive(t, requests, "a member request").Request
	if memberReq.GetEndpointId() != "2" || memberReq.GetServiceId() != "light" {
		t.Errorf("member request = %v", memberReq)
	}
	write := memberReq.GetWrite()
	if write == nil || len(write.GetValues()) != 1 || write.GetValues()[0].GetId() != "on" || !write.GetValues()[0].GetValue().GetBool() {
		t.Errorf("member write = %v", memberReq.GetAction())
	}

	deviceManager.PushActionResponse("client-a", &clientsapi.ActionResponse{ActionId: memberReq.GetActionId(), Status: clientsapi.ActionResponse_COMPLETE}, false)
}
