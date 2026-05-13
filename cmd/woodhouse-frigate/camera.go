package main

import (
	"context"
	"encoding/json"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-frigate/api"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type FrigateCamera struct {
	log        *log.Context
	serverAddr string
	name       string
	dev        *devices.Device
	motion     *services.Input
	camera     *services.Camera
}

func NewFrigateCamera(serverAddr, name string, client *wh.Client) *FrigateCamera {
	dev := &FrigateCamera{
		log:        log.NewContext(log.DefaultLogger, name, log.DebugLevel),
		serverAddr: serverAddr,
		name:       name,
		dev:        devices.NewDevice("frigate-"+name, clientsapi.Device_DEVICE),
		motion:     services.NewInput("motion"),
		camera:     services.NewCamera(""),
	}
	caser := cases.Title(language.Und, cases.NoLower)
	dev.dev.Info.Name.Set(caser.String(name))
	dev.dev.AddService(dev.motion, dev.camera)
	dev.motion.SetAlias("Motion")
	dev.motion.On.Set(false)
	dev.camera.Image.OnImageRequest(dev.handleCameraImageRequest)
	if err := client.AddDevice(dev.dev); err != nil {
		panic(err)
	}
	return dev
}

func (dev *FrigateCamera) HandleMotion(payload json.RawMessage) {
	switch string(payload) {
	case `"OFF"`:
		dev.log.Infof("motion not detected")
		dev.motion.On.Set(false)
	case `"ON"`:
		dev.log.Infof("motion was detected")
		dev.motion.On.Set(true)
	default:
		dev.log.Errorf("failed to parse motion state of: %s", payload)
	}
}

func (dev *FrigateCamera) handleCameraImageRequest() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	data, err := api.GetLatestImage(ctx, dev.serverAddr, dev.name)
	if err != nil {
		dev.log.Errorf("failed to get latest image: %s", err)
	} else {
		dev.log.Infof("got latest image: %d bytes", len(data))

		// err = os.WriteFile(fmt.Sprintf("frigate-%s-%s.jpeg", dev.name, time.Now().Format("2006-01-02-15-04-05")), data, 0644)
		// if err != nil {
		// 	panic(err)
		// }
	}
	return data, err
}
