package main

import (
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

type FakeLightbulb struct {
	dev       *devices.Device
	info      *services.Info
	online    *services.Online
	lightbulb *services.Lightbulb
}

func NewFakeLightbulb(id string) *FakeLightbulb {
	dev := &FakeLightbulb{
		dev:       devices.NewDevice(id, clientsapi.Device_DEVICE),
		info:      services.NewInfo(),
		online:    services.NewOnline(),
		lightbulb: services.NewLightbulb("lightbulb"),
	}
	dev.dev.AddService(dev.info, dev.online, dev.lightbulb)

	// Set up the info service.
	dev.info.Name.Set("Fake Lightbulb")
	dev.info.Model.Set("Fake Lightbulb Thing")
	dev.info.Manufacturer.Set("Fake Things Inc")
	dev.online.Online.Set(true)
	dev.online.LastSeen.Set(time.Now())

	// Set up the light service.
	dev.lightbulb.OnAction(dev.handleLightbulbAction)
	dev.lightbulb.On.OnAction(func(val bool) {
		time.Sleep(500 * time.Millisecond)
		log.Infof("on set to %t", val)
		dev.lightbulb.On.Set(val)
	})
	dev.lightbulb.Brightness.OnAction(func(val int64) {
		log.Infof("brightness set to %d%%", val)
		dev.lightbulb.Brightness.Set(val)
	})

	// Set default values.
	dev.lightbulb.On.Set(false)
	dev.lightbulb.Brightness.Set(75)
	dev.lightbulb.ColorTemp.Set(2703)
	dev.lightbulb.Color.Set(32.0, 82.0, 0.0, 0.0)
	dev.lightbulb.Transition.Set(time.Second)

	return dev
}

func (dev *FakeLightbulb) handleLightbulbAction(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
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
