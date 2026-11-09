package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jimjibone/log"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/random"
)

const (
	// Event IDs of the pairing service.
	pairingPairedEvent = "paired"
	pairingFailedEvent = "failed"

	devicePairedSource        = "device-paired"
	devicePairingFailedSource = "device-pairing-failed"
	devicesCategory           = "devices"
)

// DeviceNotifier turns the events of bridge pairing services into admin
// notifications.
type DeviceNotifier struct {
	log                 *log.Context
	wg                  sync.WaitGroup
	close               func()
	deviceManager       *DeviceManager
	notificationManager *NotificationManager
}

func NewDeviceNotifier(deviceManager *DeviceManager, notificationManager *NotificationManager) *DeviceNotifier {
	ctx, close := context.WithCancel(context.Background())
	notifier := &DeviceNotifier{
		log:                 log.NewContext(log.DefaultLogger, "device-notifier", log.DebugLevel),
		close:               close,
		deviceManager:       deviceManager,
		notificationManager: notificationManager,
	}

	// Subscribe before the goroutine starts so no event published after
	// construction is missed.
	deviceUpdates := deviceManager.GetDeviceUpdates()

	notifier.wg.Add(1)
	go func() {
		defer notifier.wg.Done()
		defer deviceUpdates.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-deviceUpdates.Sub():
				if update.Update == nil {
					continue
				}
				for _, event := range update.Update.GetEvents() {
					notifier.handleEvent(update.Update.GetId(), event)
				}
			}
		}
	}()
	return notifier
}

func (notifier *DeviceNotifier) Close() {
	notifier.close()
	notifier.wg.Wait()
}

// handleEvent raises a notification for a pairing event of the device
// bridgeID. The service type is looked up in the device manager because an
// update may be partial and omit it.
func (notifier *DeviceNotifier) handleEvent(bridgeID string, event *clientsapi.Event) {
	if event.GetReplayed() {
		return
	}
	if event.GetId() != pairingPairedEvent && event.GetId() != pairingFailedEvent {
		return
	}

	bridge := notifier.deviceManager.GetDevice(bridgeID)
	if bridge == nil || serviceType(bridge, event.GetEndpointId(), event.GetServiceId()) != clientsapi.Service_PAIRING {
		return
	}

	fields := values.Fields(event.GetFields())
	deviceID := fields["device_id"].GetText()
	bridgeName := deviceDisplayName(bridge)

	var notification *Notification
	switch event.GetId() {
	case pairingPairedEvent:
		if deviceID == "" {
			return
		}
		name := fields["name"].GetText()
		if name == "" {
			if dev := notifier.deviceManager.GetDevice(deviceID); dev != nil {
				name = deviceName(dev)
			}
		}
		if name == "" {
			name = deviceID
		}
		notification = NewNotification(
			"",
			"New device paired",
			fmt.Sprintf("%s joined via %s", name, bridgeName),
			SuccessLevel,
			AudienceRole(auth.AdminRole),
		)
		notification.Source = devicePairedSource
		notification.CollapseKey = "paired:" + deviceID
		notification.Link = "/devices/" + deviceID

	case pairingFailedEvent:
		body := fields["error"].GetText()
		link := "/devices/" + bridgeID
		if deviceID != "" {
			who := deviceID
			if dev := notifier.deviceManager.GetDevice(deviceID); dev != nil {
				if name := deviceName(dev); name != "" {
					who = name
				}
			}
			body = who + ": " + body
			link = "/devices/" + deviceID
		}
		notification = NewNotification(
			"",
			"Pairing failed",
			fmt.Sprintf("%s (via %s)", body, bridgeName),
			WarningLevel,
			AudienceRole(auth.AdminRole),
		)
		notification.Source = devicePairingFailedSource
		notification.CollapseKey = fmt.Sprintf("pairing-failed:%s:%s", bridgeID, deviceID)
		notification.Link = link
	}

	notification.Category = devicesCategory
	if t := values.TimeFromPb(event.GetTime()); !t.IsZero() {
		notification.Created = t.UTC()
	} else {
		notification.Created = time.Now().UTC()
	}

	id, err := random.GenerateRandomString(10)
	if err != nil {
		notifier.log.Errorf("failed to generate notification id: %s", err)
		return
	}
	notification.ID = id

	if err := notifier.notificationManager.Add(notification); err != nil {
		notifier.log.Errorf("failed to add notification %s: %s", notification, err)
	}
}

// serviceType returns the type of a service of dev, or Service_UNDEFINED
// when it is not found.
func serviceType(dev *clientsapi.Device, endpointID, serviceID string) clientsapi.Service_ServiceType {
	for _, ep := range dev.GetEndpoints() {
		if ep.GetId() != endpointID {
			continue
		}
		for _, srv := range ep.GetServices() {
			if srv.GetId() == serviceID {
				return srv.GetTyp()
			}
		}
	}
	return clientsapi.Service_UNDEFINED
}

// deviceName returns the name attribute of the info service on the root
// endpoint, or empty.
func deviceName(dev *clientsapi.Device) string {
	for _, ep := range dev.GetEndpoints() {
		if ep.GetId() != RootEndpointID {
			continue
		}
		for _, srv := range ep.GetServices() {
			if srv.GetTyp() != clientsapi.Service_INFO {
				continue
			}
			for _, attr := range srv.GetAttrs() {
				if attr.GetId() == "name" {
					return attr.GetValue().GetText()
				}
			}
		}
	}
	return ""
}

// deviceDisplayName is the name of dev, else its id.
func deviceDisplayName(dev *clientsapi.Device) string {
	if name := deviceName(dev); name != "" {
		return name
	}
	return dev.GetId()
}
