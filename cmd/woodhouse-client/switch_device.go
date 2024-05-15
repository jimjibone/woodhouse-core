package main

import (
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

type SwitchDevice struct {
	dev       *devices.DeviceImpl
	info      *services.Info
	online    *services.Online
	lightbulb *services.Lightbulb
}

func NewSwitchDevice(id string) *SwitchDevice {
	dev := &SwitchDevice{
		dev:       devices.NewDevice(id, clientsapi.Device_LIGHTBULB),
		info:      services.NewInfo(),
		online:    services.NewOnline(),
		lightbulb: services.NewLightbulb("lightbulb"),
	}
	dev.dev.AddService(dev.info, dev.online, dev.lightbulb)

	// Set up the info service.
	dev.info.Name.Set("Fake Light Switch")
	dev.info.Model.Set("Fake Switch")
	dev.info.Manufacturer.Set("Fake Things Inc")
	dev.online.Online.Set(true)

	// Set up the light service.
	dev.lightbulb.OnAction(dev.handleLightbulbAction)
	dev.lightbulb.On.OnAction(func(val bool) {
		time.Sleep(2 * time.Second)
		log.Infof("on set to %t", val)
		dev.lightbulb.On.Set(val)
	})
	dev.lightbulb.Brightness.OnAction(func(val int64) {
		log.Infof("brightness set to %d%%", val)
		dev.lightbulb.Brightness.Set(val)
	})

	return dev
}

func (dev *SwitchDevice) handleLightbulbAction(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
	feedback(&clientsapi.ActionResponse{
		ActionId: request.ActionId,
		Status:   clientsapi.ActionResponse_SENT,
	})

	for _, req := range request.Values {
		switch req.Id {
		case dev.lightbulb.On.ID():
			if req.GetBool() == nil {
				return services.ErrIncorrectTypeFor(dev.lightbulb.On)
			}
			dev.lightbulb.On.HandleAction(req.GetBool())

		case dev.lightbulb.Brightness.ID():
			if req.GetInt() == nil {
				return services.ErrIncorrectTypeFor(dev.lightbulb.Brightness)
			}
			dev.lightbulb.Brightness.HandleAction(req.GetInt())
		}
	}

	return nil
}
