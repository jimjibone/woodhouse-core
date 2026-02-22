package shelly_v2

import (
	"fmt"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

// ShellyPlus2PM - device type: Plus2PM
type ShellyPlus2PM struct {
	shelly *ShellyComms
	info   *services.Info
	online *services.Online
	input0 *services.Input
	input1 *services.Input
	relay0 *services.Relay
	relay1 *services.Relay
}

func NewShellyPlus2PM(hostname, ip string, client *wh.Client) *ShellyPlus2PM {
	dev := &ShellyPlus2PM{
		shelly: NewShellyComms(hostname, ip, clientsapi.Device_DEVICE, client),
		info:   services.NewInfo(),
		online: services.NewOnline(),
		input0: services.NewInput("input0"),
		input1: services.NewInput("input1"),
		relay0: services.NewRelay("relay0"),
		relay1: services.NewRelay("relay1"),
	}

	initRelay(dev.relay0)
	initRelay(dev.relay1)

	dev.shelly.OnConnected(dev.onConnected)
	dev.shelly.OnDisconnected(dev.onDisconnected)
	dev.shelly.OnResponseFrame(dev.onResponseFrame)
	dev.shelly.OnNotificationFrame(dev.onNotificationFrame)
	dev.shelly.dev.AddService(
		dev.info,
		dev.online,
		dev.input0,
		dev.input1,
		dev.relay0,
		dev.relay1,
	)
	dev.info.Name.OnAction(dev.handleNameAction)
	dev.relay0.OnAction(dev.handleRelay0Action)
	dev.relay1.OnAction(dev.handleRelay1Action)

	return dev
}

func (dev *ShellyPlus2PM) ID() string {
	return dev.shelly.ID()
}

func (dev *ShellyPlus2PM) Close() {
	dev.shelly.Close()
}

func (dev *ShellyPlus2PM) onConnected(config GetConfigResponse, status GetStatusResponse) {
	dev.info.Name.Set(config.System.Device.Name)
	dev.info.Model.Set("Shelly Plus 2PM")
	dev.info.Manufacturer.Set("Shelly")
	dev.info.SerialNumber.Set(config.System.Device.MAC)
	dev.info.FirmwareVersion.Set(config.System.Device.FirmwareID)
	dev.info.WebUrl.Set("http://" + dev.shelly.ip)
	dev.online.Online.Set(true)
	dev.online.LastSeen.Set(time.Now())

	// Generate inputs/scripts/switches from the config.
	for _, val := range config.Inputs {
		switch val.ID {
		case 0:
			dev.input0.SetAlias(val.Name)
		case 1:
			dev.input1.SetAlias(val.Name)
		default:
			dev.shelly.log.Warnf("config contained unexpected input %+v", val)
		}
	}
	for _, val := range config.Switches {
		switch val.ID {
		case 0:
			dev.relay0.SetAlias(val.Name)
		case 1:
			dev.relay1.SetAlias(val.Name)
		default:
			dev.shelly.log.Warnf("config contained unexpected switch %+v", val)
		}
	}

	// Update the values of inputs/scripts/switches from the status.
	for _, val := range status.Inputs {
		switch val.ID {
		case 0:
			dev.input0.On.Set(val.State)
		case 1:
			dev.input1.On.Set(val.State)
		default:
			dev.shelly.log.Warnf("status contained unexpected input %+v", val)
		}
	}
	for _, val := range status.Switches {
		switch val.ID {
		case 0:
			updateRelay(dev.relay0, val)
		case 1:
			updateRelay(dev.relay1, val)
		default:
			dev.shelly.log.Warnf("status contained unexpected switch %+v", val)
		}
	}
}

func (dev *ShellyPlus2PM) onDisconnected() {
	dev.online.Online.Set(false)
}

func (dev *ShellyPlus2PM) onResponseFrame(ResponseFrame) {

}

func (dev *ShellyPlus2PM) onNotificationFrame(frame NotificationFrame) {
	dev.online.Online.Set(true)
	dev.online.LastSeen.Set(time.Now())

	// Update the values of inputs/scripts/switches from the status.
	for _, val := range frame.NotifyStatus.Inputs {
		if val.ID == 0 {
			dev.input0.On.Set(val.State)
		} else {
			dev.shelly.log.Warnf("status contained unexpected input %+v", val)
		}
	}
	for _, val := range frame.NotifyStatus.Switches {
		switch val.ID {
		case 0:
			updateRelay(dev.relay0, val)
		case 1:
			updateRelay(dev.relay1, val)
		default:
			dev.shelly.log.Warnf("status contained unexpected switch %+v", val)
		}
	}
}

func (dev *ShellyPlus2PM) handleNameAction(val string) {
	dev.shelly.log.Errorf("not changing name to %s", val)
}

func (dev *ShellyPlus2PM) handleRelay0Action(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
	var switchSet SwitchSet

	for _, req := range request.Values {
		switch req.Id {
		case dev.relay0.On.ID():
			if req.GetBool() == nil {
				return services.ErrIncorrectTypeFor(dev.relay0.On)
			}
			switchSet = SwitchSet{
				ID: 0,
				On: req.GetBool().GetValue(),
			}
		}
	}

	err := dev.shelly.RequestSwitchSet(request.ActionId, switchSet)
	if err != nil {
		dev.shelly.log.Errorf("failed to set switch: %s", err)
		return fmt.Errorf("failed to set switch: %w", err)
	}

	return nil
}

func (dev *ShellyPlus2PM) handleRelay1Action(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
	var switchSet SwitchSet

	for _, req := range request.Values {
		switch req.Id {
		case dev.relay1.On.ID():
			if req.GetBool() == nil {
				return services.ErrIncorrectTypeFor(dev.relay0.On)
			}
			switchSet = SwitchSet{
				ID: 0,
				On: req.GetBool().GetValue(),
			}
		}
	}

	err := dev.shelly.RequestSwitchSet(request.ActionId, switchSet)
	if err != nil {
		dev.shelly.log.Errorf("failed to set switch: %s", err)
		return fmt.Errorf("failed to set switch: %w", err)
	}

	return nil
}
