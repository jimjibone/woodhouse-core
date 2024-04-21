package main

import (
	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

type SwitchDevice struct {
	dev    *devices.DeviceImpl
	info   *services.Info
	online *services.Online
	light  *services.Lightbulb
}

func NewSwitchDevice(id string) *SwitchDevice {
	dev := &SwitchDevice{
		dev:    devices.NewDevice(id, clientsapi.Device_LIGHTBULB),
		info:   services.NewInfo(),
		online: services.NewOnline(),
		light:  services.NewLightbulb(),
	}
	dev.dev.AddService(dev.info, dev.online, dev.light)

	// Set up the info service.
	dev.info.Name.Set("Fake Light Switch")
	dev.info.Model.Set("Fake Switch")
	dev.info.Manufacturer.Set("Fake Things Inc")
	dev.online.Online.Set(true)

	// Set up the light service.
	dev.light.On.OnAction(func(val bool) {
		log.Infof("on set to %t", val)
		dev.light.On.Set(val)
	})
	dev.light.Brightness.OnAction(func(val int64) {
		log.Infof("brightness set to %d%%", val)
		dev.light.Brightness.Set(val)
	})

	return dev
}
