package core

import (
	"testing"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func testDeviceManager() *DeviceManager {
	return &DeviceManager{
		devices: map[string]*Device{
			"dev-1": {
				ClientID: "client-1",
				ID:       "dev-1",
				Typ:      clientsapi.Device_DEVICE,
				Services: map[string]*clientsapi.Service{
					"info": {
						Id:  "info",
						Typ: clientsapi.Service_INFO,
						Attrs: []*clientsapi.Attribute{
							{
								Id:   "name",
								Text: &clientsapi.TextAttribute{Value: "banana", Perms: clientsapi.Permissions_PERM_READWRITE},
							},
							{
								Id:   "model",
								Text: &clientsapi.TextAttribute{Value: "b-1000", Perms: clientsapi.Permissions_PERM_READONLY},
							},
							{
								Id: "mystery",
							},
						},
					},
				},
			},
		},
	}
}

func textValue(id, value string) *clientsapi.Value {
	return &clientsapi.Value{Id: id, Text: &clientsapi.TextValue{Value: value}}
}

func TestValidateActionPerms(t *testing.T) {
	manager := testDeviceManager()

	tests := []struct {
		name      string
		deviceID  string
		serviceID string
		values    []*clientsapi.Value
		wantCode  codes.Code
	}{
		{
			name:      "readwrite attribute passes",
			deviceID:  "dev-1",
			serviceID: "info",
			values:    []*clientsapi.Value{textValue("name", "new name")},
			wantCode:  codes.OK,
		},
		{
			name:      "readonly attribute rejected",
			deviceID:  "dev-1",
			serviceID: "info",
			values:    []*clientsapi.Value{textValue("model", "nope")},
			wantCode:  codes.FailedPrecondition,
		},
		{
			name:      "readonly rejected even alongside writable",
			deviceID:  "dev-1",
			serviceID: "info",
			values:    []*clientsapi.Value{textValue("name", "new name"), textValue("model", "nope")},
			wantCode:  codes.FailedPrecondition,
		},
		{
			name:      "unknown attribute passes through to the bridge",
			deviceID:  "dev-1",
			serviceID: "info",
			values:    []*clientsapi.Value{textValue("write_only", "on")},
			wantCode:  codes.OK,
		},
		{
			name:      "undefined perms passes through to the bridge",
			deviceID:  "dev-1",
			serviceID: "info",
			values:    []*clientsapi.Value{textValue("mystery", "on")},
			wantCode:  codes.OK,
		},
		{
			name:      "missing device",
			deviceID:  "dev-2",
			serviceID: "info",
			values:    []*clientsapi.Value{textValue("name", "new name")},
			wantCode:  codes.NotFound,
		},
		{
			name:      "missing service",
			deviceID:  "dev-1",
			serviceID: "lightbulb",
			values:    []*clientsapi.Value{textValue("name", "new name")},
			wantCode:  codes.NotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := manager.ValidateActionPerms(test.deviceID, test.serviceID, test.values)
			if got := status.Code(err); got != test.wantCode {
				t.Errorf("expected code %s, got %s (err: %v)", test.wantCode, got, err)
			}
		})
	}
}

func TestAttributePerms(t *testing.T) {
	tests := []struct {
		name string
		attr *clientsapi.Attribute
		want clientsapi.Permissions
	}{
		{
			name: "text",
			attr: &clientsapi.Attribute{Text: &clientsapi.TextAttribute{Perms: clientsapi.Permissions_PERM_READONLY}},
			want: clientsapi.Permissions_PERM_READONLY,
		},
		{
			name: "bool",
			attr: &clientsapi.Attribute{Bool: &clientsapi.BoolAttribute{Perms: clientsapi.Permissions_PERM_READWRITE}},
			want: clientsapi.Permissions_PERM_READWRITE,
		},
		{
			name: "float",
			attr: &clientsapi.Attribute{Float: &clientsapi.FloatAttribute{Perms: clientsapi.Permissions_PERM_WRITEONLY}},
			want: clientsapi.Permissions_PERM_WRITEONLY,
		},
		{
			name: "no kind set",
			attr: &clientsapi.Attribute{},
			want: clientsapi.Permissions_PERM_UNDEFINED,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := attributePerms(test.attr); got != test.want {
				t.Errorf("expected %s, got %s", test.want, got)
			}
		})
	}
}
