package main

import (
	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/attributes"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

type SwitchDevice struct {
	dev        *devices.DeviceImpl
	info       *services.Info
	generic    *services.Generic
	on         *attributes.Bool
	brightness *attributes.Int
}

func NewSwitchDevice(id string) *SwitchDevice {
	dev := &SwitchDevice{
		dev:        devices.NewDevice(id, clientsapi.Device_GENERIC),
		info:       services.NewInfo(),
		generic:    services.NewGeneric(),
		on:         attributes.NewBool("on", clientsapi.Permissions_PERM_READWRITE, attributes.Required),
		brightness: attributes.NewInt("brightness", clientsapi.Permissions_PERM_READWRITE, attributes.Optional, 0, 100, 1, clientsapi.Unit_UNIT_PERCENTAGE),
	}

	// Set up the info service.
	dev.info.Name.Set("Fake Light Switch")
	dev.info.Model.Set("Fake Switch")
	dev.info.Manufacturer.Set("Fake Things Inc")
	dev.dev.AddService(dev.info)

	// Set up the generic service.
	dev.on.OnAction(func(val bool) {
		log.Infof("on set to %t", val)
		dev.on.Set(val)
	})
	dev.brightness.OnAction(func(val int64) {
		log.Infof("brightness set to %d%%", val)
		dev.brightness.Set(val)
	})
	dev.generic.AddAttribute(dev.on)
	dev.generic.AddAttribute(dev.brightness)
	dev.dev.AddService(dev.generic)

	return dev
}
