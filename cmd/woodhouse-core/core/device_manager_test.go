package core

import (
	"errors"
	"strings"
	"testing"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func boolSpec() *clientsapi.Spec {
	return &clientsapi.Spec{Kind: &clientsapi.Spec_Bool{Bool: &clientsapi.BoolSpec{}}}
}

func textSpec() *clientsapi.Spec {
	return &clientsapi.Spec{Kind: &clientsapi.Spec_Text{Text: &clientsapi.TextSpec{}}}
}

func intSpec(min, max int64) *clientsapi.Spec {
	return &clientsapi.Spec{Kind: &clientsapi.Spec_Int{Int: &clientsapi.IntSpec{Min: &min, Max: &max}}}
}

func textAttr(id, value string, perms clientsapi.Permissions) *clientsapi.Attribute {
	return &clientsapi.Attribute{Id: id, Perms: perms, Spec: textSpec(), Value: values.Text(value)}
}

func boolAttr(id string, value bool, perms clientsapi.Permissions) *clientsapi.Attribute {
	return &clientsapi.Attribute{Id: id, Perms: perms, Spec: boolSpec(), Value: values.Bool(value)}
}

func intAttr(id string, value, min, max int64, perms clientsapi.Permissions) *clientsapi.Attribute {
	return &clientsapi.Attribute{Id: id, Perms: perms, Spec: intSpec(min, max), Value: values.Int(value)}
}

// testDeviceManager holds one device, dev-1, with an info service on the root
// endpoint and a light service on endpoint "1" with a brightness attribute and
// a set command.
func testDeviceManager() *DeviceManager {
	return &DeviceManager{
		devices: map[string]*Device{
			"dev-1": {
				ClientID: "client-1",
				ID:       "dev-1",
				Typ:      clientsapi.Device_DEVICE,
				Endpoints: map[string]*DeviceEndpoint{
					RootEndpointID: {
						ID: RootEndpointID,
						Services: map[string]*clientsapi.Service{
							"info": {
								Id:  "info",
								Typ: clientsapi.Service_INFO,
								Attrs: []*clientsapi.Attribute{
									textAttr("name", "banana", clientsapi.Permissions_PERM_READWRITE),
									textAttr("model", "b-1000", clientsapi.Permissions_PERM_READONLY),
								},
							},
						},
					},
					"1": {
						ID: "1",
						Services: map[string]*clientsapi.Service{
							"light": {
								Id:  "light",
								Typ: clientsapi.Service_LIGHT,
								Attrs: []*clientsapi.Attribute{
									boolAttr("on", false, clientsapi.Permissions_PERM_READWRITE),
									intAttr("brightness", 10, 0, 100, clientsapi.Permissions_PERM_READWRITE),
									intAttr("power", 5, 0, 100, clientsapi.Permissions_PERM_READONLY),
								},
								Commands: []*clientsapi.CommandSpec{
									{
										Id: "set",
										Params: []*clientsapi.FieldSpec{
											{Id: "brightness", Spec: intSpec(0, 100)},
											{Id: "transition", Spec: textSpec(), Required: true},
										},
									},
									{Id: "toggle"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func writeAction(deviceID, endpointID, serviceID string, fields ...*clientsapi.FieldValue) *clientsapi.ActionRequest {
	return &clientsapi.ActionRequest{
		DeviceId:   deviceID,
		EndpointId: endpointID,
		ServiceId:  serviceID,
		Action:     &clientsapi.ActionRequest_Write{Write: &clientsapi.WriteAction{Values: fields}},
	}
}

func commandAction(deviceID, endpointID, serviceID, command string, params ...*clientsapi.FieldValue) *clientsapi.ActionRequest {
	return &clientsapi.ActionRequest{
		DeviceId:   deviceID,
		EndpointId: endpointID,
		ServiceId:  serviceID,
		Action:     &clientsapi.ActionRequest_Command{Command: &clientsapi.CommandAction{Id: command, Params: params}},
	}
}

func TestValidateAction(t *testing.T) {
	manager := testDeviceManager()

	tests := []struct {
		name       string
		req        *clientsapi.ActionRequest
		wantCode   codes.Code // Of a gRPC status error.
		wantStatus clientsapi.ActionResponse_ActionStatus
	}{
		{
			name: "valid write",
			req:  writeAction("dev-1", "0", "info", values.Field("name", values.Text("new name"))),
		},
		{
			name: "valid write of several attributes",
			req: writeAction("dev-1", "1", "light",
				values.Field("on", values.Bool(true)),
				values.Field("brightness", values.Int(50))),
		},
		{
			name: "valid command",
			req: commandAction("dev-1", "1", "light", "set",
				values.Field("brightness", values.Int(50)),
				values.Field("transition", values.Text("fast"))),
		},
		{
			name: "valid command without params",
			req:  commandAction("dev-1", "1", "light", "toggle"),
		},
		{
			name:       "write to readonly attribute",
			req:        writeAction("dev-1", "0", "info", values.Field("model", values.Text("nope"))),
			wantStatus: clientsapi.ActionResponse_ERR_READONLY,
		},
		{
			name: "readonly rejected alongside writable",
			req: writeAction("dev-1", "0", "info",
				values.Field("name", values.Text("new name")),
				values.Field("model", values.Text("nope"))),
			wantStatus: clientsapi.ActionResponse_ERR_READONLY,
		},
		{
			name:       "write with no values",
			req:        writeAction("dev-1", "0", "info"),
			wantStatus: clientsapi.ActionResponse_ERR_INVALID,
		},
		{
			name:       "write to unknown attribute",
			req:        writeAction("dev-1", "0", "info", values.Field("mystery", values.Text("on"))),
			wantStatus: clientsapi.ActionResponse_ERR_INVALID,
		},
		{
			name:       "write of the wrong type",
			req:        writeAction("dev-1", "0", "info", values.Field("name", values.Bool(true))),
			wantStatus: clientsapi.ActionResponse_ERR_INVALID,
		},
		{
			name:       "write out of range",
			req:        writeAction("dev-1", "1", "light", values.Field("brightness", values.Int(101))),
			wantStatus: clientsapi.ActionResponse_ERR_INVALID,
		},
		{
			name:       "write with no value",
			req:        writeAction("dev-1", "0", "info", values.Field("name", nil)),
			wantStatus: clientsapi.ActionResponse_ERR_INVALID,
		},
		{
			name:       "unknown command",
			req:        commandAction("dev-1", "1", "light", "explode"),
			wantStatus: clientsapi.ActionResponse_ERR_INVALID,
		},
		{
			name:       "command with a param out of range",
			req:        commandAction("dev-1", "1", "light", "set", values.Field("brightness", values.Int(500)), values.Field("transition", values.Text("fast"))),
			wantStatus: clientsapi.ActionResponse_ERR_INVALID,
		},
		{
			name:       "command with a missing required param",
			req:        commandAction("dev-1", "1", "light", "set", values.Field("brightness", values.Int(50))),
			wantStatus: clientsapi.ActionResponse_ERR_INVALID,
		},
		{
			name:       "command with an unknown param",
			req:        commandAction("dev-1", "1", "light", "toggle", values.Field("extra", values.Int(1))),
			wantStatus: clientsapi.ActionResponse_ERR_INVALID,
		},
		{
			name:       "neither write nor command",
			req:        &clientsapi.ActionRequest{DeviceId: "dev-1", EndpointId: "0", ServiceId: "info"},
			wantStatus: clientsapi.ActionResponse_ERR_INVALID,
		},
		{
			name:     "missing device",
			req:      writeAction("dev-2", "0", "info", values.Field("name", values.Text("new name"))),
			wantCode: codes.NotFound,
		},
		{
			name:     "missing endpoint",
			req:      writeAction("dev-1", "9", "info", values.Field("name", values.Text("new name"))),
			wantCode: codes.NotFound,
		},
		{
			name:     "service on another endpoint",
			req:      writeAction("dev-1", "1", "info", values.Field("name", values.Text("new name"))),
			wantCode: codes.NotFound,
		},
		{
			name:     "missing service",
			req:      writeAction("dev-1", "0", "lightbulb", values.Field("name", values.Text("new name"))),
			wantCode: codes.NotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := manager.ValidateAction(test.req)

			var actionErr *ActionError
			switch {
			case test.wantCode != codes.OK:
				if got := status.Code(err); got != test.wantCode {
					t.Errorf("expected code %s, got %s (err: %v)", test.wantCode, got, err)
				}
			case test.wantStatus != clientsapi.ActionResponse_UNDEFINED:
				if !errors.As(err, &actionErr) {
					t.Fatalf("expected an ActionError, got %v", err)
				}
				if actionErr.Status != test.wantStatus {
					t.Errorf("expected status %s, got %s (%s)", test.wantStatus, actionErr.Status, actionErr.Details)
				}
				if actionErr.Details == "" {
					t.Errorf("expected details")
				}
			default:
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

// Events in a device update are not stored, in the snapshot or in devices.json.
func TestDeviceManagerEventsNotStored(t *testing.T) {
	store := stores.NewMemStore()
	manager, err := NewDeviceManager(store)
	if err != nil {
		t.Fatalf("NewDeviceManager: %s", err)
	}
	defer manager.Close()

	sub := manager.GetDeviceUpdates()
	defer sub.Close()

	dev := fullStateDevice("dev-a", true)
	dev.Events = []*clientsapi.Event{{
		EndpointId: "0",
		ServiceId:  "info",
		Id:         "event-id-xyz",
		Fields:     []*clientsapi.FieldValue{values.Field("payload", values.Text("event-payload-xyz"))},
	}}
	manager.PushDeviceUpdate("client-1", dev)

	// The marker removal is queued behind the update, so once it is seen the
	// update was handled.
	manager.PushDeviceUpdate("client-1", fullStateDevice("dev-marker", true))
	manager.RemoveClientDevice("client-1", "dev-marker")
	var forwarded *clientsapi.Device
	timeout := time.After(5 * time.Second)
	for done := false; !done; {
		select {
		case update := <-sub.Sub():
			if update.Update.GetId() == "dev-a" {
				forwarded = update.Update
			}
			done = update.RemovedID == "dev-marker"
		case <-timeout:
			t.Fatalf("timed out waiting for removal of dev-marker")
		}
	}

	if forwarded == nil {
		t.Fatalf("update not forwarded")
	}
	if len(forwarded.Events) != 0 {
		t.Errorf("events forwarded: %v", forwarded.Events)
	}
	if got := manager.GetDevice("dev-a"); got == nil || len(got.Events) != 0 {
		t.Errorf("events in the snapshot: %v", got)
	}

	if err := manager.save(); err != nil {
		t.Fatalf("save: %s", err)
	}
	data, err := store.Get("devices.json")
	if err != nil {
		t.Fatalf("get devices.json: %s", err)
	}
	if !strings.Contains(string(data), "dev-a") {
		t.Fatalf("device not saved: %s", data)
	}
	for _, leaked := range []string{"event-id-xyz", "event-payload-xyz"} {
		if strings.Contains(string(data), leaked) {
			t.Errorf("devices.json contains %q", leaked)
		}
	}
}

// A saved device loads again with its endpoints, specs, values and attribute
// order.
func TestDeviceManagerSaveLoad(t *testing.T) {
	store := stores.NewMemStore()
	manager, err := NewDeviceManager(store)
	if err != nil {
		t.Fatalf("NewDeviceManager: %s", err)
	}
	sub := manager.GetDeviceUpdates()
	dev := fullStateDevice("dev-a", true)
	dev.Endpoints[0].Label = "Root"
	dev.Endpoints[0].Services[0].Features = []string{"feature-a"}
	dev.Endpoints[0].Services[0].Attrs = append(dev.Endpoints[0].Services[0].Attrs,
		intAttr("zeta", 7, 0, 10, clientsapi.Permissions_PERM_READWRITE),
		boolAttr("alpha", true, clientsapi.Permissions_PERM_READONLY))
	manager.PushDeviceUpdate("client-1", dev)
	manager.PushDeviceUpdate("client-1", fullStateDevice("dev-marker", true))
	manager.RemoveClientDevice("client-1", "dev-marker")
	waitRemoved(t, sub, "dev-marker")
	sub.Close()
	manager.Close()

	manager, err = NewDeviceManager(store)
	if err != nil {
		t.Fatalf("NewDeviceManager after restart: %s", err)
	}
	defer manager.Close()

	got := manager.GetDevice("dev-a")
	if got == nil {
		t.Fatalf("dev-a not loaded")
	}
	ep := got.GetEndpoints()[0]
	if ep.GetId() != RootEndpointID || ep.GetLabel() != "Root" {
		t.Errorf("endpoint = %v", ep)
	}
	var info *clientsapi.Service
	for _, srv := range ep.GetServices() {
		if srv.GetId() == "info" {
			info = srv
		}
	}
	if info == nil {
		t.Fatalf("info service not loaded: %v", ep)
	}
	var ids []string
	for _, attr := range info.GetAttrs() {
		ids = append(ids, attr.GetId())
	}
	if strings.Join(ids, ",") != "name,zeta,alpha" {
		t.Errorf("attribute order = %v", ids)
	}
	if attr := info.GetAttrs()[1]; attr.GetSpec().GetInt() == nil || attr.GetValue().GetInt() != 7 {
		t.Errorf("zeta = %v", attr)
	}
	if got := info.GetFeatures(); len(got) != 1 || got[0] != "feature-a" {
		t.Errorf("features = %v", got)
	}
}
