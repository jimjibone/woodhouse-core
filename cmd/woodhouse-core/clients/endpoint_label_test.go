package clients

import (
	"testing"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
)

// A user's endpoint label is for users: client streams carry no user_label, and
// a label change is not sent to them at all.
func TestDeviceStreamSkipsUserLabels(t *testing.T) {
	service, manager := newTestClientService(t)
	pushDevice(t, manager, buttonDevice("dev-a"))
	if err := manager.SetEndpointLabel("dev-a", "1", "Left"); err != nil {
		t.Fatalf("SetEndpointLabel: %s", err)
	}

	stream := startDeviceStream(t, service, &clientsapi.DeviceStreamRequest{})
	snapshot := stream.next(t)
	for _, ep := range snapshot.GetDevice().GetEndpoints() {
		if ep.GetUserLabel() != "" {
			t.Errorf("snapshot endpoint %q has user_label %q", ep.GetId(), ep.GetUserLabel())
		}
	}
	stream.next(t) // Marker.

	if err := manager.SetEndpointLabel("dev-a", "1", "Right"); err != nil {
		t.Fatalf("SetEndpointLabel: %s", err)
	}
	// The next thing the client hears about is the press, not the label.
	manager.PushDeviceUpdate("client-1", pressUpdate("dev-a"))
	res := stream.next(t)
	if len(res.GetDevice().GetEvents()) != 1 || len(res.GetDevice().GetEndpoints()) != 0 {
		t.Errorf("update after label change = %v, want only the press", res)
	}
}
