package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jimjibone/log"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"github.com/jimjibone/woodhouse-core/wh/v1/reactors"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:                 "woodhouse-reactor",
		Usage:                "Runs the woodhouse example reactor",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "store",
				Usage:    "path to config storage location",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "addr",
				Usage:    "woodhouse server address (disables automatic discovery)",
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"v"},
				Usage:   "enable debug logging",
			},
		},
		Before: func(args *cli.Context) error {
			// Setup logging.
			if args.Bool("debug") {
				log.SetOptions(log.WithMinLevel(log.DebugLevel))
			} else {
				log.SetOptions(log.WithMinLevel(log.InfoLevel))
			}
			return nil
		},
		Action: func(args *cli.Context) error {
			// Create the store.
			store := stores.NewFSStore(args.String("store"))

			// Create the client.
			client := reactors.NewClient(store, args.String("addr"))

			go reactorFunc(client)

			err := client.Run()
			if err != nil {
				return fmt.Errorf("failed to run reactor: %s", err)
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}

func reactorFunc(client *reactors.Client) {
	relay := client.GetRelay("fake2")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	relay.OnUpdate(func(changed bool) {
		log.Infof("relay %q changed on: %t, voltage: %.0fV, current: %.0fA, power: %.0fW, temperature: %.0f°C",
			relay.DeviceName(),
			relay.On(),
			relay.Voltage(),
			relay.Current(),
			relay.Power(),
			relay.Temperature(),
		)
	})

	count := 0
	for range ticker.C {
		if relay != nil {
			log.Infof("relay %q online: %t, on: %t, voltage: %.0fV, current: %.0fA, power: %.0fW, temperature: %.0f°C",
				relay.DeviceName(),
				relay.Online(),
				relay.On(),
				relay.Voltage(),
				relay.Current(),
				relay.Power(),
				relay.Temperature(),
			)

			if count%5 == 0 {
				err := relay.SetOn(context.Background(), !relay.On())
				if err != nil {
					log.Errorf("relay set failed: %s", err)
				}
			}
			count++
		} else {
			log.Infof("no relay")
		}
	}
}
