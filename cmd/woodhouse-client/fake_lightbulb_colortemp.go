package main

import (
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

type FakeLightbulbColorTemp struct {
	dev       *devices.Device
	info      *services.Info
	online    *services.Online
	lightbulb *services.Lightbulb
}

func NewFakeLightbulbColorTemp(id string) *FakeLightbulbColorTemp {
	dev := &FakeLightbulbColorTemp{
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
	dev.lightbulb.On.OnAction(func(val bool) {
		time.Sleep(500 * time.Millisecond)
		log.Infof("on set to %t", val)
		dev.lightbulb.On.Set(val)
	})
	dev.lightbulb.Brightness.OnAction(func(val int64) {
		log.Infof("brightness set to %d%%", val)
		dev.lightbulb.Brightness.Set(val)
	})
	dev.lightbulb.ColorTemp.OnAction(func(val int64) {
		log.Infof("color temperature set to %d", val)
		dev.lightbulb.ColorTemp.Set(val)
	})

	// Set default values.
	dev.lightbulb.On.Set(false)
	dev.lightbulb.Brightness.Set(75)
	dev.lightbulb.ColorTemp.Set(454)
	dev.lightbulb.ColorTemp.SetLimits(153, 555, 1)
	dev.lightbulb.Transition.Set(0)

	return dev
}
