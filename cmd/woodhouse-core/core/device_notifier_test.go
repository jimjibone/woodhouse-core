package core

import (
	"testing"
	"time"

	"github.com/jimjibone/queue/v2"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

const (
	testHubID    = "examplehub1"
	testHubName  = "Example Hub"
	testZigbeeID = "zigbee0000000000000001"
)

func namedDevice(id, name string) *clientsapi.Device {
	dev := fullStateDevice(id, true)
	dev.Endpoints[0].Services[0].Attrs[0].Value = values.Text(name)
	return dev
}

// pairingHub is a full state device with a pairing service on endpoint "1" and
// a button service on endpoint "2" that also declares an event named "paired".
func pairingHub(id, name string) *clientsapi.Device {
	dev := namedDevice(id, name)
	dev.Endpoints = append(dev.Endpoints,
		&clientsapi.Endpoint{
			Id: "1",
			Services: []*clientsapi.Service{{
				Id:  "pairing",
				Typ: clientsapi.Service_PAIRING,
				Events: []*clientsapi.EventSpec{
					{
						Id: "paired",
						Fields: []*clientsapi.FieldSpec{
							{Id: "device_id", Spec: textSpec(), Required: true},
							{Id: "name", Spec: textSpec()},
						},
					},
					{
						Id: "failed",
						Fields: []*clientsapi.FieldSpec{
							{Id: "device_id", Spec: textSpec()},
							{Id: "error", Spec: textSpec(), Required: true},
						},
					},
				},
			}},
		},
		&clientsapi.Endpoint{
			Id: "2",
			Services: []*clientsapi.Service{{
				Id:  "button",
				Typ: clientsapi.Service_BUTTON,
				Events: []*clientsapi.EventSpec{{
					Id:     "paired",
					Fields: []*clientsapi.FieldSpec{{Id: "device_id", Spec: textSpec(), Required: true}},
				}},
			}},
		},
	)
	return dev
}

func pairedEvent(deviceID, name string) *clientsapi.Event {
	event := &clientsapi.Event{
		EndpointId: "1",
		ServiceId:  "pairing",
		Id:         "paired",
		Fields:     []*clientsapi.FieldValue{values.Field("device_id", values.Text(deviceID))},
	}
	if name != "" {
		event.Fields = append(event.Fields, values.Field("name", values.Text(name)))
	}
	return event
}

func failedEvent(deviceID, msg string) *clientsapi.Event {
	event := &clientsapi.Event{
		EndpointId: "1",
		ServiceId:  "pairing",
		Id:         "failed",
		Fields:     []*clientsapi.FieldValue{values.Field("error", values.Text(msg))},
	}
	if deviceID != "" {
		event.Fields = append(event.Fields, values.Field("device_id", values.Text(deviceID)))
	}
	return event
}

type notifierFixture struct {
	devices       *DeviceManager
	notifications *NotificationManager
	lis           *queue.Sub[NotificationUpdate]
}

func newNotifierFixture(t *testing.T) *notifierFixture {
	t.Helper()
	devices, err := NewDeviceManager(stores.NewMemStore())
	if err != nil {
		t.Fatalf("NewDeviceManager: %s", err)
	}
	t.Cleanup(devices.Close)
	notifications := newTestNotificationManager(t, stores.NewMemStore(), newTestUserManager(t))
	lis := notifications.GetListener()
	t.Cleanup(lis.Close)

	pushUpdate(devices, pairingHub(testHubID, testHubName))

	// Cleanups run last in, first out: the notifier closes before the managers.
	notifier := NewDeviceNotifier(devices, notifications)
	t.Cleanup(notifier.Close)
	return &notifierFixture{devices: devices, notifications: notifications, lis: lis}
}

func (f *notifierFixture) emit(events ...*clientsapi.Event) {
	pushUpdate(f.devices, eventsUpdate(testHubID, events...))
}

// next returns the next added notification.
func (f *notifierFixture) next(t *testing.T) *Notification {
	t.Helper()
	select {
	case update := <-f.lis.Sub():
		if update.Updated == nil {
			t.Fatalf("expected an added notification, got %+v", update)
		}
		return update.Updated
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for a notification")
		return nil
	}
}

func (f *notifierFixture) adminInbox() []*Notification {
	return f.notifications.Snapshot("alice", auth.AdminRole)
}

func wantNotification(t *testing.T, n *Notification, title, body string, level Level, source, link, collapse string) {
	t.Helper()
	if n.Title != title {
		t.Errorf("title = %q, want %q", n.Title, title)
	}
	if n.Body != body {
		t.Errorf("body = %q, want %q", n.Body, body)
	}
	if n.Level != level {
		t.Errorf("level = %s, want %s", n.Level, level)
	}
	if n.Source != source {
		t.Errorf("source = %q, want %q", n.Source, source)
	}
	if n.Category != "devices" {
		t.Errorf("category = %q, want devices", n.Category)
	}
	if n.Link != link {
		t.Errorf("link = %q, want %q", n.Link, link)
	}
	if n.CollapseKey != collapse {
		t.Errorf("collapse key = %q, want %q", n.CollapseKey, collapse)
	}
	if n.Audience != AudienceRole(auth.AdminRole) {
		t.Errorf("audience = %s, want role:admin", n.Audience)
	}
	if n.ID == "" {
		t.Errorf("empty id")
	}
}

func TestDeviceNotifierPairedWithName(t *testing.T) {
	f := newNotifierFixture(t)

	f.emit(pairedEvent(testZigbeeID, "Desk Lamp"))
	n := f.next(t)
	wantNotification(t, n, "New device paired", "Desk Lamp joined via Example Hub", SuccessLevel,
		"device-paired", "/devices/"+testZigbeeID, "paired:"+testZigbeeID)
}

func TestDeviceNotifierPairedUsesKnownDeviceName(t *testing.T) {
	f := newNotifierFixture(t)
	pushUpdate(f.devices, namedDevice(testZigbeeID, "Kitchen Plug"))

	f.emit(pairedEvent(testZigbeeID, ""))
	n := f.next(t)
	wantNotification(t, n, "New device paired", "Kitchen Plug joined via Example Hub", SuccessLevel,
		"device-paired", "/devices/"+testZigbeeID, "paired:"+testZigbeeID)
}

func TestDeviceNotifierPairedUnknownDeviceUsesID(t *testing.T) {
	f := newNotifierFixture(t)

	f.emit(pairedEvent(testZigbeeID, ""))
	n := f.next(t)
	wantNotification(t, n, "New device paired", testZigbeeID+" joined via Example Hub", SuccessLevel,
		"device-paired", "/devices/"+testZigbeeID, "paired:"+testZigbeeID)
}

func TestDeviceNotifierBridgeNameFallsBackToID(t *testing.T) {
	f := newNotifierFixture(t)
	// An unnamed hub is shown by id.
	pushUpdate(f.devices, pairingHub("examplehub2", ""))
	pushUpdate(f.devices, eventsUpdate("examplehub2", pairedEvent(testZigbeeID, "Desk Lamp")))
	n := f.next(t)
	if n.Body != "Desk Lamp joined via examplehub2" {
		t.Errorf("body = %q", n.Body)
	}
}

func TestDeviceNotifierFailed(t *testing.T) {
	f := newNotifierFixture(t)

	// Without a device.
	f.emit(failedEvent("", "no devices answered"))
	n := f.next(t)
	wantNotification(t, n, "Pairing failed", "no devices answered (via Example Hub)", WarningLevel,
		"device-pairing-failed", "/devices/"+testHubID, "pairing-failed:"+testHubID+":")

	// With an unknown device: shown by id.
	f.emit(failedEvent(testZigbeeID, "interview timed out"))
	n = f.next(t)
	wantNotification(t, n, "Pairing failed", testZigbeeID+": interview timed out (via Example Hub)", WarningLevel,
		"device-pairing-failed", "/devices/"+testZigbeeID, "pairing-failed:"+testHubID+":"+testZigbeeID)

	// With a known device: shown by name.
	pushUpdate(f.devices, namedDevice("zigbee0000000000000002", "Desk Lamp"))
	f.emit(failedEvent("zigbee0000000000000002", "interview timed out"))
	n = f.next(t)
	if n.Body != "Desk Lamp: interview timed out (via Example Hub)" {
		t.Errorf("body = %q", n.Body)
	}
}

func TestDeviceNotifierIgnoresNonPairingService(t *testing.T) {
	f := newNotifierFixture(t)

	// Event id "paired" on a button service.
	f.emit(&clientsapi.Event{
		EndpointId: "2",
		ServiceId:  "button",
		Id:         "paired",
		Fields:     []*clientsapi.FieldValue{values.Field("device_id", values.Text("zigbee0000000000000002"))},
	})
	// Events are handled in order, so once the sentinel arrives the ignored
	// one has been seen.
	f.emit(pairedEvent(testZigbeeID, "Desk Lamp"))
	n := f.next(t)
	if n.CollapseKey != "paired:"+testZigbeeID {
		t.Errorf("first notification = %s", n)
	}
	if got := len(f.adminInbox()); got != 1 {
		t.Errorf("inbox has %d notifications, want 1", got)
	}
}

func TestDeviceNotifierIgnoresReplayed(t *testing.T) {
	f := newNotifierFixture(t)

	// The device manager clears the flag on ingest, so deliver a replayed
	// event straight to the handler.
	notifier := &DeviceNotifier{deviceManager: f.devices, notificationManager: f.notifications}
	replayed := pairedEvent(testZigbeeID, "Desk Lamp")
	replayed.Replayed = true
	notifier.handleEvent(testHubID, replayed)

	f.emit(pairedEvent("zigbee0000000000000002", "Hall Light"))
	n := f.next(t)
	if n.CollapseKey != "paired:zigbee0000000000000002" {
		t.Errorf("first notification = %s", n)
	}
	if got := len(f.adminInbox()); got != 1 {
		t.Errorf("inbox has %d notifications, want 1", got)
	}
}

func TestDeviceNotifierCollapsesRepair(t *testing.T) {
	f := newNotifierFixture(t)

	f.emit(pairedEvent(testZigbeeID, "Desk Lamp"))
	first := f.next(t)
	f.emit(pairedEvent(testZigbeeID, "Desk Lamp"))

	removed := <-f.lis.Sub()
	if removed.Removed == nil || *removed.Removed != first.ID {
		t.Fatalf("expected removal of %q, got %+v", first.ID, removed)
	}
	second := f.next(t)
	if second.ID == first.ID {
		t.Errorf("re-pair reused id %q", first.ID)
	}
	inbox := f.adminInbox()
	if len(inbox) != 1 || inbox[0].ID != second.ID {
		t.Errorf("inbox = %v, want only %q", inbox, second.ID)
	}
}

func TestDeviceNotifierAudienceIsAdminOnly(t *testing.T) {
	f := newNotifierFixture(t)

	f.emit(pairedEvent(testZigbeeID, "Desk Lamp"))
	f.next(t)
	if got := len(f.notifications.Snapshot("alice", auth.AdminRole)); got != 1 {
		t.Errorf("admin sees %d, want 1", got)
	}
	if got := len(f.notifications.Snapshot("bob", auth.UserRole)); got != 0 {
		t.Errorf("user sees %d, want 0", got)
	}
}

func TestDeviceNotifierCreatedFromEventTime(t *testing.T) {
	f := newNotifierFixture(t)

	// Recent, or the manager trims it as too old.
	at := time.Now().Add(-time.Hour).Truncate(time.Second).UTC()
	event := pairedEvent(testZigbeeID, "Desk Lamp")
	event.Time = values.TimeToPb(at)
	f.emit(event)
	if n := f.next(t); !n.Created.Equal(at) {
		t.Errorf("created = %s, want %s", n.Created, at)
	}
}
