package users

import (
	"context"
	"testing"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeGetDevicesStream is the server side of a GetDevices call.
type fakeGetDevicesStream struct {
	grpc.ServerStream
	devices []*clientsapi.Device
}

func (s *fakeGetDevicesStream) Context() context.Context { return context.Background() }

func (s *fakeGetDevicesStream) Send(dev *clientsapi.Device) error {
	s.devices = append(s.devices, dev)
	return nil
}

func endpointOf(dev *clientsapi.Device, id string) *clientsapi.Endpoint {
	for _, ep := range dev.GetEndpoints() {
		if ep.GetId() == id {
			return ep
		}
	}
	return nil
}

func TestSetEndpointLabelRPC(t *testing.T) {
	manager, _, _ := newDevicesStreamTest(t, 0, 0)
	service := &UserService{deviceManager: manager}

	set := func(ctx context.Context, deviceID, endpointID, label string) error {
		_, err := service.SetEndpointLabel(ctx, &clientsapi.SetEndpointLabelRequest{DeviceId: deviceID, EndpointId: endpointID, Label: label})
		return err
	}

	if err := set(claimsCtx("alice", auth.UserRole), "dev-a", "1", "Left"); status.Code(err) != codes.PermissionDenied {
		t.Errorf("non-admin: got %v, want PermissionDenied", err)
	}
	if err := set(claimsCtx("alice", auth.AdminRole), "", "1", "Left"); status.Code(err) != codes.InvalidArgument {
		t.Errorf("no device: got %v, want InvalidArgument", err)
	}
	if err := set(claimsCtx("alice", auth.AdminRole), "dev-a", "", "Left"); status.Code(err) != codes.InvalidArgument {
		t.Errorf("no endpoint: got %v, want InvalidArgument", err)
	}
	if err := set(claimsCtx("alice", auth.AdminRole), "ghost", "1", "Left"); status.Code(err) != codes.NotFound {
		t.Errorf("unknown device: got %v, want NotFound", err)
	}
	if err := set(claimsCtx("alice", auth.AdminRole), "dev-a", "9", "Left"); status.Code(err) != codes.NotFound {
		t.Errorf("unknown endpoint: got %v, want NotFound", err)
	}
	if got := manager.EndpointUserLabel("dev-a", "1"); got != "" {
		t.Errorf("label after rejected calls: got %q, want none", got)
	}

	if err := set(claimsCtx("alice", auth.AdminRole), "dev-a", "1", "  Left  "); err != nil {
		t.Fatalf("set: %s", err)
	}
	if got := manager.EndpointUserLabel("dev-a", "1"); got != "Left" {
		t.Errorf("label: got %q, want trimmed Left", got)
	}
	if err := set(claimsCtx("alice", auth.AdminRole), "dev-a", "1", ""); err != nil {
		t.Fatalf("clear: %s", err)
	}
	if got := manager.EndpointUserLabel("dev-a", "1"); got != "" {
		t.Errorf("label after clearing: got %q, want none", got)
	}
}

func TestGetDevicesAppliesUserLabels(t *testing.T) {
	manager, _, _ := newDevicesStreamTest(t, 0, 0)
	service := &UserService{deviceManager: manager}
	if err := manager.SetEndpointLabel("dev-a", "1", "Left"); err != nil {
		t.Fatalf("SetEndpointLabel: %s", err)
	}

	stream := &fakeGetDevicesStream{}
	if err := service.GetDevices(&clientsapi.GetDevicesRequest{}, stream); err != nil {
		t.Fatalf("GetDevices: %s", err)
	}
	if len(stream.devices) != 2 {
		t.Fatalf("devices = %d, want 2", len(stream.devices))
	}
	for _, dev := range stream.devices {
		want := ""
		if dev.GetId() == "dev-a" {
			want = "Left"
		}
		if got := endpointOf(dev, "1").GetUserLabel(); got != want {
			t.Errorf("device %q user_label: got %q, want %q", dev.GetId(), got, want)
		}
	}
}

// Labels are in the snapshots, and a change reaches an open stream as a
// partial update holding the endpoint and no services.
func TestDevicesStreamUserLabels(t *testing.T) {
	manager, _, start := newDevicesStreamTest(t, 0, 0)
	if err := manager.SetEndpointLabel("dev-b", "1", "Right"); err != nil {
		t.Fatalf("SetEndpointLabel: %s", err)
	}
	stream := start(&clientsapi.DevicesStreamRequest{})

	for i := 0; i < 2; i++ {
		res := stream.next(t)
		want := ""
		if res.GetDevice().GetId() == "dev-b" {
			want = "Right"
		}
		if got := endpointOf(res.GetDevice(), "1").GetUserLabel(); got != want {
			t.Errorf("snapshot of %q user_label: got %q, want %q", res.GetDevice().GetId(), got, want)
		}
	}
	if marker := stream.next(t); marker.GetDevice() != nil {
		t.Errorf("marker = %v", marker)
	}

	if err := manager.SetEndpointLabel("dev-a", "1", "Left"); err != nil {
		t.Fatalf("SetEndpointLabel: %s", err)
	}
	res := stream.next(t)
	dev := res.GetDevice()
	if dev.GetId() != "dev-a" || dev.GetFullState() || len(dev.GetEndpoints()) != 1 {
		t.Fatalf("label update = %v", res)
	}
	if ep := dev.GetEndpoints()[0]; ep.GetId() != "1" || ep.GetUserLabel() != "Left" || len(ep.GetServices()) != 0 {
		t.Errorf("label update endpoint = %v", ep)
	}

	if err := manager.SetEndpointLabel("dev-a", "1", ""); err != nil {
		t.Fatalf("SetEndpointLabel clear: %s", err)
	}
	res = stream.next(t)
	if ep := endpointOf(res.GetDevice(), "1"); res.GetDevice().GetId() != "dev-a" || ep == nil || ep.GetUserLabel() != "" {
		t.Errorf("clear update = %v", res)
	}
}

// Live updates of a labelled device carry the label without changing what the
// device manager shares with other streams.
func TestDevicesStreamLiveUpdateKeepsLabel(t *testing.T) {
	manager, _, start := newDevicesStreamTest(t, 0, 0)
	if err := manager.SetEndpointLabel("dev-a", "1", "Left"); err != nil {
		t.Fatalf("SetEndpointLabel: %s", err)
	}
	stream := start(&clientsapi.DevicesStreamRequest{IncludeDeviceIds: []string{"dev-a"}})
	stream.next(t) // Snapshot.
	stream.next(t) // Marker.

	update := buttonDevice("dev-a")
	update.FullState = false
	manager.PushDeviceUpdate("client-1", update)
	res := stream.next(t)
	if got := endpointOf(res.GetDevice(), "1").GetUserLabel(); got != "Left" {
		t.Errorf("live update user_label: got %q, want Left", got)
	}

	// The device manager's own state never holds the label.
	if got := endpointOf(manager.GetDevice("dev-a"), "1").GetUserLabel(); got != "" {
		t.Errorf("device manager state holds a user label: %q", got)
	}
}
