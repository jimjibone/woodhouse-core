package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/jimjibone/woodhouse-4/wh/v1/reactors"
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
			client := wh.NewClient(store, args.String("addr"), wh.WithReactors())

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

func reactorFunc(client *wh.Client) {
	relayReactor := reactors.NewDevice("fake2")
	client.AddReactor(relayReactor)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	relayReactor.OnOnlineUpdated(func(srv *reactors.OnlineService) {
		log.Infof("relay %q online changed %t %s", relayReactor.Info().Name(), srv.Online(), srv.LastSeen())
	})
	relayReactor.OnRelayUpdated(func(srv *reactors.RelayService) {
		log.Infof("relay %q relay changed on: %t, voltage: %.0fV, current: %.0fA, power: %.0fW, temperature: %.0f°C", relayReactor.Info().Name(), srv.On(), srv.Voltage(), srv.Current(), srv.Power(), srv.Temperature())
	})

	for range ticker.C {
		log.Infof("relay %q online: %t, on: %t, voltage: %.0fV, current: %.0fA, power: %.0fW, temperature: %.0f°C", relayReactor.Info().Name(), relayReactor.Online().Online(), relayReactor.Relay().On(), relayReactor.Relay().Voltage(), relayReactor.Relay().Current(), relayReactor.Relay().Power(), relayReactor.Relay().Temperature())
	}
}
